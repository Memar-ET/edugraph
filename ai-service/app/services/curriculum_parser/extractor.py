"""
Core "brain work" of curriculum PDF parsing.

Two extraction strategies, tried in order:
  A) TOC strategy   -- if the PDF has an embedded Table of Contents,
                        use it as the exact skeleton (Unit -> Topic).
  B) Font heuristic -- otherwise, cluster heading candidates by font
                        size/boldness (e.g. larger+bold = Unit, smaller
                        bold = Topic).

Either way we end up with a flat, ordered list of "headings"
(level, title, page, rect-or-None), which we then turn into a nested
Unit -> Topic tree, pulling the body text under each heading and
running a light keyword/bullet heuristic over it to split out
"key concepts" vs "learning outcomes".

This is intentionally deterministic and dependency-light (no LLM calls)
so it runs fast and cheaply on every upload. The output is *provisional*
-- it's stored in curriculum.upload_jobs.parsed_structure for a human
curriculum officer to review before anything is promoted into the real
curriculum.units / curriculum.topics tables. A later LLM-assisted pass
(app/utils/llm.py) can refine the concepts/outcomes split further on
messy real-world scans; this heuristic pass is what gets a first draft
in front of a human quickly.
"""

from __future__ import annotations

import re
from collections import Counter
from dataclasses import dataclass, field
from typing import Optional

import fitz  # PyMuPDF

BOLD_FLAG = 1 << 4  # PyMuPDF span flag bit for bold text

CONCEPT_HEADER_RE = re.compile(
    r"^\s*(key\s+concepts?|core\s+concepts?)\s*[:\-]?\s*$", re.IGNORECASE
)
OUTCOME_HEADER_RE = re.compile(
    r"^\s*(learning\s+outcomes?|clos?|learners?\s+will\s+be\s+able\s+to)\s*[:\-]?\s*$",
    re.IGNORECASE,
)
BULLET_RE = re.compile(r"^\s*(?:[-•*▪]|\(?\d+[.)]|\(?[ivxlcdm]+[.)])\s+(.*)", re.IGNORECASE)
INLINE_LABEL_RE = re.compile(
    r"^\s*(key\s+concepts?|core\s+concepts?|learning\s+outcomes?|clos?)\s*[:\-]\s*(.+)$",
    re.IGNORECASE,
)


@dataclass
class Heading:
    level: int
    title: str
    page: int  # 0-indexed
    rect: Optional[fitz.Rect] = None


@dataclass
class TopicNode:
    sequence_order: int
    title_en: str
    key_concepts: list[str] = field(default_factory=list)
    learning_outcomes: list[str] = field(default_factory=list)
    raw_text: str = ""
    page: int = 0


@dataclass
class UnitNode:
    number: int
    title_en: str
    topics: list[TopicNode] = field(default_factory=list)
    page: int = 0


def extract_structure(
    pdf_bytes: bytes,
    subject_code: str,
    grade_level: int,
    academic_year: str,
) -> dict:
    """Entry point: parses the PDF and returns the JSON-serializable tree
    that gets written to curriculum.upload_jobs.parsed_structure."""
    warnings: list[str] = []
    doc = fitz.open(stream=pdf_bytes, filetype="pdf")
    try:
        headings, strategy = _find_headings(doc, warnings)

        if not headings:
            warnings.append(
                "No table of contents and no clear heading structure were "
                "found; the whole document was kept as a single unit for "
                "manual review."
            )
            units = [_whole_document_as_one_unit(doc)]
        else:
            units = _build_tree(doc, headings)

        return {
            "subjectCode": subject_code,
            "gradeLevel": grade_level,
            "academicYear": academic_year,
            "extractionStrategy": strategy,
            "pageCount": doc.page_count,
            "units": [_unit_to_dict(u) for u in units],
            "warnings": warnings,
        }
    finally:
        doc.close()


# ---------------------------------------------------------------------------
# Strategy A: Table of Contents
# ---------------------------------------------------------------------------

def _find_headings(doc: fitz.Document, warnings: list[str]) -> tuple[list[Heading], str]:
    toc = doc.get_toc(simple=True)  # [[level, title, page(1-indexed)], ...]
    if toc:
        headings = [
            Heading(level=lvl, title=title.strip(), page=max(page - 1, 0))
            for lvl, title, page in toc
            if title and title.strip()
        ]
        # Attach a rect for each heading by searching for its title text on
        # its page, so we can later slice body text more precisely than
        # "the whole page".
        for h in headings:
            h.rect = _locate_heading_rect(doc, h)
        return headings, "toc"

    warnings.append("PDF has no embedded table of contents; used font-size heuristics instead.")
    return _find_headings_by_font(doc), "font_heuristic"


