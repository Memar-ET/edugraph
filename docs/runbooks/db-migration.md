# Runbook: Database Migrations (Flyway + Supabase)

**Applies to:** Running Postgres migrations against the hosted Supabase project, and the local-fallback container  
**Migration tool:** Flyway 10 (standalone Docker image, not the compose `flyway` service for Supabase)

---

## Important constraints

- **Migrations are append-only.** Never edit a merged migration. Add a new `V{n+1}__description.sql` instead.
- **Never `DROP` a column or table.** Additive schema changes only (new columns must be nullable or have a default).
- **Strictly sequential versioning.** The next migration must be `V{current_max + 1}`. Check before creating:

```bash
# Find current max:
ls backend/db/migrations/ | sort -t_ -k1 | tail -5
```

As of 2026-08-15, the latest is **V049**.

---

## Run a migration against Supabase (standard)

Supabase requires:
- The **session pooler** host (not the direct-connect host, which is IPv6-only)
- The pooler user format: `postgres.<project-ref>`
- `sslmode=require`

Read the current values from `.env`:

```powershell
$env:PGHOST = (Get-Content .env | Select-String "^POSTGRES_HOST").ToString().Split("=")[1]
$env:PGUSER = (Get-Content .env | Select-String "^POSTGRES_USER").ToString().Split("=")[1]
$env:PGPASS = (Get-Content .env | Select-String "^POSTGRES_PASSWORD").ToString().Split("=")[1]
$env:PGDB   = (Get-Content .env | Select-String "^POSTGRES_DB").ToString().Split("=")[1]
```

Run Flyway via the standalone Docker image (**use PowerShell, not Git Bash** — MSYS path translation mangles the repo's space-containing directory name):

```powershell
$migrationsPath = (Resolve-Path "backend\db\migrations").Path
docker run --rm `
  -e "FLYWAY_URL=jdbc:postgresql://${env:PGHOST}/${env:PGDB}?sslmode=require" `
  -e "FLYWAY_USER=${env:PGUSER}" `
  -e "FLYWAY_PASSWORD=${env:PGPASS}" `
  -v "${migrationsPath}:/flyway/sql" `
  flyway/flyway:10 `
  -locations=filesystem:/flyway/sql migrate
```

Expected final output:
```
Successfully applied 1 migration to schema "public", now at version v049 (execution time 00:02.123s)
```

### Validate (dry-run before applying)

```powershell
docker run --rm `
  ... (same env/volume flags) `
  flyway/flyway:10 -locations=filesystem:/flyway/sql validate
```

---

## Run a migration against the local-fallback Postgres

The compose `flyway` service targets the local-fallback Postgres container (profile `local-db`):

```bash
docker compose --profile local-db up -d postgres
docker compose --profile local-db run --rm flyway migrate
```

---

## Neo4j migrations

Neo4j schema changes live in `backend/db/neo4j/migrations/{n}_{description}.cypher`. There is no automated runner — apply manually:

```bash
docker compose exec neo4j cypher-shell -u neo4j -p "${NEO4J_PASSWORD}" \
  --file /path/to/migration.cypher
```

---

## Create a new migration

1. Create `backend/db/migrations/V{n}__description.sql` (use the next sequential integer; snake_case description)
2. Write additive DDL only (no `DROP`, no `ALTER COLUMN ... TYPE` that could lose data)
3. Test locally first:
   ```bash
   docker compose --profile local-db run --rm flyway validate
   docker compose --profile local-db run --rm flyway migrate
   ```
4. Apply to Supabase using the PowerShell command above
5. Verify with `flyway info` that the migration applied cleanly

---

## Recovery: migration conflict (V012 pattern)

If two migrations were merged with the same version number:

1. Identify which one was applied first by checking `flyway_schema_history` in Postgres:
   ```sql
   SELECT version, description, installed_on, success
   FROM flyway_schema_history
   ORDER BY installed_rank DESC LIMIT 10;
   ```
2. Rename the unapplied conflicting migration to the next available version number
3. Re-run `flyway migrate`

See commit `7cddb81` ("fix: resolve V012 migration collision") for an example.

---

## Recovery: migration failed mid-way

Flyway wraps each migration in a transaction (Postgres DDL is transactional). If a migration fails, it is rolled back in full — no partial state is left. Safe to fix the SQL and re-run.

Exception: `CREATE INDEX CONCURRENTLY` cannot run inside a transaction. If a migration uses this, and it fails, check for an invalid index:

```sql
SELECT schemaname, tablename, indexname, indisvalid
FROM pg_indexes
JOIN pg_class c ON c.relname = indexname
JOIN pg_index i ON i.indexrelid = c.oid
WHERE NOT i.indisvalid;
```

Drop the invalid index and re-run.

---

## Supabase `app_storage` schema note

Supabase provisions its own `storage` schema (for the Storage product). Our file storage uses `app_storage` (not `storage`) — see `ADR-009`. Never create objects in the `storage` schema.

---

## Connection string reference

| Environment | POSTGRES_HOST | POSTGRES_USER | POSTGRES_SSLMODE |
|---|---|---|---|
| Supabase (prod/staging) | `aws-0-<region>.pooler.supabase.com` | `postgres.<project-ref>` | `require` |
| Local Docker (fallback) | `postgres` (service name) | `edugraph` | `disable` |
| School-Box | `postgres` (service name) | `edugraph` | `disable` |
