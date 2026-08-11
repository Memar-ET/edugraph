"""
DOCX counterpart to extractor.py (Capability 2A).

Exam DOCX files won't reliably use Heading styles per-question the way
curriculum unit documents do, so this reuses the same flat, regex-driven
line parser as the PDF path (_parse_lines in extractor.py) -- only the
text-extraction front end differs (python-docx paragraphs vs PyMuPDF page
text).
"""

from __future__ import annotations

import io

import docx

from app.services.exam_parser.extractor import _parse_lines


def extract_questions(docx_bytes: bytes) -> list[dict]:
    document = docx.Document(io.BytesIO(docx_bytes))
    lines = [p.text for p in document.paragraphs]
    return _parse_lines(lines)
