"""FastAPI route for AI-generated national curriculum insights (feature 6.2)."""
from fastapi import APIRouter, HTTPException
from pydantic import BaseModel

from app.services.insights.service import generate_national_insights

router = APIRouter(prefix="/api/v1/insights", tags=["insights"])


class NationalInsightsRequest(BaseModel):
    total_regions: int = 0
    total_schools: int = 0
    total_students: int = 0
    total_teachers: int = 0


@router.post("/national")
async def national_insights(req: NationalInsightsRequest):
    try:
        result = await generate_national_insights(req.model_dump())
        return result
    except Exception as exc:
        raise HTTPException(status_code=500, detail=str(exc)) from exc
