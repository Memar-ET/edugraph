"""
Core "brain work" of exam PDF parsing (Capability 2A).

Unlike curriculum's PDF path (heading/font driven), exam papers are mostly
flat body text: a "Section A"/"Part B" section label, then a flat run of
numbered questions ("Q1.", "1.", ...). So this walks the extracted text line
by line rather than clustering headings by font size.

Real "Extraction-Ready" exam documents (the format this was designed
against, same spirit as curriculum's CLO documents) annotate each question
with a bracketed line right after it:

    Q1. (1 mark) Which structure controls what enters and leaves a cell?
    [CLO: G11-BIO-U1-01  |  Bloom: Analyse  |  Marks: 1]
    A. Cell wall
    B. Cell (plasma) membrane
    ...

That bracket is a much stronger signal than guessing: it gives an exact
curriculum.clos.code to link the question to (see service.py, which
verifies the code actually exists for the exam's subject before writing
it -- clo_code is a real FK) and an authoritative marks value, preferred
over the "(1 mark)" annotation inline in the question text.

For each question found:
  - questionText: just the question stem -- option lines (A)/B)/C)/D)) are
    split out into their own structured list rather than left concatenated
    into the text, so the frontend can render real labeled choice buttons
    instead of a raw wall of text.
  - options: [{"letter": "A", "text": "Cell membrane"}, ...] for mcq
    questions, None otherwise.
  - marks: the CLO bracket's "Marks:" value if present, else an inline
    annotation (e.g. "(5 marks)", "[10 pts]") stripped out of the stem,
    else DEFAULT_MARKS with a logged warning (assessment.questions.marks
    is NOT NULL).
  - questionType: classified against the DB's actual 5-value enum. Option
    lines are the strongest signal (2+ -> mcq); otherwise the enclosing
    section's own label is used when set (e.g. "Section C -- Extended
    Response" -> essay) since real documents self-describe this far more
    reliably than per-question keyword guessing; a keyword heuristic over
    the question text is the last-resort fallback.
  - partLabel: the nearest preceding "Part X"/"Section X" heading, if any.
  - cloCode: from the bracket annotation, if present (else None).

This is intentionally the same "deterministic, dependency-light, provisional
first draft" philosophy as curriculum_parser/extractor.py -- getting a
structured draft in front of a human/2B-validation quickly, not a perfect
final read.
"""

from __future__ import annotations

import re
from typing import Optional

import fitz  # PyMuPDF
import structlog

logger = structlog.get_logger()

SECTION_HEADER_RE = re.compile(r"^\s*(?:part|section)\s+([A-Za-z0-9]+)\b", re.IGNORECASE)
QUESTION_START_RE = re.compile(r"^\s*(?:q(?:uestion)?\.?\s*)?(\d{1,3})[.)]\s*(.*)$", re.IGNORECASE)
MCQ_OPTION_RE = re.compile(r"^\s*([A-Da-d])[.)]\s+(\S.*)$")
MARKS_RE = re.compile(r"[\(\[]\s*(\d+)\s*(?:marks?|pts?|points?)\s*[\)\]]", re.IGNORECASE)
CLO_BRACKET_RE = re.compile(
    r"^\s*\[\s*CLO:\s*([^|\]]+?)\s*\|\s*Bloom:\s*([^|\]]+?)\s*\|\s*Marks:\s*(\d+)\s*\]\s*$",
    re.IGNORECASE,
)
# Marks the end of the question list in "Extraction-Ready" exam documents --
# everything from here on (an answer key / marking-guide table, teacher
# notes) is reference material, not another question, and would otherwise
# get glued onto the last question's text since nothing else terminates it
# before end of document.
STOP_MARKER_RE = re.compile(r"^\s*answer\s+key\b", re.IGNORECASE)