def _locate_heading_rect(doc: fitz.Document, heading: Heading) -> Optional[fitz.Rect]:
    if heading.page >= doc.page_count:
        return None
    page = doc[heading.page]
    matches = page.search_for(heading.title)
    if matches:
        return matches[0]
    # Try matching just the first several words in case of truncation/odd
    # whitespace in the TOC title vs. the rendered heading text.
    short = " ".join(heading.title.split()[:4])
    if short:
        matches = page.search_for(short)
        if matches:
            return matches[0]
    return None


# ---------------------------------------------------------------------------
# Strategy B: Font-size / boldness heuristic
# ---------------------------------------------------------------------------

def _find_headings_by_font(doc: fitz.Document) -> list[Heading]:
    lines: list[dict] = []  # {"text", "size", "bold", "page", "rect"}
    size_char_counts: Counter[float] = Counter()

    for page_no in range(doc.page_count):
        page = doc[page_no]
        raw = page.get_text("dict")
        for block in raw.get("blocks", []):
            for line in block.get("lines", []):
                spans = line.get("spans", [])
                if not spans:
                    continue
                text = "".join(s.get("text", "") for s in spans).strip()
                if not text:
                    continue
                # A line's "size" is its dominant span size; "bold" is true
                # if the majority of its characters are bold.
                sizes = [round(s.get("size", 0), 1) for s in spans]
                dominant_size = Counter(sizes).most_common(1)[0][0]
                bold_chars = sum(
                    len(s.get("text", "")) for s in spans if int(s.get("flags", 0)) & BOLD_FLAG
                )
                is_bold = bold_chars >= len(text) / 2
                bbox = fitz.Rect(line.get("bbox", (0, 0, 0, 0)))

                lines.append(
                    {"text": text, "size": dominant_size, "bold": is_bold, "page": page_no, "rect": bbox}
                )
                size_char_counts[dominant_size] += len(text)

    if not lines:
        return []

    body_size = size_char_counts.most_common(1)[0][0]

    # Heading candidates: meaningfully larger than body text, or bold and
    # at least a bit larger. Short lines only (headings aren't paragraphs).
    candidates = [
        ln
        for ln in lines
        if len(ln["text"]) <= 120
        and (ln["size"] >= body_size * 1.3 or (ln["bold"] and ln["size"] >= body_size * 1.1))
    ]
    if not candidates:
        return []

    # Cluster the distinct candidate sizes into up to 2 heading levels:
    # the largest size(s) become Units, the next tier becomes Topics.
    distinct_sizes = sorted({round(c["size"], 1) for c in candidates}, reverse=True)
    level_for_size: dict[float, int] = {}
    if len(distinct_sizes) == 1:
        level_for_size[distinct_sizes[0]] = 1
    else:
        level_for_size[distinct_sizes[0]] = 1
        for s in distinct_sizes[1:]:
            level_for_size[s] = 2

    return [
        Heading(level=level_for_size[round(c["size"], 1)], title=c["text"], page=c["page"], rect=c["rect"])
        for c in candidates
    ]


# ---------------------------------------------------------------------------
# Body text extraction between one heading and the next
# ---------------------------------------------------------------------------

def _text_between(doc: fitz.Document, start: Heading, end: Optional[Heading]) -> str:
    end_page = end.page if end else doc.page_count - 1
    end_rect = end.rect if end else None

    parts: list[str] = []
    for page_no in range(start.page, min(end_page, doc.page_count - 1) + 1):
        page = doc[page_no]
        blocks = page.get_text("blocks")  # (x0, y0, x1, y1, text, block_no, block_type)
        for x0, y0, x1, y1, text, *_ in blocks:
            if not text.strip():
                continue
            if page_no == start.page and start.rect is not None and y1 <= start.rect.y1:
                continue  # this block is the heading itself (or above it)
            if page_no == end_page and end_rect is not None and page_no == (end.page if end else -1):
                if y0 >= end_rect.y0:
                    continue  # this block belongs to the next heading onward
            parts.append(text.strip())
    return "\n".join(parts).strip()


# ---------------------------------------------------------------------------
# Tree building
# ---------------------------------------------------------------------------

