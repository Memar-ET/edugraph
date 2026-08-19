# Runbook: School-Box Setup

**Applies to:** First-time setup of a School-Box (offline-capable local node) at a new school  
**Estimated time:** 30–60 minutes depending on hardware

---

## Prerequisites

- A machine at the school running Ubuntu 22.04 LTS (or Debian 11+), or a compatible ARM board
- Docker Engine ≥ 24 and Docker Compose v2 installed
- An `.env` file provisioned by the central IT team (see "Provision credentials" below)
- Network access to Supabase at least once during initial setup (subsequent operation can be offline)

---

## 1. Clone the repository

```bash
git clone https://github.com/Memar-ET/edugraph.git
cd edugraph
```

If the School-Box has no internet access, transfer a pre-built image tarball via USB instead:

```bash
# On a machine with internet:
docker compose build
docker save edugraph-api edugraph-ai-service edugraph-frontend | gzip > edugraph-images.tar.gz
# Transfer to School-Box via USB, then on the School-Box:
docker load < edugraph-images.tar.gz
```

---

## 2. Provision credentials

The central IT team must generate a `.env` file for this school. The School-Box `.env` differs from the cloud `.env` in:

| Variable | Cloud value | School-Box value |
|---|---|---|
| `POSTGRES_HOST` | Supabase pooler host | `postgres` (local container) |
| `POSTGRES_USER` | `postgres.<ref>` | `edugraph` |
| `POSTGRES_DB` | Supabase DB name | `edugraph` |
| `POSTGRES_SSLMODE` | `require` | `disable` |
| `POSTGRES_MAX_CONNS` | `15` | `20` (local, no shared budget) |
| `NEO4J_URI` | cloud Neo4j URI | `bolt://neo4j:7687` |
| `SCHOOL_ID` | — | The school's UUID from `public.schools` |

The JWT keys (`JWT_PRIVATE_KEY`, `JWT_PUBLIC_KEY`) are the same for all nodes (so tokens issued by the cloud are accepted locally, and vice versa). Generate a provisioned `.env` on the cloud admin machine and copy it to the School-Box via USB.

Copy `.env` to the repo root:
```bash
cp /path/to/school.env .env
```

---

## 3. Start the local-db profile

School-Box runs its own local Postgres (not Supabase):

```bash
docker compose --profile local-db up -d postgres flyway
# Wait for Flyway migrations to complete:
docker compose logs flyway --follow
# Expected final line: "Successfully applied N migrations"
```

Then start the rest of the stack:

```bash
docker compose up -d api ai-service neo4j redis frontend
```

---

## 4. Seed school data

If this is a fresh School-Box with no synced data yet, load the initial seed:

```bash
# Apply demo seed (creates roles, school row, demo users):
# This is already applied by flyway via V015__seed_demo_data.sql
# Verify:
docker compose exec postgres psql -U edugraph -c "SELECT role, email FROM public.users LIMIT 10;"
```

Confirm the school's `school_id` matches the `SCHOOL_ID` in `.env`:

```bash
docker compose exec postgres psql -U edugraph -c \
  "SELECT id, name FROM public.schools WHERE id = '$(grep SCHOOL_ID .env | cut -d= -f2)';"
```

---

## 5. Verify Neo4j curriculum graph

If the central curriculum has already been ingested into the cloud Neo4j, you need to replicate it locally:

```bash
# Check current node count:
docker compose exec neo4j cypher-shell -u neo4j -p "${NEO4J_PASSWORD}" \
  "MATCH (n) RETURN labels(n)[0] AS label, count(*) AS count ORDER BY count DESC;"
```

If the local Neo4j is empty (fresh setup), restore from the cloud Neo4j dump:

```bash
# On the cloud machine, export:
docker compose exec neo4j neo4j-admin database dump --to-path=/var/lib/neo4j/data/dumps neo4j
# Copy to School-Box, then import:
docker compose exec neo4j neo4j-admin database load --from-path=/var/lib/neo4j/data/dumps neo4j --overwrite-destination
docker compose restart neo4j
```

For small curriculum datasets, the resync endpoint is faster:

```bash
# This requires the local api to be running and connected to local Postgres:
curl -X POST http://localhost:8080/api/v1/curriculum/prerequisites/resync \
  -H "Authorization: Bearer <ministry_admin_token>"
```

---

## 6. Verify services

```bash
# Health check:
curl http://localhost:8080/health
# Expected: {"status":"ok"}

# Frontend:
curl -I http://localhost:5173
# Expected: HTTP 200

# Neo4j browser:
# Open http://localhost:7474 in a browser
```

Log in with `teacher@edugraph.et` / `password123` (demo data) to confirm login works.

---

## 7. Ongoing sync

The School-Box `api` service runs a background sync worker that pushes outbox rows to the cloud API when connectivity is available. Configure the cloud API endpoint:

```
CLOUD_API_URL=https://<cloud-api-host>
SCHOOL_BOX_SYNC_INTERVAL=300  # seconds
```

Monitor sync status:

```sql
-- Via local Postgres:
SELECT table_name, COUNT(*) AS pending
FROM sync.outbox WHERE synced_at IS NULL
GROUP BY table_name;
```

---

## 8. Troubleshooting

| Symptom | Check |
|---|---|
| `api` exits immediately | `docker compose logs api` — likely `.env` missing or Postgres not ready |
| Flyway migration error | Check migration count vs cloud: `SELECT version FROM schema_version ORDER BY installed_rank DESC LIMIT 5;` |
| Neo4j connection refused | `docker compose ps neo4j` — it may still be warming up; wait 30s |
| Login returns 401 | Confirm `JWT_PUBLIC_KEY` matches the cloud's `JWT_PRIVATE_KEY` pair |
| Sync not uploading | Check `CLOUD_API_URL` in `.env` and that the School-Box has outbound HTTPS access |

For persistent outbox failures, see `docs/runbooks/sync-failure.md`.
