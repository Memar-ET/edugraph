"""
Orchestrates one exam-parsing job end to end (Capability 2A):

  1. Fetch the exam row (school/file reference) from Postgres.
  2. Download the raw file bytes (BYTEA in dev -- same fetch_file_bytes
     curriculum uses, it's storage-provider-agnostic).
  3. Detect PDF vs DOCX by magic bytes (assessment.exams has no stored
     file_name/extension column, unlike curriculum.upload_jobs).
  4. Look for a Subject/Grade Level/Exam Type/Total Marks metadata table
     (metadata.py) -- when it's present and fully resolves, it corrects the
     exam row (the title typed at upload time was only ever a fallback
     guess for the NOT NULL columns; the document itself is authoritative).
  5. Run the matching question extractor.
  6. For each question without an explicit [CLO: ...] annotation, match it
     to a curriculum CLO -- the local vector-embedding matcher
     (vector_matcher.py, Capability 2.1) first when the target subject has
     embedded CLOs, then the Gemini-backed matcher (clo_matcher_llm.py)
     when GEMINI_API_KEY is configured, else the plain keyword-overlap
     matcher (clo_matcher.py).
  7. Save the resulting question rows to assessment.questions and flip
     status to 'draft' -- or 'failed' with the error message.
"""

from __future__ import annotations

import structlog

from app.db import postgres, postgres_assessment
from app.services.exam_parser import answer_key, clo_matcher, clo_matcher_llm, docx_extractor, extractor, metadata, vector_matcher

logger = structlog.get_logger()

PDF_MAGIC = b"%PDF-"
ZIP_MAGICS = (b"PK\x03\x04", b"PK\x05\x06", b"PK\x07\x08")  # DOCX is an OOXML/zip container


async def process_exam_job(exam_id: str) -> None:
    log = logger.bind(exam_id=exam_id)

    exam = await postgres_assessment.fetch_exam(exam_id)
    if exam is None:
        log.warning("exam_parse.exam_not_found")
        return

    if exam["status"] not in ("pending", "failed"):
        log.info("exam_parse.skip_already_processed", status=exam["status"])
        return

    await postgres_assessment.mark_exam_parsing(exam_id)

    try:
        file_bytes = await postgres.fetch_file_bytes(exam["file_s3_key"])
        is_pdf = file_bytes.startswith(PDF_MAGIC)
        is_docx = file_bytes.startswith(ZIP_MAGICS)
        if not (is_pdf or is_docx):
            raise ValueError("file content is not a valid PDF or DOCX document")

        subject_code = exam["subject_code"]

        raw_meta = (
            metadata.extract_pdf_metadata_table(file_bytes) if is_pdf else metadata.extract_docx_metadata_table(file_bytes)
        )
        resolved = metadata.parse_metadata_table(raw_meta)
        if resolved:
            matched_subject = await postgres_assessment.match_subject_code(
                resolved["subjectHint"], resolved["gradeLevel"]
            )
            if matched_subject:
                await postgres_assessment.update_exam_metadata(
                    exam_id, matched_subject, resolved["gradeLevel"], resolved["examScope"], resolved["totalMarks"],
                    unit_numbers=resolved.get("unitNumbers"),
                )
                subject_code = matched_subject
                log.info("exam_parse.metadata_table_applied", **resolved, subjectCode=matched_subject)
            else:
                log.warning("exam_parse.metadata_table_subject_unmatched", subject_hint=resolved["subjectHint"])

        questions = extractor.extract_questions(file_bytes) if is_pdf else docx_extractor.extract_questions(file_bytes)
        if not questions:
            raise ValueError("no numbered questions were found in the document")

        # Only present in teacher-authored copies (an "Answer Key & Marking
        # Guide" table) -- student-facing copies never have this, {} is the
        # normal case, not an error. Only feeds MCQ auto-grading (2C);
        # non-MCQ questions stay manually graded regardless.
        answer_key_map = (
            answer_key.extract_pdf_answer_key(file_bytes) if is_pdf else answer_key.extract_docx_answer_key(file_bytes)
        )
        for q in questions:
            if q["questionType"] == "mcq" and q["sequenceNumber"] in answer_key_map:
                q["answerKey"] = {"correctOption": answer_key_map[q["sequenceNumber"]]}

        # clo_code is a real FK (curriculum.clos.code) -- a typo, or a code
        # from curriculum not yet uploaded/approved, would fail the whole
        # insert batch, so drop unresolvable codes rather than let one bad
        # value take down every question in the exam.
        clo_rows = await postgres_assessment.fetch_clos_for_subject(subject_code)
        clo_pairs = [(r["code"], r["description_en"]) for r in clo_rows]
        valid_clo_codes = {code for code, _ in clo_pairs}

        for q in questions:
            if q.get("cloCode"):
                # Explicit annotation from the document -- verify it's real
                # rather than guess a replacement.
                if q["cloCode"] not in valid_clo_codes:
                    log.warning("exam_parse.unknown_clo_code", clo_code=q["cloCode"], sequence_number=q["sequenceNumber"])
                    q["cloCode"] = None
                continue

            # Capability 2.1: real semantic (vector embedding) matching
            # first -- runs locally/offline, no per-question cloud call.
            # Falls through silently (see vector_matcher.match_clo_vector's
            # docstring) to the Gemini matcher, then the keyword matcher,
            # exactly preserving the two matchers' prior behavior for
            # subjects that don't have CLO embeddings yet.
            code, score, _classification = await vector_matcher.match_clo_vector(q["questionText"], subject_code)
            if code:
                q["cloCode"], q["cloAlignScore"], q["cloAlignMethod"] = code, score, "embedding_auto"
                continue

            code, score = await clo_matcher_llm.match_clo_llm(q["questionText"], clo_pairs)
            if code:
                q["cloCode"], q["cloAlignScore"], q["cloAlignMethod"] = code, score, "llm_verified"
                continue

            code, score = clo_matcher.match_clo(q["questionText"], clo_pairs)
            if code:
                q["cloCode"], q["cloAlignScore"], q["cloAlignMethod"] = code, score, "ai_draft"

        await postgres_assessment.save_exam_questions(exam_id, exam["school_id"], questions)
        matched_count = sum(1 for q in questions if q.get("cloCode"))
        vector_matched_count = sum(1 for q in questions if q.get("cloAlignMethod") == "embedding_auto")
        llm_matched_count = sum(1 for q in questions if q.get("cloAlignMethod") == "llm_verified")
        answer_key_count = sum(1 for q in questions if q.get("answerKey"))
        log.info(
            "exam_parse.parsed",
            question_count=len(questions),
            clo_matched=matched_count,
            vector_matched=vector_matched_count,
            llm_matched=llm_matched_count,
            answer_key_matched=answer_key_count,
        )

    except Exception as exc:  # noqa: BLE001 -- we want to persist *any* failure
        log.exception("exam_parse.failed")
        await postgres_assessment.mark_exam_failed(exam_id, f"{type(exc).__name__}: {exc}")
