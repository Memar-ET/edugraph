"""
Shared strategy for documents following the "Unified ID Convention"
curriculum format (see such a document's own "Unified ID Convention"
section) -- e.g. the Ethiopian MoE Biology G7-G12 syllabus. Used by both
extractor.py (PDF) and docx_extractor.py (DOCX): the format is
plain-text-shaped (bullet lines carrying a strict dot-separated ID
prefix, e.g. "G9.U2.T2.2.CLO1: ..."), not table-driven and not
heading-hierarchy-driven the way the other two strategies are, so it
needs its own detection + extraction path rather than a tweak to either.

Detection and parsing both operate on a single flattened text blob
(paragraphs/lines joined with "\\n" by the caller) rather than anything
format-specific -- confirmed necessary by direct inspection of a real
such document in both PDF and DOCX form: Grade 9-10 content in *both*
formats has Markdown "###"/"####" section markers leaked into plain
paragraph text instead of proper heading structure (a source-document
defect, present identically in both exports, not a bug introduced by
either conversion), which collapses what should be several logical lines
into one. Splitting the whole blob on regex *marker* boundaries (not on
"\\n", and not per-paragraph) handles that uniformly. The PDF path adds
its own wrinkle on top -- LaTeX's line-wrap hyphenation ("...the differ-
ent resolutions...") breaks a sentence across a raw newline mid-word --
handled the same way: whitespace *inside* a marker-delimited chunk
(including embedded newlines) collapses to single spaces rather than
being treated as a line break, so a wrapped sentence stays intact
regardless of where PyMuPDF happened to break the raw text.

Scoped to ONE grade per call (grade_level, e.g. 9 -> "G9"), since a real
curriculum upload is for one subject+grade even when the source document
bundles several (this Biology syllabus covers all of G7-G12 plus a
prerequisites appendix, "PART II", in one file) -- everything outside the
requested grade is ignored here. Prerequisites are out of scope for this
module entirely: curriculum.upload_jobs.parsed_structure has no slot for
them (they're recorded separately, post-approval, once topics have real
IDs -- see curriculum/handler/prerequisites.go), so "PART II" is never
parsed, only skipped past.
"""

from __future__ import annotations

import re
from typing import Optional

DASH = "—"  # em dash -- consistent across both the PDF and DOCX exports of this format

UNIT_RE = re.compile(
    rf"^UNIT (G\d+)\.U(\d+)\s*{DASH}\s*(.+?)\s*\((\d+)\s*periods?\)\s*$"
)
# The .y position is usually numeric but can carry a single alphabetic
# suffix (e.g. "G9.U5.T5.2b" -- a sibling topic sharing position "5.2"
# with T5.2, confirmed by direct inspection, not a typo to normalize
# away), hence the optional [a-z]?. The "(N periods)" suffix is also
# optional -- the one topic with a lettered suffix is also the one topic
# in the whole document missing it.
TOPIC_RE = re.compile(
    rf"^#{{0,3}}\s*(G\d+)\.U(\d+)\.T(\d+)\.(\d+[a-z]?)\s*{DASH}\s*(.+?)\s*(?:\((\d+)\s*periods?\))?\s*$"
)
# Multi-word marker phrases use \s+ between words, not a literal space:
# PDF text-extraction can render a phrase across a line wrap as a raw
# newline mid-phrase (confirmed by direct inspection -- "Topic/Subtopic
# CLOs (...)" is long enough to wrap in this document's column width,
# coming out as "Topic/Subtopic\nCLOs (...)"), which a literal space
# would silently fail to match. Header patterns also tolerate an optional
# trailing page number -- a PDF page-footer digit can land immediately
# after a header when that header happens to be the last thing on its
# page (confirmed: "Topic/Subtopic CLOs (G11.U5.T5.1): 57" is that
# topic's real header with its page's footer number stuck on the end).
# Scoped to just these anchored header patterns, not free-text CLO/
# subtopic bodies, so a description that genuinely ends in a number
# ("...produces 36 ATP") is never at risk of being truncated.
_TRAILING_PAGE_NUM = r"(?:\s*\d+)?"
UNIT_CLO_HDR_RE = re.compile(r"^Unit\s+CLOs\s*\((G\d+)\.U(\d+)\)" + _TRAILING_PAGE_NUM + r"$")
UNIT_CLO_RE = re.compile(r"^(G\d+)\.U(\d+)\.CLO(\d+):\s*(.+)$")
SUBTOPIC_HDR_RE = re.compile(r"^Subtopics\s*\((G\d+)\.U(\d+)\.T(\d+)\.(\d+[a-z]?)\):" + _TRAILING_PAGE_NUM + r"$")
SUBTOPIC_RE = re.compile(r"^(G\d+)\.U(\d+)\.T(\d+)\.(\d+[a-z]?)\.S(\d+):\s*(.+)$")
TOPIC_CLO_HDR_RE = re.compile(
    r"^Topic/Subtopic\s+CLOs\s*\((G\d+)\.U(\d+)\.T(\d+)\.(\d+[a-z]?)\):" + _TRAILING_PAGE_NUM + r"$"
)
TOPIC_CLO_RE = re.compile(r"^(G\d+)\.U(\d+)\.T(\d+)\.(\d+[a-z]?)\.CLO(\d+):\s*(.+)$")
ASSESSMENT_HDR_RE = re.compile(r"^#{0,4}\s*Assessment\s*\((G\d+)\.U(\d+)\)" + _TRAILING_PAGE_NUM + r"$")
ASSESSMENT_RE = re.compile(r"^(G\d+)\.U(\d+)\.ASSESS(\d+):\s*(.+)$")

