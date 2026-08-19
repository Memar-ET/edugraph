# ADR-003: Supabase Session Pooler (Not Direct Connect)

Date: 2026-08-10  
Status: Accepted

## Context

Postgres was migrated from a local Docker container to a hosted Supabase project on 2026-08-10. The standard Supabase connection string points to `db.<ref>.supabase.co:5432`, the direct-connect host.

On the development network (and most networks without explicit IPv6 routing), this hostname resolves to an IPv6 address only. The local machine has no outbound IPv6 route and cannot reach it. Every direct-connect attempt timed out.

Additionally, Supabase-generated passwords routinely contain `@`, `?`, and `&` — characters that break naive DSN string construction when interpolated directly into a URL.

## Decision

1. **Use the session pooler**, not the direct-connect host:
   - `POSTGRES_HOST=aws-0-<region>.pooler.supabase.com` (port 5432, session mode)
   - NOT the transaction-mode pooler on port 6543 — pgx and asyncpg both use server-side prepared statements by default, which are not supported in transaction mode.

2. **Username format:** `POSTGRES_USER=postgres.<project-ref>` (required by the pooler).

3. **Percent-encode credentials** in DSN construction. Both `PostgresConfig.DSN()` (Go, `pkg/config/config.go`) and `Settings.POSTGRES_DSN` (Python, `app/core/config.py`) now URL-encode the user and password fields.

4. **TLS:** `POSTGRES_SSLMODE=require`. The local-fallback Postgres container still uses `disable`.

5. **Connection budget:** `POSTGRES_MAX_CONNS=15` (down from 80). The session pooler's connection budget is shared across all processes hitting the project.

6. **`app_storage` schema:** Supabase pre-provisions a `storage` schema for its own Storage product before any migration runs. Our dev file storage was renamed from `storage.local_files` to `app_storage.local_files` to avoid the collision. Migration V013 was edited in place (the one exception to "never edit a merged migration") because V013 had never successfully applied on any Supabase project and Postgres DDL is transactional, so the failed attempt left no trace to reconcile.

## Consequences

**Good:**
- Works on IPv4-only development networks.
- Credentials with special characters are correctly URL-encoded.
- The `app_storage` rename is permanent and unambiguous.

**Bad:**
- Session pooler URL/username is less obvious than direct connect; new developers may use the wrong string from the Supabase dashboard.
- `POSTGRES_MAX_CONNS=15` is a shared budget — a future background job surge could exhaust it.
- The in-place edit of V013 is a one-time exception; the "never edit merged migrations" rule applies to all subsequent migrations.
