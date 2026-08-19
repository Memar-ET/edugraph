"""
Item Response Theory -- 2PL ability estimation (EG-GCKT Milestone 8, spec
sections 7.5 and 16).

Real IRT math (2PL logistic item response function, Newton-Raphson MLE
ability estimation) run on PROVISIONAL item parameters, exactly per spec
section 15's cold-start guidance: "New item -> use provisional item
metadata and avoid over-weighting uncalibrated psychometrics." This
system has ~zero real response volume (documented during planning), so a
genuine joint item+ability EM calibration isn't attempted here -- that's
explicitly Milestone 9 batch-job work, gated on each item accumulating
enough responses (MIN_RESPONSES_FOR_CALIBRATION) to be worth fitting.
Until then, every item's (a, b) parameters are a same-spirit-but-
independent Python re-derivation of the classical-test-theory proxy
`backend/internal/assessment/service/exam_quality.go` already computes in
Go (p-value -> difficulty, upper/lower-half split -> discrimination) --
duplicated rather than shared across the Go/Python boundary, but
producing the same class of honestly-provisional estimate the spec calls
for, with correspondingly low `reliability` on every evidence row this
module writes.

theta (ability) is estimated per (student, topic) from whichever items in
this attempt map to that topic, then squashed through a standard logistic
into [0, 1] so it's comparable with BKT/DINA's mastery_probability
estimates inside the fusion engine -- a documented simplification (theta
is a z-score-like ability scale, not literally a mastery probability),
not a claim of psychometric equivalence.
"""

from __future__ import annotations

import json
import math
from collections import defaultdict
from typing import Optional

import asyncpg
import structlog

from app.db.postgres import get_pool
from app.db.postgres_gckt import get_active_model_snapshot, insert_evidence

logger = structlog.get_logger()

DEFAULT_DISCRIMINATION = 1.0
DEFAULT_DIFFICULTY = 0.0
MIN_RESPONSES_FOR_PROVISIONAL_CALIBRATION = 5
DISCRIMINATION_MIN, DISCRIMINATION_MAX = 0.3, 2.5

MAX_NEWTON_ITERATIONS = 15
THETA_CLAMP = 4.0


