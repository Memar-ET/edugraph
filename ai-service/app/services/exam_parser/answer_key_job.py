"""
Processes a separately-uploaded answer-key document (Capability 2C).

Real "Extraction-Ready" exams turned out to ship the Q#/CLO/Bloom/Marks/
Correct-Answer table as its own standalone document (a "Test 1 - Answer
Key.docx"), not embedded in the student-facing exam paper -- confirmed
against real files during this session. answer_key.py's table extraction
is format-agnostic (it just looks for a table with the right header
shape), so it's reused as-is for this standalone-document case; only the
orchestration differs from exam_parser/service.py's main parse job: there
are no questions to create here, just an existing exam's already-parsed
questions to update by sequence_number.
"""

from __future__ import annotations

import structlog

from app.db import postgres, postgres_assessment
from app.services.exam_parser import answer_key

logger = structlog.get_logger()

PDF_MAGIC = b"%PDF-"
ZIP_MAGICS = (b"PK\x03\x04", b"PK\x05\x06", b"PK\x07\x08")


async def process_answer_key_job(exam_id: str, file_ref: str) -> None:
    log = logger.bind(exam_id=exam_id)

    school_id = await postgres_assessment.fetch_exam_school(exam_id)
    if school_id is None:
        log.warning("answer_key_parse.exam_not_found")
        return

    try:
        file_bytes = await postgres.fetch_file_bytes(file_ref)

        if file_bytes.startswith(PDF_MAGIC):
            answer_map = answer_key.extract_pdf_answer_key(file_bytes)
        elif file_bytes.startswith(ZIP_MAGICS):
            answer_map = answer_key.extract_docx_answer_key(file_bytes)
        else:
            raise ValueError("file content is not a valid PDF or DOCX document")

        if not answer_map:
            log.warning("answer_key_parse.no_table_found")
            return

        matched = await postgres_assessment.apply_answer_key(exam_id, answer_map)
        log.info("answer_key_parse.applied", found=len(answer_map), matched=matched)

    except Exception:  # noqa: BLE001 -- log and drop; this isn't the exam's
        # primary parse job, a failure here shouldn't touch exam status.
        log.exception("answer_key_parse.failed")
