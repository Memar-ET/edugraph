# Step 2 — Curriculum Parsing (AI Service)

Almost everything in `ai-service` was an empty placeholder file (0 bytes) —
only `main.py`, `config.py`, and `logging.py` had real content. This fills
in the pieces needed for: pick job off Redis → download file → parse with
PyMuPDF/python-docx → save structured JSON → mark job `parsed`.

## What actually happens now, end to end

1. **`app/main.py`** — on FastAPI startup, launches `curriculum_worker.run_forever()`
   as a background asyncio task (and cancels it cleanly on shutdown). No
   separate worker container needed for local dev — it rides along with
   the existing single `ai-service` container in your compose file.

2. **`app/workers/curriculum_worker.py`** — blocks on `BRPOP queue:curriculum:parse`
   (the exact list the Go backend `LPUSH`es job IDs onto — see
   `internal/curriculum/service/service.go`). This is a plain Redis list,
   not a Celery-protocol message, so it's a small standalone consumer loop
   rather than a Celery task — wiring it through Celery would need the Go
   side to push actual Celery-formatted messages, which it doesn't. If you
   want real Celery later (e.g. for retries/monitoring), the natural move
   is changing the Go push to a proper Celery task submission — happy to
   do that as a follow-up.

3. **`app/services/curriculum_parser/service.py`** — for each job id:
   fetches `curriculum.upload_jobs`, downloads the bytes (currently from
   `storage.local_files` BYTEA, as you asked — this is the *only* function
   that needs to change when you move to S3), picks PDF or DOCX extraction
   by file extension, and writes the result to
   `curriculum.upload_jobs.parsed_structure` (that JSONB column already
   existed from migration V012), setting `status = 'parsed'` — or `'failed'`
   with the error message if anything throws.

4. **`app/services/curriculum_parser/extractor.py`** (PDF) — exactly the
   two-strategy approach from your plan:
   - **Strategy A (TOC):** `doc.get_toc()`. If present, used as the exact
     skeleton — level 1 → Unit, level 2+ → Topic.
   - **Strategy B (font heuristics):** if no TOC, clusters text by font
     size + bold, picks the body-text size as the most common one, and
     treats meaningfully-larger (and/or bold) short lines as headings —
     largest size tier → Unit, next tier → Topic.
   - Either way, body text under each heading is scanned for
     `Key Concepts:` / `Learning Outcomes:` markers (and bullet lists under
     them) to split it into `keyConcepts` / `learningOutcomes`; anything
     left over goes into `rawText` so nothing is silently dropped.

5. **`app/services/curriculum_parser/docx_extractor.py`** (DOCX) — Word
   documents carry real heading styles (`Heading 1`/`Heading 2`), which is
   a cleaner signal than font-guessing, so this walks the document by
   style instead. I tested this directly against the Biology CLOs document
   you uploaded — worth flagging what I found:

   **Your sample document's actual CLO data lives inside Word tables, not
   paragraphs** (`cloCode | description | bloomLevel | mandatory | Key
   Concept/Topic | Evidence of Learning`, matching the field-key section
   the document itself defines). `python-docx`'s normal `document.paragraphs`
   walk silently skips table content — first version of this extractor
   returned empty topics for every unit because of that. Fixed by walking
   the document body in true order (paragraphs *and* tables interleaved)
   and specifically recognizing that column layout by header keywords —
   so it survives the columns being reordered or slightly renamed. Each
   row becomes a structured entry:
   ```json
   { "code": "G11-BIO-U1-01", "description": "...", "bloomLevel": "analyse",
     "mandatory": true, "keyConcept": "Cell structures", "evidence": "Labelled diagrams" }
   ```
   Verified against your actual file: correctly pulled all 6 units, 3 CLOs
   each, with codes/descriptions/bloom levels/key concepts intact. Any
   other table shape (e.g. the 2-column "Unit Metadata" tables) is
   flattened into the section's text rather than dropped, since it's not
   specifically modeled yet.

## The output shape (`parsed_structure`)

Mirrors the real target schema (`curriculum.units`/`curriculum.topics`/`curriculum.clos`
from migration V011) so it's straightforward to promote after human review:

```json
{
  "subjectCode": "BIO",
  "gradeLevel": 11,
  "academicYear": "2026",
  "extractionStrategy": "toc | font_heuristic | docx_heading_styles | docx_single_unit",
  "units": [
    {
      "number": 1,
      "titleEn": "Unit 1: Cell Biology",
      "topics": [
        {
          "sequenceOrder": 1,
          "titleEn": "Topic 1.1: Cell Structure",
          "keyConcepts": ["Prokaryotic cells", "Eukaryotic cells"],
          "learningOutcomes": ["Learners will be able to compare cell types."],
          "clos": [{ "code": "...", "description": "...", "bloomLevel": "...", "mandatory": true, "keyConcept": "...", "evidence": "..." }],
          "rawText": "anything not classified into the above"
        }
      ]
    }
  ],
  "warnings": ["e.g. no TOC found, used font-size heuristic"]
}
```

## What I verified (not just syntax-checked)

Installed the real dependencies (`pymupdf`, `python-docx`, `asyncpg`,
`redis`, `structlog`) and actually ran the code:
- **DOCX path** against your real uploaded file — caught and fixed the
  table-skipping bug above, plus a `paragraph.style is None` crash on
  certain paragraphs (title-page text with no style at all).
- **PDF TOC path** against a synthetic PDF with an explicit TOC — correct.
- **PDF font-heuristic path** against a synthetic PDF with bold
  20pt/14pt/11pt text and no TOC — correctly recovered the same Unit/Topic
  structure as the TOC path.
- **Failure path** — feeding it garbage bytes raises cleanly, which
  `service.py` catches and turns into `status = 'failed'` with the error
  message, rather than crashing the worker loop.
- **Full pipeline** with the DB layer mocked out, using your real
  document's bytes — job correctly ends up "parsed" with all 13 units.

I could not run this against a live Postgres/Redis (no docker in this
sandbox), so the actual SQL execution against your running containers is
the one thing still worth watching on first run.

## One thing to decide

Nothing currently reads `parsed_structure` back out for a human to review
or approve — that's naturally "Step 3" (a review/approve endpoint that
promotes it into `curriculum.units`/`topics`/`clos`). Let me know when
you're ready for that and I'll build it against this same shape.
