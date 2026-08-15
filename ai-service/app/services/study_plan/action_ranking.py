"""
Next-Best-Learning-Action ranking (EG-GCKT Milestone 7, spec section 13).

Replaces the original study-plan generator's severity-only tie-break
(service.py's old `rank()`) with a composite score combining: expected
learning gain, information gain, prerequisite impact, forgetting/recency
need, curriculum priority (mandatory-CLO weighting), difficulty fit, and
a repetition penalty. Confidence-improvement (one more factor the spec
lists) is deliberately NOT separately implemented: in practice it's the
same signal as information_gain (both ask "how much would resolving this
topic's uncertainty help") -- inventing a second, barely-distinct number
for it would manufacture precision this system doesn't have.

Difficulty fit uses Milestone 8's IRT-calibrated item difficulty/
discrimination where available (checklist follow-up -- the original pass
skipped this, deferring to "once Milestone 8 exists"; it now does) and
falls back to a neutral score for topics with no calibrated items yet,
per the same cold-start honesty principle as everywhere else in this
system.

classify_action_types (checklist section 14) assigns each topic a
DISTINCT candidate action type -- practice/diagnostic/concept_
explanation/alternative_representation/prerequisite_review/
spaced_review/teacher_escalation -- instead of the original pass's
"everything is 'practice'" placeholder.

This governs ORDER AMONG TOPICS ALREADY UNBLOCKED at a given point in the
topological sort (service.py's Kahn's-algorithm dependency structure is
untouched) -- it is a tie-break policy, not a substitute for prerequisite
ordering.
"""

from __future__ import annotations

import math
from typing import Optional

from app.db import postgres_studyplan as db
from app.db.postgres_gckt import (
    fetch_blocked_topics,
    fetch_skill_states_bulk,
    get_active_model_snapshot,
    insert_model_snapshot,
)
from app.services.gap_analysis.root_cause import WEAK_THRESHOLD, downstream_impact
from app.services.knowledge_tracing.irt import p_correct

# Linear-combination weights -- a documented, defensible starting point,
# not values fit to any outcome data (there isn't any yet; Milestone 10's
# evaluation harness is where these would eventually get calibrated
# against real learning-gain measurements).
WEIGHT_LEARNING_GAIN = 0.25
WEIGHT_INFORMATION_GAIN = 0.15
WEIGHT_PREREQUISITE_IMPACT = 0.15
WEIGHT_FORGETTING_NEED = 0.15
WEIGHT_CURRICULUM_PRIORITY = 0.10
WEIGHT_DIFFICULTY_FIT = 0.10
WEIGHT_REPETITION_PENALTY = 0.10  # subtracted, not added

# A topic recommended this many times in the lookback window without
# resolving is treated as maximally repetition-penalized -- diminishing
# returns on suggesting the exact same thing yet again.
REPETITION_SATURATION = 4

# A topic the student is currently blocked on (checklist section 13's
# Blocked-Learning & Recovery System has an active recovery_attempts row
# for it) is heavily discounted here -- re-suggesting direct practice on
# something already known to trigger repeated failure is exactly the
# "cycling through ineffective interventions" the spec warns against.
# recovery.py is what should be surfacing the alternative route topic
# instead; this penalty just keeps the blocked topic itself out of the
# way while that's in progress.
BLOCKED_TOPIC_PENALTY = 0.5

# The "zone of proximal development" target success probability a
# well-fit item should produce -- not too easy, not too hard.
TARGET_SUCCESS_PROBABILITY = 0.6


def _difficulty_fit(mastery: Optional[float], avg_item: Optional[tuple[float, float]]) -> float:
    """1.0 when a typical calibrated item for this topic would put the
    student's predicted success right at TARGET_SUCCESS_PROBABILITY,
    falling off as predicted success drifts toward "trivially easy" or
    "hopelessly hard." Neutral (0.5) when there's no calibrated item to
    reason about yet -- never fabricates a fit score from nothing."""
    if avg_item is None or mastery is None:
        return 0.5
    difficulty, discrimination = avg_item
    theta = math.log(max(1e-6, mastery) / max(1e-6, 1 - mastery))  # inverse of the sigmoid squash irt.py uses
    predicted_success = p_correct(theta, discrimination, difficulty)
    return max(0.0, 1.0 - abs(predicted_success - TARGET_SUCCESS_PROBABILITY) / TARGET_SUCCESS_PROBABILITY)


async def get_recommendation_policy_snapshot_id() -> Optional[str]:
    """Ensures a 'recommendation_policy' modeling.model_snapshots row
    exists (spec section 18: "version recommendation policy") -- same
    governance pattern as fusion.py's fusion_policy snapshot."""
    snap = await get_active_model_snapshot("recommendation_policy", scope=None)
    if snap is not None:
        return str(snap["id"])
    config = {
        "weightLearningGain": WEIGHT_LEARNING_GAIN,
        "weightInformationGain": WEIGHT_INFORMATION_GAIN,
        "weightPrerequisiteImpact": WEIGHT_PREREQUISITE_IMPACT,
        "weightForgettingNeed": WEIGHT_FORGETTING_NEED,
        "weightCurriculumPriority": WEIGHT_CURRICULUM_PRIORITY,
        "weightDifficultyFit": WEIGHT_DIFFICULTY_FIT,
        "weightRepetitionPenalty": WEIGHT_REPETITION_PENALTY,
        "blockedTopicPenalty": BLOCKED_TOPIC_PENALTY,
        "targetSuccessProbability": TARGET_SUCCESS_PROBABILITY,
    }
    return await insert_model_snapshot(
        "recommendation_policy",
        config,
        scope=None,
        status="active",
        notes="Initial next-best-action weighting configuration -- hand-set, not fit to any outcome data.",
    )


