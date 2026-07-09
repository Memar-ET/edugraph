"""
DOCX counterpart to extractor.py.

Word documents almost always carry proper paragraph styles ("Heading 1",
"Heading 2", ...), which is a much more reliable structural signal than
font-size guessing -- effectively the DOCX equivalent of a PDF's table of
contents. So unlike the PDF path, we don't need a font-heuristic fallback
here: Heading 1 -> Unit, Heading 2 -> Topic.

Curriculum documents built for direct machine ingestion (like the
"Extraction-Ready" format this was designed against) put the actual CLO
data in *tables*, not paragraphs -- python-docx's `document.paragraphs`
silently skips table content entirely, so we walk the document body in
its real order (paragraphs and tables interleaved) and specifically
recognize the CLO table shape:

    cloCode | description ("Learners will be able to...") | bloomLevel |
    mandatory | Key Concept / Topic | Evidence of Learning

When a table matches that shape (detected by header keywords, not exact
column count/order, so re-ordered or slightly renamed columns still
work), each row becomes a structured CLO entry. Any other table (e.g. a
"Unit Metadata" label/value table) is flattened into the current
section's text so nothing is silently dropped, even though we don't
specifically model it.

If a document has no heading styles or CLO-shaped tables at all, we fall
back to treating it as a single unit, same as the PDF path does.
"""

from __future__ import annotations

import re

import docx
from docx.document import Document as _DocxDocument
from docx.oxml.ns import qn
from docx.table import Table
from docx.text.paragraph import Paragraph

from app.services.curriculum_parser.extractor import _classify_body_text

# Header-keyword -> field name. Matched with "in" against the lowercased
# header cell text, so column order/exact wording can vary.
CLO_COLUMN_HINTS = {
    "code": "code",
    "learners will be able": "description",
    "description": "description",
    "bloom": "bloomLevel",
    "mandatory": "mandatory",
    "concept": "keyConcept",
    "topic": "keyConcept",
    "evidence": "evidence",
}


def _iter_block_items(document: _DocxDocument):
    """Yield each Paragraph/Table in the document in true document order."""
    for child in document.element.body.iterchildren():
        if child.tag == qn("w:p"):
            yield Paragraph(child, document)
        elif child.tag == qn("w:tbl"):
            yield Table(child, document)


def _map_clo_columns(header_cells: list[str]) -> dict[str, int]:
    mapping: dict[str, int] = {}
    for idx, cell_text in enumerate(header_cells):
        lowered = cell_text.lower()
        for hint, field in CLO_COLUMN_HINTS.items():
            if hint in lowered and field not in mapping:
                mapping[field] = idx
                break
    return mapping


def _table_as_clo_rows(table: Table) -> list[dict] | None:
    if len(table.rows) < 2:
        return None
    header_cells = [c.text.strip() for c in table.rows[0].cells]
    columns = _map_clo_columns(header_cells)
    # Require at minimum a code column and a description column to treat
    # this as a CLO table rather than some other 2-column metadata table.
    if "code" not in columns or "description" not in columns:
        return None

    rows = []
    for row in table.rows[1:]:
        cells = [c.text.strip() for c in row.cells]
        if not any(cells):
            continue
        get = lambda field: cells[columns[field]] if field in columns and columns[field] < len(cells) else ""
        code = get("code")
        if not code:
            continue
        rows.append(
            {
                "code": code,
                "description": get("description"),
                "bloomLevel": get("bloomLevel").lower() or None,
                "mandatory": get("mandatory").strip().upper() in ("TRUE", "YES", "1"),
                "keyConcept": get("keyConcept") or None,
                "evidence": get("evidence") or None,
            }
        )
    return rows or None


def _table_as_plain_lines(table: Table) -> list[str]:
    lines = []
    for row in table.rows:
        cells = [c.text.strip() for c in row.cells if c.text.strip()]
        if cells:
            lines.append(" | ".join(cells))
    return lines


