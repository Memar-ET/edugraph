-- MCQ options, split out of the question text during parsing (previously
-- concatenated into question_text as raw prose, e.g. "...cell? A. Cell
-- wall B. Cell membrane...") so the frontend can render real labeled
-- choice buttons instead of plain unlabeled A/B/C/D radios.
-- [{"letter": "A", "text": "Cell wall"}, ...]; NULL for non-mcq questions
-- and for rows parsed before this column existed.

ALTER TABLE assessment.questions ADD COLUMN options JSONB;