def _build_tree(doc: fitz.Document, headings: list[Heading]) -> list[UnitNode]:
    units: list[UnitNode] = []
    unit_number = 0
    topic_seq = 0
    current_unit: Optional[UnitNode] = None

    for i, h in enumerate(headings):
        nxt = headings[i + 1] if i + 1 < len(headings) else None
        body = _text_between(doc, h, nxt)
        concepts, outcomes, raw_leftover = _classify_body_text(body)

        if h.level <= 1 or current_unit is None:
            unit_number += 1
            topic_seq = 0
            current_unit = UnitNode(number=unit_number, title_en=h.title, page=h.page)
            units.append(current_unit)
            # A unit heading can itself carry body text (e.g. a unit intro
            # paragraph) before its first topic sub-heading -- keep it as
            # the unit's first (untitled-topic) text bucket only if there's
            # no deeper heading immediately following it.
            if nxt is None or nxt.level <= 1:
                topic_seq += 1
                current_unit.topics.append(
                    TopicNode(
                        sequence_order=topic_seq,
                        title_en=h.title,
                        key_concepts=concepts,
                        learning_outcomes=outcomes,
                        raw_text=raw_leftover,
                        page=h.page,
                    )
                )
        else:
            topic_seq += 1
            current_unit.topics.append(
                TopicNode(
                    sequence_order=topic_seq,
                    title_en=h.title,
                    key_concepts=concepts,
                    learning_outcomes=outcomes,
                    raw_text=raw_leftover,
                    page=h.page,
                )
            )

    return units


def _whole_document_as_one_unit(doc: fitz.Document) -> UnitNode:
    all_text = "\n".join(doc[p].get_text() for p in range(doc.page_count))
    concepts, outcomes, raw_leftover = _classify_body_text(all_text)
    return UnitNode(
        number=1,
        title_en="Untitled document",
        page=0,
        topics=[
            TopicNode(
                sequence_order=1,
                title_en="Untitled document",
                key_concepts=concepts,
                learning_outcomes=outcomes,
                raw_text=raw_leftover,
                page=0,
            )
        ],
    )


# ---------------------------------------------------------------------------
# Key concepts / learning outcomes classification
# ---------------------------------------------------------------------------

def _classify_body_text(text: str) -> tuple[list[str], list[str], str]:
    concepts: list[str] = []
    outcomes: list[str] = []
    raw_leftover: list[str] = []
    mode: Optional[str] = None  # "concepts" | "outcomes" | None

    for raw_line in text.splitlines():
        line = raw_line.strip()
        if not line:
            continue

        if CONCEPT_HEADER_RE.match(line):
            mode = "concepts"
            continue
        if OUTCOME_HEADER_RE.match(line):
            mode = "outcomes"
            continue

        inline = INLINE_LABEL_RE.match(line)
        if inline:
            label, rest = inline.group(1).lower(), inline.group(2)
            bucket = concepts if "concept" in label else outcomes
            bucket.extend(_split_inline_list(rest))
            continue

        bullet = BULLET_RE.match(line)
        if bullet:
            item = bullet.group(1).strip()
            if mode == "concepts":
                concepts.append(item)
            elif mode == "outcomes":
                outcomes.append(item)
            else:
                # No active section header -- guess from phrasing.
                (outcomes if _looks_like_outcome(item) else raw_leftover).append(item)
            continue

        if mode == "concepts":
            concepts.append(line)
        elif mode == "outcomes":
            outcomes.append(line)
        elif _looks_like_outcome(line):
            outcomes.append(line)
        else:
            raw_leftover.append(line)

    return concepts, outcomes, "\n".join(raw_leftover).strip()


def _looks_like_outcome(line: str) -> bool:
    lowered = line.lower()
    return "able to" in lowered or lowered.startswith(("explain ", "describe ", "identify ", "compare "))


def _split_inline_list(text: str) -> list[str]:
    parts = re.split(r",|;", text)
    return [p.strip() for p in parts if p.strip()]


# ---------------------------------------------------------------------------
# Serialization
# ---------------------------------------------------------------------------

def _unit_to_dict(unit: UnitNode) -> dict:
    return {
        "number": unit.number,
        "titleEn": unit.title_en,
        "page": unit.page,
        "topics": [
            {
                "sequenceOrder": t.sequence_order,
                "titleEn": t.title_en,
                "keyConcepts": t.key_concepts,
                "learningOutcomes": t.learning_outcomes,
                "rawText": t.raw_text[:5000],  # cap so one giant block can't bloat the row
                "page": t.page,
            }
            for t in unit.topics
        ],
    }
