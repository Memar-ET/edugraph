# ADR-009: `app_storage` Schema (Not `storage`)

Date: 2026-08-10  
Status: Accepted

## Context

Migration V013 (`create_local_storage`) originally targeted `storage.local_files`. When the project migrated to Supabase (ADR-003), every fresh Supabase project pre-provisions a `storage` schema for its own Storage product (buckets, objects, etc.), owned by `supabase_admin`. The `postgres` role has USAGE on this schema but not CREATE. Migration V013 therefore failed on every fresh Supabase project with a permission error.

## Decision

Migration V013 was edited in place to target `app_storage.local_files` instead of `storage.local_files`. This is the **one documented exception** to the "never edit a merged migration" rule in this repository, justified by two facts:

1. V013 had never successfully applied on any target environment — it fails identically on every fresh Supabase project. There was no live data in the old table to reconcile.
2. Postgres DDL is transactional. The failed V013 attempt left no partial state; the schema simply didn't exist.

All references in application code were updated:
- `backend/pkg/storage/postgres.go` — SQL queries
- `ai-service/app/db/postgres.py` — `fetch_file_bytes` query

## Consequences

**Good:**
- V013 now applies cleanly on every fresh Supabase project.
- `app_storage` clearly signals "our schema" vs. Supabase's built-in `storage`.
- The naming collision with Supabase is permanently resolved.

**Bad:**
- Any existing documentation or scripts that reference `storage.local_files` must be updated.
- The in-place edit of a migration creates an exception to a strong policy. The policy still applies to all subsequent migrations (V014 onward) — this was a one-time fix for a migration that had never successfully run.
