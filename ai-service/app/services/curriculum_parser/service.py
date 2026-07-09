"""
Orchestrates one curriculum-parsing job end to end:

  1. Fetch the job row (subject/grade/file reference) from Postgres.
  2. Download the raw file bytes (BYTEA in dev, would be S3 in prod --
     see app/db/postgres.py:fetch_file_bytes).
  3. Run the appropriate extractor based on the file's extension.
  4. Save the resulting JSON tree to upload_jobs.parsed_structure and
     flip status to 'parsed' -- or 'failed' with the error message.
"""

from __future__ import annotations

import structlog

from app.db import postgres
from app.services.curriculum_parser import docx_extractor, extractor

logger = structlog.get_logger()

SUPPORTED_EXTENSIONS = (".pdf", ".docx")


async def process_job(job_id: str) -> None:
    log = logger.bind(job_id=job_id)

    job = await postgres.fetch_job(job_id)
    if job is None:
        log.warning("curriculum_parse.job_not_found")
        return

    if job["status"] not in ("pending", "failed"):
        # Already parsed/parsing/reviewed -- don't reprocess (e.g. if the
        # same job id somehow ends up on the queue twice).
        log.info("curriculum_parse.skip_already_processed", status=job["status"])
        return

    await postgres.mark_parsing(job_id)

    file_name = job["file_name"] or ""
    lower_name = file_name.lower()

    try:
        file_bytes = await postgres.fetch_file_bytes(job["file_s3_key"])

        if lower_name.endswith(".pdf"):
            structure = extractor.extract_structure(
                file_bytes,
                subject_code=job["subject_code"],
                grade_level=job["grade_level"],
                academic_year=job["academic_year"],
            )
        elif lower_name.endswith(".docx"):
            structure = docx_extractor.extract_structure(
                file_bytes,
                subject_code=job["subject_code"],
                grade_level=job["grade_level"],
                academic_year=job["academic_year"],
            )
        else:
            raise ValueError(
                f"unsupported file type for '{file_name}'; expected one of {SUPPORTED_EXTENSIONS}"
            )

        await postgres.save_parsed_structure(job_id, structure)
        log.info(
            "curriculum_parse.parsed",
            strategy=structure.get("extractionStrategy"),
            unit_count=len(structure.get("units", [])),
        )

    except Exception as exc:  # noqa: BLE001 -- we want to persist *any* failure
        log.exception("curriculum_parse.failed")
        await postgres.mark_failed(job_id, f"{type(exc).__name__}: {exc}")
