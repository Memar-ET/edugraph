"""
Postgres access for career matching (Capability 3D).

Reads career_paths_v010, not the bare career_paths name the table was
originally created with -- V011__updated_curriculum.sql renamed
(archived, not dropped) it to make room for a newer careers.careers/
careers.career_topic_requirements schema (topic-level requirements +
importance weighting, closer to the PRD's Neo4j-native design). That
newer schema is completely unused anywhere in the codebase today
(confirmed by grep, not assumed) and has no create/curation UI for
career_topic_requirements either, so this stays on the archived-but-
still-live career_paths_v010 table rather than migrating as part of
this fix -- see backend/internal/career/repository/repository.go's
matching comment for the full story (this was the deeper reason
"Generate Career Matches" was broken: every query against the bare
career_paths name failed outright with "relation does not exist").
"""

from __future__ import annotations

import json

from app.db.postgres import get_pool


async def fetch_career_paths() -> list[dict]:
    """required_subjects comes back JSONB-as-text from asyncpg (no JSON
    codec registered on this pool -- see postgres.py) so it's decoded
    here rather than leaving every caller to remember to."""
    pool = await get_pool()
    rows = await pool.fetch("SELECT id, title, required_subjects FROM career_paths_v010")
    return [
        {"id": str(r["id"]), "title": r["title"], "required_subjects": json.loads(r["required_subjects"] or "[]")}
        for r in rows
    ]