async def compute_action_scores(student_id: str, topic_ids: list[str]) -> dict[str, float]:
    """One composite next-best-action score per topic -- higher means
    "schedule sooner" among topics with no outstanding prerequisite in
    the current plan."""
    if not topic_ids:
        return {}

    states = await fetch_skill_states_bulk(student_id, topic_ids)
    repetition = await db.fetch_repetition_counts(student_id, topic_ids)
    mandatory_fraction = await db.fetch_mandatory_clo_fraction(topic_ids)
    blocked = await fetch_blocked_topics(student_id, topic_ids)
    avg_item_difficulty = await db.fetch_avg_item_difficulty(topic_ids)

    scores: dict[str, float] = {}
    for tid in topic_ids:
        state = states.get(tid)
        mastery = state["mastery_probability"] if state else None
        uncertainty = state["uncertainty"] if state else None
        forgetting_risk = state["forgetting_risk"] if state else None

        weakness = (max(0.0, WEAK_THRESHOLD - mastery) / WEAK_THRESHOLD) if mastery is not None else 0.5
        impact = await downstream_impact(tid)
        learning_gain = impact * weakness

        # A topic with no fused estimate at all is maximally uncertain by
        # definition (cold start) -- resolving that uncertainty has
        # genuine diagnostic value, so this deliberately does NOT default
        # to 0 for unseen topics.
        information_gain = uncertainty if uncertainty is not None else 0.7

        forgetting_need = forgetting_risk if forgetting_risk is not None else 0.0
        curriculum_priority = mandatory_fraction.get(tid, 0.5)
        repetition_penalty = min(1.0, repetition.get(tid, 0) / REPETITION_SATURATION)
        difficulty_fit = _difficulty_fit(mastery, avg_item_difficulty.get(tid))

        scores[tid] = (
            WEIGHT_LEARNING_GAIN * learning_gain
            + WEIGHT_INFORMATION_GAIN * information_gain
            + WEIGHT_PREREQUISITE_IMPACT * impact
            + WEIGHT_FORGETTING_NEED * forgetting_need
            + WEIGHT_CURRICULUM_PRIORITY * curriculum_priority
            + WEIGHT_DIFFICULTY_FIT * difficulty_fit
            - WEIGHT_REPETITION_PENALTY * repetition_penalty
            - (BLOCKED_TOPIC_PENALTY if tid in blocked else 0.0)
        )

    return scores


# Recovery attempts failed at least this many times for the same blocked
# topic escalates to a human rather than trying yet another automated
# route (spec section 13's repetition-penalty principle, applied at the
# point where automation should stop and a teacher should step in).
TEACHER_ESCALATION_FAILED_ROUTES = 2

# A skill_state with fewer observations than this is still "cold" enough
# that introducing the concept is more appropriate than jumping straight
# to practice or a diagnostic probe.
COLD_START_EVIDENCE_THRESHOLD = 1

# High forgetting risk on an already-adequately-mastered topic calls for
# spaced review rather than new practice.
SPACED_REVIEW_FORGETTING_THRESHOLD = 0.5


async def classify_action_types(
    student_id: str, topic_ids: list[str], is_root_cause: Optional[dict[str, bool]] = None
) -> dict[str, str]:
    """One candidate action TYPE per topic (spec section 14's action
    list), distinct from compute_action_scores' ordering. is_root_cause
    (optional): topic_id -> True for topics service.py already knows are
    a root cause of some other gap (from gap_records) -- when omitted,
    that signal just isn't available and the classifier falls through to
    its other rules, never fabricated."""
    if not topic_ids:
        return {}

    is_root_cause = is_root_cause or {}
    states = await fetch_skill_states_bulk(student_id, topic_ids)
    blocked = await fetch_blocked_topics(student_id, topic_ids)
    failed_routes = await db.fetch_failed_recovery_route_counts(student_id, topic_ids)

    action_types: dict[str, str] = {}
    for tid in topic_ids:
        if failed_routes.get(tid, 0) >= TEACHER_ESCALATION_FAILED_ROUTES:
            action_types[tid] = "teacher_escalation"
            continue
        if tid in blocked:
            # A blocked topic itself shouldn't be recommended for direct
            # practice (see BLOCKED_TOPIC_PENALTY) -- recovery.py is
            # already routing the student to an alternative topic, so if
            # this one still surfaces here it's as context, not a
            # practice target.
            action_types[tid] = "alternative_representation"
            continue

        state = states.get(tid)
        evidence_count = state["evidence_count"] if state else 0
        mastery = state["mastery_probability"] if state else None
        forgetting_risk = state["forgetting_risk"] if state else None

        if evidence_count <= COLD_START_EVIDENCE_THRESHOLD:
            action_types[tid] = "concept_explanation"
        elif is_root_cause.get(tid):
            action_types[tid] = "prerequisite_review"
        elif forgetting_risk is not None and forgetting_risk >= SPACED_REVIEW_FORGETTING_THRESHOLD and mastery is not None and mastery >= WEAK_THRESHOLD:
            action_types[tid] = "spaced_review"
        elif state is not None and (state["uncertainty"] or 0) > 0.5:
            action_types[tid] = "diagnostic"
        else:
            action_types[tid] = "practice"

    return action_types