async def _provisional_item_params(question_id: str) -> tuple[float, float, int]:
    """Prefers a real calibration if Milestone 9's nightly refit job has
    produced one AND a human has promoted it to 'active' governance
    status (modeling.model_snapshots, model_type='irt_calibration',
    scope=question_id) -- this is what makes that governance loop
    actually matter to the live engine, not just a number nobody reads.
    Falls back to a same-spirit CTT proxy re-derived independently in
    Python from exam_quality.go's bucketDifficulty/discriminationIndex:
    p-value (fraction correct across every student who's answered this
    item) maps to a b-parameter via the standard logit transform under a
    population-ability-zero assumption; an upper/lower-half score-split
    index maps to a provisional a-parameter, clipped to a sane range.
    Falls back further to population-neutral defaults (a=1.0, b=0.0)
    below MIN_RESPONSES_FOR_PROVISIONAL_CALIBRATION -- an uncalibrated
    item should look "average," not confidently anything."""
    active = await get_active_model_snapshot("irt_calibration", scope=question_id)
    if active is not None:
        config = active["config"]
        params = json.loads(config) if isinstance(config, str) else config
        return float(params["a"]), float(params["b"]), int(params.get("n", MIN_RESPONSES_FOR_PROVISIONAL_CALIBRATION))

    pool = await get_pool()
    rows = await pool.fetch(
        """
        SELECT sa.marks_awarded, sa.marks_possible, sa.attempt_id
        FROM assessment.student_answers sa
        WHERE sa.question_id = $1 AND sa.marks_awarded IS NOT NULL AND sa.marks_possible > 0
        """,
        question_id,
    )
    n = len(rows)
    if n < MIN_RESPONSES_FOR_PROVISIONAL_CALIBRATION:
        return DEFAULT_DISCRIMINATION, DEFAULT_DIFFICULTY, n

    ratios = [float(r["marks_awarded"]) / float(r["marks_possible"]) for r in rows]
    p_value = sum(ratios) / n
    p_value = min(0.98, max(0.02, p_value))  # keep the logit finite
    b = -math.log(p_value / (1 - p_value))  # higher p-value (easier) -> lower difficulty

    ordered = sorted(ratios)
    half = max(1, n // 2)
    lower_avg = sum(ordered[:half]) / half
    upper_avg = sum(ordered[-half:]) / half
    raw_discrimination = max(0.0, upper_avg - lower_avg) * 4.0  # scale a 0..1 split-gap into a plausible a-range
    a = min(DISCRIMINATION_MAX, max(DISCRIMINATION_MIN, raw_discrimination or DEFAULT_DISCRIMINATION))

    return a, b, n


def p_correct(theta: float, a: float, b: float) -> float:
    return 1.0 / (1.0 + math.exp(-a * (theta - b)))


def _estimate_theta(items: list[tuple[float, float, bool]]) -> tuple[float, float]:
    """Newton-Raphson MLE for theta given [(a, b, correct), ...]. Returns
    (theta, standard_error_proxy). With very few items (the norm here --
    most topics get 1-3 mapped items per exam) this is numerically
    unstable by nature of the method itself, not a bug -- that instability
    is exactly why the resulting evidence gets a low reliability score
    rather than being trusted at face value."""
    theta = 0.0
    for _ in range(MAX_NEWTON_ITERATIONS):
        info = 0.0
        gradient = 0.0
        for a, b, correct in items:
            p = p_correct(theta, a, b)
            gradient += a * ((1.0 if correct else 0.0) - p)
            info += (a ** 2) * p * (1 - p)
        if info <= 1e-6:
            break
        step = gradient / info
        theta += step
        theta = max(-THETA_CLAMP, min(THETA_CLAMP, theta))
        if abs(step) < 1e-4:
            break

    total_info = sum((a ** 2) * p_correct(theta, a, b) * (1 - p_correct(theta, a, b)) for a, b, _ in items)
    standard_error = 1.0 / math.sqrt(total_info) if total_info > 1e-6 else 2.0
    return theta, standard_error


async def process_attempt(attempt_id: str, events: list[asyncpg.Record]) -> None:
    """One ability estimate per (student, topic) touched by this attempt,
    from whichever of the attempt's items map to that topic. Writes one
    modeling.evidence_log row (provenance='irt') per topic -- reliability
    scales down when item parameters are still population-default
    (uncalibrated) and when very few items informed the estimate, so the
    fusion engine (Milestone 4) correctly treats a shaky single-item
    theta as weak evidence rather than a confident number."""
    by_topic: dict[str, list[tuple[str, bool, str]]] = defaultdict(list)  # topic_id -> [(question_id, correct, event_id)]
    student_id: Optional[str] = None
    for event in events:
        if event["correctness"] is None:
            continue
        student_id = str(event["student_id"])
        for topic_id in event["skill_ids"] or []:
            by_topic[str(topic_id)].append((str(event["item_id"]), bool(event["correctness"]), str(event["id"])))

    if student_id is None or not by_topic:
        return

    written = 0
    for topic_id, item_events in by_topic.items():
        params = []
        min_n = None
        for question_id, correct, _event_id in item_events:
            a, b, n = await _provisional_item_params(question_id)
            params.append((a, b, correct))
            min_n = n if min_n is None else min(min_n, n)

        theta, standard_error = _estimate_theta(params)
        mastery_like = 1.0 / (1.0 + math.exp(-theta))  # squash theta into [0,1], see module docstring

        calibrated = (min_n or 0) >= MIN_RESPONSES_FOR_PROVISIONAL_CALIBRATION
        # Low base reliability either way (spec's Authority column: IRT =
        # assessment evidence, not primary), further discounted when
        # nothing here is actually calibrated yet or the estimate rests
        # on very few items.
        reliability = (0.5 if calibrated else 0.25) * min(1.0, len(item_events) / 3.0)
        uncertainty = min(1.0, standard_error / THETA_CLAMP)

        await insert_evidence(
            student_id,
            topic_id,
            estimate=mastery_like,
            uncertainty=uncertainty,
            sample_size=len(item_events),
            reliability=reliability,
            provenance="irt",
            context={"theta": round(theta, 4), "itemCount": len(item_events), "itemsCalibrated": calibrated},
            source_event_id=item_events[0][2],
        )
        written += 1

    logger.info("irt.process_attempt.done", attempt_id=attempt_id, topics=written)