# Detection: a document following this format always has all three of
# these markers (checked instead of a single regex so a near-miss format
# -- e.g. one with Unit CLOs but no Subtopics -- doesn't silently
# misfire into this strategy). Checked against whitespace-normalized
# text since a marker phrase can wrap across a raw newline (see above).
_DETECT_MARKERS = ("Unit CLOs (", "Subtopics (", "Topic/Subtopic CLOs (")


def looks_like_id_convention(text: str) -> bool:
    normalized = re.sub(r"\s+", " ", text)
    return all(m in normalized for m in _DETECT_MARKERS) and bool(re.search(r"UNIT G\d+\.U\d+\s", normalized))


# Boundary patterns for the main body: anywhere one of these starts
# mid-chunk, force a split point there so a paragraph/line carrying
# several markers (the Grade 9-10 leaked-Markdown case) becomes separate
# logical entries, same as a one-marker-per-line source.
_BOUNDARY_BODY = re.compile(
    r"(?=UNIT G\d+\.U\d+\s)"
    r"|(?=#{0,3}\s*G\d+\.U\d+\.T\d+\.\d+[a-z]?\s*" + DASH + r")"
    r"|(?=Unit\s+CLOs\s*\()"
    r"|(?=G\d+\.U\d+\.CLO\d+:)"
    r"|(?=Subtopics\s*\()"
    r"|(?=G\d+\.U\d+\.T\d+\.\d+[a-z]?\.S\d+:)"
    r"|(?=Topic/Subtopic\s+CLOs\s*\()"
    r"|(?=G\d+\.U\d+\.T\d+\.\d+[a-z]?\.CLO\d+:)"
    r"|(?=#{0,4}\s*Assessment\s*\()"
    r"|(?=G\d+\.U\d+\.ASSESS\d+:)"
    r"|(?=PART II)"
)

_LEADING_BULLET_RE = re.compile(r"^[•‣◦⁃∙\-*]\s*")
_TRAILING_BULLET_RE = re.compile(r"\s*[•‣◦⁃∙]\s*$")
# LaTeX/PDF line-wrap hyphenation ("...the differ-\nent resolutions...")
# breaks a word across a raw newline; reconnect it into one word before
# whitespace-collapse turns that newline into a plain space (which would
# otherwise leave "differ- ent" -- not broken, just an odd extra space,
# and silently merging is right far more often than not for this kind of
# wrap).
_WRAP_HYPHEN_RE = re.compile(r"(\w)-\s*\n\s*(\w)")


def _tokenize(text: str) -> list[str]:
    """Splits the whole document into logical lines on marker boundaries
    (not on raw "\\n"), collapsing any whitespace -- including embedded
    newlines from PDF word-wrap -- inside each resulting chunk to single
    spaces, then strips a leading/trailing bullet glyph if the source
    rendered one literally (PDF text does; DOCX paragraph text doesn't --
    a *trailing* one shows up here because the *next* bullet's glyph
    sits right before the marker that starts the next chunk, so it's
    left dangling at the end of this one)."""
    text = _WRAP_HYPHEN_RE.sub(r"\1\2", text)
    lines: list[str] = []
    for chunk in _BOUNDARY_BODY.split(text):
        normalized = re.sub(r"\s+", " ", chunk).strip()
        normalized = _LEADING_BULLET_RE.sub("", normalized)
        normalized = _TRAILING_BULLET_RE.sub("", normalized).strip()
        if normalized:
            lines.append(normalized)
    return lines


