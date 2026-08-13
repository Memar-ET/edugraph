"""
POST /api/v1/career/match (Capability 3D) -- called by the Go API's
career service (pkg/ai/ai.go's MatchCareers), which owns fetching the
student's per-subject averages and persisting the returned scores; this
service just does the scoring. Trust model matches tutor.py/embeddings.py:
no auth here, the Go API is the only caller.
"""

from __future__ import annotations

from fastapi import APIRouter
from pydantic import BaseModel

from app.services.career_matcher.service import match_careers

router = APIRouter(prefix="/api/v1/career", tags=["career"])


class CareerMatchRequest(BaseModel):
    student_id: str
    grades: dict[str, float]


class CareerMatchResult(BaseModel):
    career_path_id: str
    score: float


@router.post("/match", response_model=list[CareerMatchResult])
async def career_match(req: CareerMatchRequest) -> list[dict]:
    return await match_careers(req.student_id, req.grades)
