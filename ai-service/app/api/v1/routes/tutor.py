"""
POST /api/v1/tutor/ask (Capability 3C) -- called by the Go API's tutor
proxy (internal/assessment tutor endpoint), which owns auth/role checks;
this service trusts the studentId it is handed, same trust model as the
Redis queue payloads.
"""

from __future__ import annotations

from fastapi import APIRouter, HTTPException
from pydantic import BaseModel, Field

from app.services.tutor.service import TutorUnavailable, ask

router = APIRouter(prefix="/api/v1/tutor", tags=["tutor"])


class AskRequest(BaseModel):
    student_id: str = Field(alias="studentId")
    question: str = Field(min_length=3, max_length=2000)
    language: str = "en"

    model_config = {"populate_by_name": True}


@router.post("/ask")
async def ask_tutor(req: AskRequest) -> dict:
    try:
        return await ask(req.student_id, req.question, req.language)
    except TutorUnavailable as exc:
        raise HTTPException(status_code=503, detail=str(exc)) from exc