def extract_grade(text: str, grade_level: int, subject_code: str, academic_year: str) -> dict:
    """Extracts one grade's units/topics/subtopics/CLOs, ignoring every
    other grade's content and the Part II prerequisites appendix (if
    present). Returns the same JSON-serializable shape extractor.py's
    other strategies produce, extended with `subtopics` (nested, same
    shape as a topic) and `unitClos` (unit-level CLOs, mapped to every
    topic/subtopic in the unit at approval time -- see
    ApproveAndPromote), which only this strategy populates."""
    grade_prefix = f"G{grade_level}"
    # rpartition, not partition: a table of contents typically mentions
    # "PART II" once on its own (as a TOC entry) well before the actual
    # section heading later in the document: cutting at the *first*
    # occurrence would discard nearly the whole document.
    body_text, sep, _ = text.rpartition("PART II")
    if not sep:
        body_text = text
    lines = _tokenize(body_text)

    units: list[dict] = []
    cur_unit: Optional[dict] = None
    cur_topic: Optional[dict] = None
    mode: Optional[str] = None  # "unit_clos" | "subtopics" | "topic_clos" | "assessment" | None
    warnings: list[str] = []

    for line in lines:
        m = UNIT_RE.match(line)
        if m:
            g, u, title, periods = m.groups()
            cur_topic = None
            mode = None
            if g != grade_prefix:
                cur_unit = None  # a different grade's unit -- ignore its content until the next match
                continue
            cur_unit = {
                "number": int(u),
                "titleEn": title,
                "topics": [],
                "unitClos": [],
                "_seq": 0,  # internal sequence counter, stripped before returning
            }
            units.append(cur_unit)
            continue

        if cur_unit is None:
            continue  # content belonging to a grade we're not extracting

        m = UNIT_CLO_HDR_RE.match(line)
        if m:
            mode = "unit_clos"
            continue

        m = UNIT_CLO_RE.match(line)
        if m and mode == "unit_clos":
            _, _, _, desc = m.groups()
            code = line.split(":", 1)[0].strip()
            cur_unit["unitClos"].append(_clo(code, desc))
            continue

        m = TOPIC_RE.match(line)
        if m:
            g, u, x, y, title, periods = m.groups()
            if g != grade_prefix:
                cur_topic = None
                continue
            cur_unit["_seq"] += 1
            cur_topic = {
                "sequenceOrder": cur_unit["_seq"],
                "titleEn": title,
                "keyConcepts": [],
                "learningOutcomes": [],
                "clos": [],
                "rawText": "",
                "externalCode": f"{g}.U{u}.T{x}.{y}",
                "subtopics": [],
            }
            cur_unit["topics"].append(cur_topic)
            mode = None
            continue

        m = SUBTOPIC_HDR_RE.match(line)
        if m:
            mode = "subtopics"
            continue

        m = SUBTOPIC_RE.match(line)
        if m and mode == "subtopics" and cur_topic is not None:
            g, u, x, y, k, desc = m.groups()
            cur_unit["_seq"] += 1
            cur_topic["subtopics"].append(
                {
                    "sequenceOrder": cur_unit["_seq"],
                    "titleEn": desc,
                    "keyConcepts": [],
                    "learningOutcomes": [],
                    "clos": [],
                    "rawText": "",
                    "externalCode": f"{g}.U{u}.T{x}.{y}.S{k}",
                    "subtopics": [],
                }
            )
            continue

        m = TOPIC_CLO_HDR_RE.match(line)
        if m:
            mode = "topic_clos"
            continue

        m = TOPIC_CLO_RE.match(line)
        if m and mode == "topic_clos" and cur_topic is not None:
            _, _, _, _, _, desc = m.groups()
            code = line.split(":", 1)[0].strip()
            cur_topic["clos"].append(_clo(code, desc))
            continue

        if ASSESSMENT_HDR_RE.match(line):
            mode = "assessment"  # deliberately not persisted -- no schema home, same as the docx/PDF table strategies
            cur_topic = None
            continue
        if ASSESSMENT_RE.match(line) and mode == "assessment":
            continue

    for u in units:
        del u["_seq"]
        if not u["topics"]:
            warnings.append(f'Unit "{u["titleEn"]}" (grade {grade_level}) had no topics extracted.')

    if not units:
        warnings.append(
            f"Document matched the Unified ID Convention format but no UNIT G{grade_level}.* "
            "headings were found -- check that the requested grade level is actually present."
        )

    return {
        "subjectCode": subject_code,
        "gradeLevel": grade_level,
        "academicYear": academic_year,
        "extractionStrategy": "id_convention",
        "pageCount": None,
        "units": units,
        "warnings": warnings,
    }


def _clo(code: str, description: str) -> dict:
    return {
        "code": code,
        "description": description,
        "bloomLevel": None,
        "mandatory": True,
        "keyConcept": None,
        "evidence": None,
    }
