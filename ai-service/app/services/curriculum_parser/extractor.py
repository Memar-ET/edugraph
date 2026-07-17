"""
Core "brain work" of curriculum PDF parsing.

Two heading-detection strategies, tried in order:
  A) TOC strategy   -- if the PDF has an embedded Table of Contents,
                        use it as the exact skeleton.
  B) Font heuristic -- otherwise, cluster heading candidates by font
                        size/boldness (e.g. larger+bold = level 1, smaller
                        bold = level 2).

Either way we end up with a flat, ordered list of "headings"
(level, title, page, rect-or-None).

On top of that, extraction is scoped to the curriculum's
"Unit(s), Topics and CLOs" section: we look for a level-1 heading whose
title matches that phrase (see SECTION_HEADING_RE) and only extract
between it and the next level-1 heading (or end of document). Inside
that scope, each level-2 heading is a Unit. A Unit's structured data
comes from the *tables* under it, not further headings -- curriculum
documents built for machine ingestion (the "Extraction-Ready" format)
put a label/value metadata table right under the unit heading
(subjectCode, unitNumber, titleEn, focus, ...) and a CLO table
(cloCode | description | bloomLevel | mandatory | Key Concept/Topic |
Evidence of Learning) under a numbered sub-heading (e.g. "5.1.1 Topics
and Curriculum Learning Outcomes"). We don't need to detect that
sub-heading precisely: table *shape* alone (a code+description header
vs. a plain 2-column table) is enough to tell the two apart, which is
more robust than trying to guess a third font tier. CLO rows are then
grouped by their "Key Concept / Topic" cell to form Topics, so a unit
with several key concepts ends up with several topics.

If no such section heading is found at all, we fall back to the older
whole-document heading extraction (level-1 = Unit, level-2 = Topic) so
documents that don't follow the new format still produce something.

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

# Matches a level-1 heading like "5. Unit, Topics and CLOs" or
# "Units, Topics and CLOs" (optional leading numbering, singular/plural
# "Unit(s)", optional comma) -- this is the section boundary, not a unit
# itself. Compared against whitespace-normalized heading text.
SECTION_HEADING_RE = re.compile(
    r"^\d*\.?\s*units?\s*,?\s*topics?\s*and\s*clos?\.?$", re.IGNORECASE
)

# Matches a unit's "Topics and (Curriculum) Learning Outcomes" sub-heading
# (e.g. "5.1.1. Topics and Curriculum Learning Outcomes"). A PDF's embedded
# TOC sometimes flattens this to the same outline level as the unit heading
# above it (level 2, same as "5.1. unit 1: ..."), so we can't rely on
# heading level alone to tell a new unit from this sub-heading -- match on
# text and fold it into the current unit instead of starting a new one.
TOPICS_MARKER_RE = re.compile(
    r"topics?\s+and\s+(?:curriculum\s+)?learning\s+outcomes", re.IGNORECASE
)

# Bloom's taxonomy levels curriculum.clos.bloom_level's CHECK constraint
# allows (backend/db/migrations/V011__updated_curriculum.sql). PDF table
# cells that wrap mid-word (narrow columns) can turn "understand" into
# "understan\nd"; comparing the whitespace-stripped cell text against this
# set lets us reconstruct the exact enum value rather than send a value
# that fails the constraint at approval time.
BLOOM_LEVELS = {"remember", "understand", "apply", "analyse", "evaluate", "create"}

# Header-keyword -> field name for a CLO table, matched with "in" against
# the lowercased header cell text so column order/exact wording can vary.
# Shared with docx_extractor.py.
CLO_COLUMN_HINTS = {
    "code": "code",
    "learners will be able": "description",
    "description": "description",
    "bloom": "bloomLevel",
    "mandatory": "mandatory",
    "concept": "keyConcept",
    "topic": "topic",
    "evidence": "evidence",
}


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
    clos: list[dict] = field(default_factory=list)
    raw_text: str = ""
    page: int = 0


@dataclass
class UnitNode:
    number: int
    title_en: str
    topics: list[TopicNode] = field(default_factory=list)
    page: int = 0
    metadata: dict = field(default_factory=dict)


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
        headings, base_strategy = _find_headings(doc, warnings)
        section_start, section_end = _locate_units_topics_clos_section(headings)

        if section_start is not None:
            scoped = headings[section_start + 1 : section_end if section_end is not None else len(headings)]
            boundary = headings[section_end] if section_end is not None else None
            units = _build_units_topics_clos_tree(doc, scoped, boundary, warnings)
            strategy = f"{base_strategy}_units_topics_clos"
            if not units:
                warnings.append(
                    'Found a "Unit(s), Topics and CLOs" heading but no level-2 unit '
                    "sub-headings under it; nothing was extracted from that section."
                )
        elif not headings:
            warnings.append(
                "No table of contents and no clear heading structure were "
                "found; the whole document was kept as a single unit for "
                "manual review."
            )
            units = [_whole_document_as_one_unit(doc)]
            strategy = base_strategy
        else:
            warnings.append(
                'No "Unit(s), Topics and CLOs" heading was found; used the legacy '
                "whole-document heading extraction instead."
            )
            units = _build_tree(doc, headings)
            strategy = base_strategy

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
        # its page, so we can later slice body text/tables more precisely
        # than "the whole page".
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
    # the largest size(s) become level 1, the next tier becomes level 2.
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
# "Unit(s), Topics and CLOs" section scoping
# ---------------------------------------------------------------------------

def _normalize_ws(text: str) -> str:
    return re.sub(r"\s+", " ", text).strip()


def _locate_units_topics_clos_section(headings: list[Heading]) -> tuple[Optional[int], Optional[int]]:
    """Returns (start_index, end_index) of the section heading and the next
    level-1 heading after it (end_index is None if it runs to the end of the
    document). Returns (None, None) if no such section heading exists."""
    start = None
    for i, h in enumerate(headings):
        if h.level <= 1 and SECTION_HEADING_RE.match(_normalize_ws(h.title)):
            start = i
            break
    if start is None:
        return None, None

    end = None
    for j in range(start + 1, len(headings)):
        if headings[j].level <= 1:
            end = j
            break
    return start, end


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


def _tables_between(
    doc: fitz.Document, start: Heading, end: Optional[Heading]
) -> list[list[list[str]]]:
    """Finds tables (as raw row-of-cell-strings) between one heading and the
    next, using the same page/rect bounding logic as _text_between."""
    end_page = end.page if end else doc.page_count - 1
    end_rect = end.rect if end else None

    tables: list[list[list[str]]] = []
    for page_no in range(start.page, min(end_page, doc.page_count - 1) + 1):
        page = doc[page_no]
        try:
            finder = page.find_tables()
        except Exception:
            continue
        for tab in finder.tables:
            bbox = fitz.Rect(tab.bbox)
            if page_no == start.page and start.rect is not None and bbox.y1 <= start.rect.y1:
                continue  # table sits above/at the heading itself
            if page_no == end_page and end_rect is not None and page_no == (end.page if end else -1):
                if bbox.y0 >= end_rect.y0:
                    continue  # table belongs to the next heading onward
            rows = tab.extract()
            if rows:
                tables.append([[_clean_pdf_cell(cell) for cell in row] for row in rows])
    return tables


def _clean_pdf_cell(text: Optional[str]) -> str:
    """Narrow PDF table columns wrap cell text onto multiple lines with no
    marker of whether the break replaced a space (two words) or split a
    single token (e.g. a hyphenated code). We join with a space (correct
    for the common prose case) and then rejoin a trailing hyphen that
    wrapped onto its own line (e.g. "G11-BIO-U1\\n-01" -> "G11-BIO-U1-01"),
    which is the other common case in these tables."""
    joined = (text or "").replace("\n", " ")
    joined = re.sub(r"\s+-\s*", "-", joined)
    return _normalize_ws(joined)


# ---------------------------------------------------------------------------
# CLO / metadata table classification (shared with docx_extractor.py)
# ---------------------------------------------------------------------------

def _map_clo_columns(header_cells: list[str]) -> dict[str, int]:
    # Compare with all whitespace stripped on both sides: a narrow PDF table
    # column can wrap a header word mid-token (e.g. "mandatory" ->
    # "mandato ry"), which would otherwise break substring matching against
    # a clean hint like "mandatory".
    mapping: dict[str, int] = {}
    for idx, cell_text in enumerate(header_cells):
        compact = re.sub(r"\s+", "", cell_text or "").lower()
        for hint, col in CLO_COLUMN_HINTS.items():
            hint_compact = re.sub(r"\s+", "", hint)
            if hint_compact in compact and col not in mapping:
                mapping[col] = idx
                break
    return mapping


def _rows_as_clo_rows(rows: list[list[str]]) -> Optional[list[dict]]:
    """A table is CLO-shaped if its header row has both a code and a
    description column. Returns None if the table doesn't match, so callers
    can fall back to treating it as metadata or plain text."""
    if len(rows) < 2:
        return None
    header_cells = [(c or "").strip() for c in rows[0]]
    columns = _map_clo_columns(header_cells)
    if "code" not in columns or "description" not in columns:
        return None

    out: list[dict] = []
    for row in rows[1:]:
        cells = [(c or "").strip() for c in row]
        if not any(cells):
            continue
        get = lambda col: cells[columns[col]] if col in columns and columns[col] < len(cells) else ""
        code = get("code")
        if not code:
            continue
        out.append(
            {
                "code": code,
                "description": get("description"),
                "bloomLevel": _normalize_bloom_level(get("bloomLevel")),
                "mandatory": get("mandatory").strip().upper() in ("TRUE", "YES", "1"),
                "keyConcept": get("keyConcept") or None,
                "topic": get("topic") or None,
                "evidence": get("evidence") or None,
            }
        )
    return out or None


def _normalize_bloom_level(raw: str) -> Optional[str]:
    if not raw:
        return None
    compact = re.sub(r"\s+", "", raw).lower()
    if compact in BLOOM_LEVELS:
        return compact
    return raw.strip().lower() or None


def _rows_as_metadata(rows: list[list[str]]) -> dict[str, str]:
    """Treats a table as a flat label/value table (e.g. unit metadata:
    subjectCode | BIO, titleEn | Cell Biology, ...). Every row with at least
    two non-empty cells contributes one key/value pair; there's no header
    row to skip since these tables are pure data."""
    meta: dict[str, str] = {}
    for row in rows:
        cells = [c for c in ((c or "").strip() for c in row) if c]
        if len(cells) >= 2:
            # Keys are always single camelCase tokens (subjectCode, titleEn,
            # indicativeCloCount, ...); a narrow PDF column can wrap one
            # mid-word (e.g. "indicativeCloCoun t"), so strip all internal
            # whitespace from the key specifically. Values are free text
            # (e.g. "focus"), where spaces are meaningful, so only collapse.
            key = re.sub(r"\s+", "", cells[0])
            value = cells[1]
            if key:
                meta[key] = value
    return meta


def _int_or_none(value: Optional[str]) -> Optional[int]:
    if not value:
        return None
    try:
        return int(value.strip())
    except (ValueError, AttributeError):
        return None


def _group_clos_into_topics(clo_rows: list[dict], page: int) -> list[TopicNode]:
    """Groups CLO rows into Topics using their "Topic" cell (e.g. "U1-T1:
    Cell Structure and Organelles"). Older-format documents that only have a
    combined "Key Concept / Topic" column (no distinct Topic column) fall
    back to grouping by keyConcept instead, same as before."""
    topics: dict[str, TopicNode] = {}
    order: list[str] = []
    for clo in clo_rows:
        key = clo.get("topic") or clo.get("keyConcept") or "General"
        if key not in topics:
            order.append(key)
            topics[key] = TopicNode(sequence_order=len(order), title_en=key, page=page)
        topic = topics[key]
        key_concept = clo.get("keyConcept")
        if key_concept and key_concept not in topic.key_concepts:
            topic.key_concepts.append(key_concept)
        if clo.get("description"):
            topic.learning_outcomes.append(clo["description"])
        topic.clos.append(clo)
    return [topics[k] for k in order]


# ---------------------------------------------------------------------------
# Tree building: "Unit(s), Topics and CLOs" section (new, table-driven)
# ---------------------------------------------------------------------------

def _build_units_topics_clos_tree(
    doc: fitz.Document,
    scoped: list[Heading],
    section_end: Optional[Heading],
    warnings: list[str],
) -> list[UnitNode]:
    unit_headings = [
        h for h in scoped if h.level <= 2 and not TOPICS_MARKER_RE.search(h.title)
    ]
    if not unit_headings:
        return []

    units: list[UnitNode] = []
    for i, h in enumerate(unit_headings):
        next_unit = unit_headings[i + 1] if i + 1 < len(unit_headings) else None
        region_end = next_unit or section_end

        unit_meta: dict[str, str] = {}
        clo_rows: list[dict] = []
        for rows in _tables_between(doc, h, region_end):
            clos = _rows_as_clo_rows(rows)
            if clos is not None:
                clo_rows.extend(clos)
                continue
            unit_meta.update(_rows_as_metadata(rows))

        title_en = unit_meta.get("titleEn") or h.title
        number = _int_or_none(unit_meta.get("unitNumber")) or (i + 1)
        unit = UnitNode(number=number, title_en=title_en, page=h.page, metadata=unit_meta)

        if clo_rows:
            unit.topics = _group_clos_into_topics(clo_rows, h.page)
        else:
            body = _text_between(doc, h, region_end)
            concepts, outcomes, raw_leftover = _classify_body_text(body)
            unit.topics = [
                TopicNode(
                    sequence_order=1,
                    title_en=title_en,
                    key_concepts=concepts,
                    learning_outcomes=outcomes,
                    raw_text=raw_leftover,
                    page=h.page,
                )
            ]
            warnings.append(
                f'Unit "{title_en}" has no CLO table; kept its text as a single topic '
                "for manual review."
            )
        units.append(unit)

    return units


# ---------------------------------------------------------------------------
# Tree building: legacy whole-document fallback (level-1 = Unit, level-2 = Topic)
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
    d: dict = {
        "number": unit.number,
        "titleEn": unit.title_en,
        "page": unit.page,
        "topics": [
            {
                "sequenceOrder": t.sequence_order,
                "titleEn": t.title_en,
                "keyConcepts": t.key_concepts,
                "learningOutcomes": t.learning_outcomes,
                "clos": t.clos,
                "rawText": t.raw_text[:5000],  # cap so one giant block can't bloat the row
                "page": t.page,
            }
            for t in unit.topics
        ],
    }
    if unit.metadata:
        d["metadata"] = unit.metadata
    return d
