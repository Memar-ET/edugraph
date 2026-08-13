"""
Capability 3D: Career Recommendation Engine.

The PRD's design is Neo4j-native: (:Career)-[:REQUIRES]->(:Topic) edges,
readiness scored as alreadyMastered/totalRequired against a student's
:MASTERED edges. None of that exists in the live graph today -- no
(:Career) nodes, no :MASTERED edges (student mastery lives entirely in
Postgres students.mastery_records, never mirrored to Neo4j as graph
edges) -- and building it would mean seeding a whole new graph layer,
well beyond "fix the broken button" scope.

What's actually wired end-to-end today (Go's
internal/career/service.go/repository.go) is coarser but real: career
paths declare required_subjects as free-text subject-family strings
(e.g. "PHY", "MATH" -- see career_paths_v010.required_subjects, JSONB),
and the Go side computes the student's average exam percentage per
subject family. This service scores each career path as the mean of the
student's grades across whichever required subjects they actually have
exam data for -- if a career requires a subject the student has never
been examined in, that subject simply doesn't contribute (no evidence
isn't scored as failure, same principle gap_analysis's cold-start
handling uses for prerequisites); if NONE of a career's required
subjects have data, the score is 0.0 -- zero demonstrated readiness in
every required area is a real, honest signal, not a missing-data
artifact to hide.

career_paths_v010.embedding (vector(1024), V007) is a second unused
design this migration anticipated -- semantic similarity between a
career's description and something to compare it against. Left
untouched: there is no existing "student profile" text or embedding to
compare it with anywhere in this system, and inventing one wasn't part
of this fix.

Subject-family matching is case-insensitive, trimmed exact match --
required_subjects is free text a ministry_admin types with no dropdown
against real curriculum.subjects (see CareerPathsPage.tsx's own
placeholder, "PHY, MATH"), and the Go side already normalizes its side
to upper-cased, grade-digit-stripped subject codes (e.g. "BIO7" ->
"BIO"), so this is the matching contract both sides need to honor.
"""

from __future__ import annotations

import structlog

from app.db.postgres_career import fetch_career_paths

logger = structlog.get_logger()


def _normalize(subject: str) -> str:
    return subject.strip().upper()


async def match_careers(student_id: str, grades: dict[str, float]) -> list[dict]:
    """grades: {subject_family: avg_percentage_0_to_100}, already computed
    and normalized by the Go caller. Returns [{"career_path_id", "score"}],
    score 0..1 (career_matches.match_score is NUMERIC(5,4))."""
    normalized_grades = {_normalize(k): v for k, v in grades.items()}
    paths = await fetch_career_paths()

    results = []
    for path in paths:
        required = [_normalize(s) for s in path["required_subjects"]]
        matched_scores = [normalized_grades[s] for s in required if s in normalized_grades]
        score = (sum(matched_scores) / len(matched_scores) / 100.0) if matched_scores else 0.0
        results.append({"career_path_id": path["id"], "score": round(score, 4)})

    logger.info("career_matcher.matched", student_id=student_id, career_paths=len(paths))
    results.sort(key=lambda r: -r["score"])
    return results
