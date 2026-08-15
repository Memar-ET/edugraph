# ADR-010: Magic-Byte File Validation (Not Content-Type Header)

Date: 2026-07-17  
Status: Accepted

## Context

The curriculum upload endpoint accepts PDF and DOCX files. The naive approach is to check the `Content-Type` header sent by the browser. However, `Content-Type` is a client-supplied header — any caller can send `Content-Type: application/pdf` with an arbitrary payload. Accepting files based on header alone means an attacker could upload a malicious file (e.g. a PHP script, an HTML file with XSS, an executable) disguised as a curriculum PDF.

The `edugraph-architecture.docx` security checklist explicitly requires magic-byte validation at the server side.

## Decision

The curriculum upload handler (`backend/internal/curriculum/handler/handler.go`) validates uploaded files by reading the first bytes of the actual file content and comparing them against known magic-byte signatures:

- PDF: `%PDF` (bytes `25 50 44 46`)
- DOCX: PK zip header (`50 4B 03 04`) — DOCX files are ZIP archives

This is implemented in `sniffCurriculumMime()`. The `Content-Type` header is ignored for validation purposes (it may still be logged for diagnostics).

Files that don't match either magic signature are rejected with HTTP 400 before the file is stored or any job is created.

## Consequences

**Good:**
- An attacker cannot disguise a malicious file as a PDF by setting a header.
- Protection works regardless of browser, HTTP client, or API consumer.
- Consistent with the `edugraph-architecture.docx` security hardening checklist.

**Bad:**
- Magic-byte sniffing only validates the file header, not the entire file. A specially crafted file with a valid PDF header followed by malicious content would pass this check (though the PyMuPDF parser would then fail to parse it as a valid PDF).
- The check must be updated if new file types (e.g. PPTX, ODT) are supported in the future.
- Reading the first bytes requires the file to be partially loaded into memory before storage; for very large files this adds a small overhead.
