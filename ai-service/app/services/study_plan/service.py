"""
Capability 3B: The Study Plan Generator ("Personal Tutor").

Input is the rich 3A output, not a generic failed-topic list: each
students.gap_records row carries the symptom topic, the root-cause topic
the prerequisite walk found, and the explanation of WHY the student
missed that question (gap_analysis's own Gemini/Ollama synthesis).

Ordering (PRD Steps 1-4) is a topological sort over the prerequisite
graph (Neo4j HAS_PREREQUISITE edges first, Postgres topic_prerequisites
as fallback -- same dual-path as the 3A root-cause walk), augmented with
the implied "root cause before symptom" edges 3A already proved per-gap.
So the plan always fixes foundations before the topics built on them,
even when the two stores disagree. Implemented as a hand-rolled Kahn's
algorithm rather than the PRD's specified Neo4j GDS
(gds.dag.topologicalSort) -- GDS isn't available on Neo4j Community
Edition, which is what this deployment actually runs; Kahn's algorithm
over the same edge set produces the same topological guarantee without
the plugin dependency. Step 3 prioritization among topics with no
outstanding prerequisite (formerly severity-only) is now EG-GCKT
Milestone 7's composite next-best-action score -- expected learning gain,
information gain, prerequisite impact, forgetting/recency need,
mandatory-CLO curriculum priority, and a repetition penalty, see
action_ranking.py -- with raw severity kept only as a tie-breaker below
it. Exam-date weighting and the PRD's declared daily-availability input
still aren't read; HOURS_PER_DAY below is a fixed constant.

Step 5 (LLM enrichment) previously didn't exist in this codebase at all
-- see study_plan/llm.py for the actual implementation now wired in
(Gemini -> Ollama, additive to each block's deterministic "why" text,
never blocks generation if both tiers fail).

Output: a day-by-day plan stored in students.study_plans.plan_data
(JSONB).
"""

from __future__ import annotations

from collections import deque
from datetime import date
from typing import Optional

import structlog

from app.db import neo4j as neo4j_db
from app.db import postgres_studyplan as db
from app.services.study_plan.action_ranking import (
    classify_action_types,
    compute_action_scores,
    get_recommendation_policy_snapshot_id,
)
from app.services.study_plan.llm import enrich_days

logger = structlog.get_logger()

HOURS_PER_DAY = 2.0
# A single topic never eats more than this many plan hours, however
# optimistic curriculum.topics.estimated_hours is.
MAX_TOPIC_HOURS = 6.0


async def process_study_plan_job(payload: dict) -> None:
    student_id = payload.get("studentId")
    if not student_id:
        logger.warning("study_plan.bad_payload", payload=payload)
        return
    target_exam_id: Optional[str] = payload.get("targetExamId") or None
    language = payload.get("language") or "en"

    school_id = payload.get("schoolId") or await db.fetch_student_school(student_id)
    if not school_id:
        logger.warning("study_plan.student_not_found", student_id=student_id)
        return

    gaps = await db.fetch_gaps_for_plan(student_id, target_exam_id)

    topics = _aggregate_topics(gaps)
    meta = await db.fetch_topic_meta(list(topics))
    # Drop topics the curriculum no longer knows (deleted units etc.).
    topics = {tid: t for tid, t in topics.items() if tid in meta}

    ordered = await _topological_order(student_id, topics, gaps, meta)
    days, total_hours = _pack_days(ordered, topics, meta)

    exam_days_away = None
    if target_exam_id:
        due = await db.fetch_target_exam_due_date(target_exam_id)
        if due:
            exam_days_away = max((due - date.today()).days, 0)

    # Step 5 (PRD Capability 3B): per-day plain-language goals -- see
    # study_plan/llm.py. Additive: each block's existing "why" text stays
    # as-is and is the only thing the frontend needs if this fails, so a
    # missing/unreachable LLM never blocks plan generation.
    day_goals, day_goals_model = await enrich_days(days, exam_days_away)
    if day_goals:
        for d in days:
            goal = day_goals.get(d["day"])
            if goal:
                d["dayGoal"] = goal

    # EG-GCKT Milestone 7 + checklist section 14: a distinct action type
    # per topic (not just 'practice'), computed before plan_data is
    # assembled so it's actually included in what gets persisted below --
    # and which recommendation_policy version produced this plan (spec
    # section 18).
    is_root_cause = {tid: t["is_root_cause"] for tid, t in topics.items()}
    action_types = await classify_action_types(student_id, ordered, is_root_cause)
    recommendation_policy_snapshot_id = await get_recommendation_policy_snapshot_id()
    for d in days:
        for block in d["blocks"]:
            block["actionType"] = action_types.get(block["topicId"], "practice")

    plan_data = {
        "generatedFrom": {
            "gapCount": len(gaps),
            "targetExamId": target_exam_id,
            "topicCount": len(ordered),
        },
        "days": days,
        "summary": _plan_summary(days, topics, meta),
        "dayGoalsModel": day_goals_model,
    }
    plan_id = await db.insert_study_plan(
        student_id=student_id,
        school_id=school_id,
        target_exam_id=target_exam_id,
        plan_data=plan_data,
        total_days=len(days),
        total_hours=round(total_hours, 1),
        language=language,
    )
    # EG-GCKT Milestone 7: record what was recommended so the NEXT
    # generation's repetition penalty (action_ranking.py) can see it.
    await db.log_recommendations(
        student_id, school_id, ordered, plan_id, action_types, recommendation_policy_snapshot_id
    )

    logger.info(
        "study_plan.generated",
        plan_id=plan_id,
        student_id=student_id,
        topics=len(ordered),
        days=len(days),
        hours=round(total_hours, 1),
        day_goals_model=day_goals_model,
    )