ESSAY_RE = re.compile(r"\b(discuss|explain in detail|essay|elaborate)\b", re.IGNORECASE)
CALCULATION_RE = re.compile(r"\b(calculate|solve|compute|find the value)\b", re.IGNORECASE)
SHORT_ANSWER_RE = re.compile(r"\b(define|state|list|name|identify)\b", re.IGNORECASE)

DEFAULT_MARKS = 1


def extract_questions(pdf_bytes: bytes) -> list[dict]:
    """Entry point: parses the PDF and returns a flat list of question dicts
    ready for app.db.postgres_assessment.save_exam_questions."""
    doc = fitz.open(stream=pdf_bytes, filetype="pdf")
    try:
        lines: list[str] = []
        for page in doc:
            lines.extend(page.get_text().splitlines())
        return _parse_lines(lines)
    finally:
        doc.close()


def _parse_lines(lines: list[str]) -> list[dict]:
    questions: list[dict] = []
    current_section: Optional[str] = None
    buf: list[str] = []
    options: list[dict] = []
    bracket_marks: Optional[int] = None
    clo_code: Optional[str] = None

    def finalize():
        nonlocal buf, options, bracket_marks, clo_code
        if not buf:
            return
        raw_text = " ".join(buf).strip()
        text, inline_marks = _extract_marks(raw_text)
        marks = bracket_marks if bracket_marks is not None else inline_marks
        if marks is None:
            logger.warning("exam_parse.marks_not_found", question_preview=text[:60])
            marks = DEFAULT_MARKS
        questions.append(
            {
                "sequenceNumber": len(questions) + 1,
                "questionText": text,
                "options": options or None,
                "questionType": _classify_question_type(text, len(options), current_section),
                "marks": marks,
                "partLabel": current_section,
                "cloCode": clo_code,
            }
        )
        buf, options, bracket_marks, clo_code = [], [], None, None

    for raw_line in lines:
        line = raw_line.strip()
        if not line:
            continue

        if STOP_MARKER_RE.match(line):
            break

        section_match = SECTION_HEADER_RE.match(line)
        if section_match:
            finalize()
            current_section = line  # keep the full label ("Section C -- Extended Response") for both display and type-hinting
            continue

        q_match = QUESTION_START_RE.match(line)
        if q_match:
            finalize()
            buf.append(q_match.group(2).strip())
            continue

        if not buf:
            continue  # body text before the first numbered question -- discarded

        bracket_match = CLO_BRACKET_RE.match(line)
        if bracket_match:
            clo_code = bracket_match.group(1).strip()
            bracket_marks = int(bracket_match.group(3))
            continue  # annotation only -- not part of the visible question text

        option_match = MCQ_OPTION_RE.match(line)
        if option_match:
            options.append({"letter": option_match.group(1).upper(), "text": option_match.group(2).strip()})
            continue  # split out, not appended to the stem

        buf.append(line)

    finalize()
    return questions


def _extract_marks(text: str) -> tuple[str, Optional[int]]:
    m = MARKS_RE.search(text)
    if not m:
        return text, None
    marks = int(m.group(1))
    cleaned = re.sub(r"\s+", " ", text[: m.start()] + text[m.end() :]).strip()
    return cleaned, marks


def _section_type_hint(section_label: Optional[str]) -> Optional[str]:
    if not section_label:
        return None
    lower = section_label.lower()
    if "multiple choice" in lower or "mcq" in lower:
        return "mcq"
    if "extended" in lower or "essay" in lower:
        return "essay"
    if "calculation" in lower or "problem" in lower:
        return "calculation"
    if "short" in lower:
        return "short_answer"
    if "long" in lower:
        return "long_answer"
    return None


def _classify_question_type(text: str, option_count: int, section_label: Optional[str]) -> str:
    if option_count >= 2:
        return "mcq"

    hint = _section_type_hint(section_label)
    if hint:
        return hint

    if CALCULATION_RE.search(text):
        return "calculation"
    if ESSAY_RE.search(text):
        return "essay"
    if SHORT_ANSWER_RE.search(text):
        return "short_answer"
    return "long_answer"