def extract_structure(
    docx_bytes: bytes,
    subject_code: str,
    grade_level: int,
    academic_year: str,
) -> dict:
    import io

    document = docx.Document(io.BytesIO(docx_bytes))
    warnings: list[str] = []

    # sections: {"level", "title", "lines": [...], "clos": [...]}
    sections: list[dict] = []

    for block in _iter_block_items(document):
        if isinstance(block, Paragraph):
            style_name = (getattr(block.style, "name", None) or "").lower()
            text = block.text.strip()
            if not text:
                continue

            if style_name.startswith("heading 1") or style_name == "title":
                sections.append({"level": 1, "title": text, "lines": [], "clos": []})
            elif style_name.startswith("heading 2"):
                sections.append({"level": 2, "title": text, "lines": [], "clos": []})
            elif style_name.startswith("heading"):
                # Heading 3+ folds into the current topic rather than
                # growing the tree deeper than Unit -> Topic.
                if sections:
                    sections[-1]["lines"].append(text)
                else:
                    sections.append({"level": 2, "title": text, "lines": [], "clos": []})
            else:
                if sections:
                    sections[-1]["lines"].append(text)
                # else: text before any heading (cover page etc.) -- discarded.

        elif isinstance(block, Table):
            clo_rows = _table_as_clo_rows(block)
            if clo_rows is not None:
                if sections:
                    sections[-1]["clos"].extend(clo_rows)
                # A CLO table with no preceding heading shouldn't happen in
                # a well-formed doc; if it does, just drop it rather than
                # inventing a section.
            elif sections:
                sections[-1]["lines"].extend(_table_as_plain_lines(block))

    if not sections:
        warnings.append(
            "No heading styles were found in the document; the whole "
            "document was kept as a single unit for manual review."
        )
        all_text = "\n".join(p.text for p in document.paragraphs if p.text.strip())
        concepts, outcomes, raw_leftover = _classify_body_text(all_text)
        return {
            "subjectCode": subject_code,
            "gradeLevel": grade_level,
            "academicYear": academic_year,
            "extractionStrategy": "docx_single_unit",
            "pageCount": None,
            "units": [
                {
                    "number": 1,
                    "titleEn": "Untitled document",
                    "page": 0,
                    "topics": [
                        {
                            "sequenceOrder": 1,
                            "titleEn": "Untitled document",
                            "keyConcepts": concepts,
                            "learningOutcomes": outcomes,
                            "clos": [],
                            "rawText": raw_leftover[:5000],
                            "page": 0,
                        }
                    ],
                }
            ],
            "warnings": warnings,
        }

    units: list[dict] = []
    unit_number = 0
    topic_seq = 0
    current_unit: dict | None = None

    for sec in sections:
        body = "\n".join(sec["lines"])
        concepts, outcomes, raw_leftover = _classify_body_text(body)
        # Fold CLO-table data in: each CLO's description is itself a
        # learning outcome, and its key concept (if present) joins the
        # topic's key concepts list.
        for clo in sec["clos"]:
            if clo["description"]:
                outcomes.append(clo["description"])
            if clo["keyConcept"] and clo["keyConcept"] not in concepts:
                concepts.append(clo["keyConcept"])

        has_content = bool(concepts or outcomes or raw_leftover or sec["clos"])

        if sec["level"] == 1 or current_unit is None:
            unit_number += 1
            topic_seq = 0
            current_unit = {"number": unit_number, "titleEn": sec["title"], "page": None, "topics": []}
            units.append(current_unit)
            if has_content:
                topic_seq += 1
                current_unit["topics"].append(
                    {
                        "sequenceOrder": topic_seq,
                        "titleEn": sec["title"],
                        "keyConcepts": concepts,
                        "learningOutcomes": outcomes,
                        "clos": sec["clos"],
                        "rawText": raw_leftover[:5000],
                        "page": None,
                    }
                )
        else:
            topic_seq += 1
            current_unit["topics"].append(
                {
                    "sequenceOrder": topic_seq,
                    "titleEn": sec["title"],
                    "keyConcepts": concepts,
                    "learningOutcomes": outcomes,
                    "clos": sec["clos"],
                    "rawText": raw_leftover[:5000],
                    "page": None,
                }
            )

    return {
        "subjectCode": subject_code,
        "gradeLevel": grade_level,
        "academicYear": academic_year,
        "extractionStrategy": "docx_heading_styles",
        "pageCount": None,
        "units": [u for u in units if u["topics"]],
        "warnings": warnings,
    }
