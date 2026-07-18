-- Capability 2B: The AI Exam Validation Report.
--
-- unit_numbers scopes a Unit Test/Midterm to specific curriculum.units
-- (NULL/empty means "all units" for a final_exam, or "scope not resolved"
-- for a unit_test/midterm -- callers must treat that as "skip the
-- scope-narrowed coverage check" rather than assume the whole book).
-- Derived the same two-stage way subject/grade/scope already are (Go title
-- regex at upload as a fallback, Python metadata-table extraction as the
-- authoritative correction -- see service/title_parser.go and
-- ai-service/app/services/exam_parser/metadata.py).
--
-- validation_report holds the actual structured 5-part report, mirroring
-- curriculum.upload_jobs.parsed_structure's in-DB JSONB pattern.
-- validation_report_s3 (existing, unused TEXT column) is left untouched --
-- its "_s3" naming suggests an external-file design that was never
-- implemented; the report lives in Postgres directly instead.

ALTER TABLE assessment.exams
    ADD COLUMN unit_numbers INT[],
    ADD COLUMN validation_report JSONB;