def _aggregate_topics(gaps) -> dict[str, dict]:
    """
    Collapse gap records into per-topic study entries. A topic can appear
    as a symptom (questions missed on it) and/or as a root cause (it
    broke a later topic's chain); root causes inherit the explanation of
    the worst gap they caused, prefixed so the student sees the causal
    link on that day.
    """
    topics: dict[str, dict] = {}

    def entry(tid: str) -> dict:
        return topics.setdefault(
            tid, {"severity": 0.0, "why": None, "is_root_cause": False, "gap_ids": []}
        )

    # Worst gaps first so the first explanation seen per topic is the one
    # from the most severe miss.
    for g in sorted(gaps, key=lambda g: -float(g["severity_score"])):
        tid = str(g["topic_id"])
        e = entry(tid)
        e["severity"] = max(e["severity"], float(g["severity_score"]))
        e["gap_ids"].append(str(g["id"]))
        if e["why"] is None and g["llm_explanation"]:
            e["why"] = g["llm_explanation"]

        if g["root_cause_topic_id"]:
            rid = str(g["root_cause_topic_id"])
            rc = entry(rid)
            rc["is_root_cause"] = True
            rc["severity"] = max(rc["severity"], float(g["severity_score"]))
            rc["gap_ids"].append(str(g["id"]))
            if rc["why"] is None and g["llm_explanation"]:
                rc["why"] = "This is the root cause behind a later gap: " + g["llm_explanation"]
    return topics


async def _topological_order(student_id: str, topics: dict[str, dict], gaps, meta) -> list[str]:
    """Kahn's algorithm over prerequisite edges among the study topics.
    Ready set is popped by EG-GCKT Milestone 7's composite next-best-action
    score (learning gain, information gain, prerequisite impact,
    forgetting need, curriculum priority, repetition penalty -- see
    action_ranking.py), falling back to this attempt's raw severity and
    then curriculum order as tie-breakers, so equally-ranked topics still
    resolve deterministically."""
    ids = list(topics)
    try:
        edges = await neo4j_db.fetch_prerequisite_edges_among(ids)
    except Exception:  # noqa: BLE001 -- graph-service outage (FI-008): degrade to the Postgres mirror rather than fail plan generation
        logger.warning("study_plan.neo4j_unavailable_falling_back", topic_count=len(ids), exc_info=True)
        edges = []
    if not edges:
        edges = await db.fetch_prereq_edges_pg(ids)
    # Implied edges from 3A's per-gap verdicts: the root cause must be
    # studied before the symptom it explains, whether or not a direct
    # graph edge exists between them (the break may be transitive).
    for g in gaps:
        if g["root_cause_topic_id"]:
            t, p = str(g["topic_id"]), str(g["root_cause_topic_id"])
            if t in topics and p in topics and t != p:
                edges.append((t, p))

    # edge (t, p): t requires p => p comes first.
    dependents: dict[str, set[str]] = {tid: set() for tid in ids}
    indegree: dict[str, int] = {tid: 0 for tid in ids}
    seen: set[tuple[str, str]] = set()
    for t, p in edges:
        if t not in indegree or p not in indegree or (t, p) in seen:
            continue
        seen.add((t, p))
        dependents[p].add(t)
        indegree[t] += 1

    action_scores = await compute_action_scores(student_id, ids)

    def rank(tid: str) -> tuple:
        m = meta[tid]
        return (
            -action_scores.get(tid, 0.0),
            -topics[tid]["severity"],
            m["grade_level"],
            m["unit_number"],
            m["sequence_order"],
        )

    ready = sorted((tid for tid in ids if indegree[tid] == 0), key=rank)
    queue = deque(ready)
    ordered: list[str] = []
    while queue:
        tid = queue.popleft()
        ordered.append(tid)
        unlocked = []
        for dep in dependents[tid]:
            indegree[dep] -= 1
            if indegree[dep] == 0:
                unlocked.append(dep)
        for dep in sorted(unlocked, key=rank):
            queue.append(dep)

    # A cycle can't happen through the API (write path rejects them), but
    # hand-inserted data must not silently drop topics -- append leftovers.
    if len(ordered) < len(ids):
        leftovers = sorted((t for t in ids if t not in set(ordered)), key=rank)
        logger.warning("study_plan.prerequisite_cycle_detected", leftovers=len(leftovers))
        ordered.extend(leftovers)
    return ordered


def _pack_days(ordered: list[str], topics: dict[str, dict], meta) -> tuple[list[dict], float]:
    """Greedy day packing at HOURS_PER_DAY: small topics share a day, a
    big topic spans several (blocks after the first are marked
    "continued"). Every block repeats the topic's "why" -- per the spec,
    each day must tell the student why they're studying that topic."""
    days: list[dict] = []
    current: list[dict] = []
    room = HOURS_PER_DAY
    total_hours = 0.0

    def close_day() -> None:
        nonlocal current, room
        if current:
            days.append({"day": len(days) + 1, "blocks": current})
        current, room = [], HOURS_PER_DAY

    for tid in ordered:
        m = meta[tid]
        t = topics[tid]
        remaining = min(float(m["estimated_hours"] or 2.0), MAX_TOPIC_HOURS)
        total_hours += remaining
        why = t["why"] or (
            "You lost marks on questions testing this topic -- review it to close the gap."
            if not t["is_root_cause"]
            else "This foundational topic is holding back topics that build on it."
        )
        continued = False
        while remaining > 0:
            if room <= 0:
                close_day()
            hours = round(min(remaining, room), 2)
            current.append(
                {
                    "topicId": tid,
                    "title": m["title_en"],
                    "gradeLevel": m["grade_level"],
                    "unitNumber": m["unit_number"],
                    "hours": hours,
                    "why": why,
                    "isRootCause": t["is_root_cause"],
                    "severity": round(t["severity"], 2),
                    "keyConcepts": list(m["key_concepts"] or []),
                    "continued": continued,
                    "gapIds": t["gap_ids"],
                }
            )
            remaining = round(remaining - hours, 2)
            room = round(room - hours, 2)
            continued = True
    close_day()
    return days, total_hours


def _plan_summary(days: list[dict], topics: dict[str, dict], meta) -> str:
    if not days:
        return "No unresolved knowledge gaps -- nothing to schedule. Keep up the good work!"
    root_causes = [meta[tid]["title_en"] for tid, t in topics.items() if t["is_root_cause"] and tid in meta]
    parts = [f"{len(days)}-day plan covering {len(topics)} topic(s), foundations first."]
    if root_causes:
        parts.append("Root causes scheduled before the topics they hold back: " + ", ".join(sorted(root_causes)) + ".")
    return " ".join(parts)
