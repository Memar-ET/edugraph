# School Box

The offline-first edge deployment: the entire EduGraph AI platform —
including AI inference — running on local hardware inside a school, with
zero internet dependency after initial setup. This is the PRD's core
differentiator (Section 7.1): over 60% of Ethiopian public schools have
no reliable internet connection, and a purely cloud-hosted system
excludes exactly the schools that need this most.

## Hardware baseline (assumption, not PRD fact — read this)

The PRD's own hardware-spec section (`edugraph-architecture.docx`
§7.2) is empty — no CPU/RAM/storage numbers were ever specified. Every
config file in this directory is tuned against an **explicit assumption**
made here, not a PRD requirement:

| Resource | Assumed | Why |
|---|---|---|
| CPU | 4–8 cores | Postgres + Neo4j + Redis + Go API + Python AI service + Ollama, five real services sharing one box |
| RAM | 16 GB | The 7B-parameter local LLM (`qwen2.5:7b-instruct-q4_K_M`, ~4.7 GB on disk, needs headroom beyond that to run) alongside everything else; going lower starts forcing real tradeoffs between the LLM and the databases |
| Storage | 256 GB SSD, room to grow | Postgres + Neo4j data grow with enrollment; Ollama model weights; the sync outbox buffers changes for the full offline duration the PRD targets |
| Network | LAN only, no assumed internet | The entire point of this deployment |

**If you're actually procuring hardware for a real School Box, confirm
these numbers against real budget/availability constraints and update
this table and the config files below — don't treat 16 GB as gospel.**
A typical "mini PC" / NUC-class machine meets this baseline; a Raspberry
Pi does not (not enough RAM to run a 7B model alongside three database
engines).

## What's here

| Path | Purpose |
|---|---|
| `compose/docker-compose.yml` | The actual service definitions (api, ai-service, frontend, postgres, neo4j, redis, ollama, sync-agent, caddy, watchtower) |
| `config/Caddyfile` | LAN-facing reverse proxy — routes `/api/v1/*` and `/ws/*` to the Go API, everything else to the frontend SPA. See its own header comment for why `auto_https off`. |
| `config/postgres.conf` | Tuned for the hardware baseline above — see its own header for the actual numbers and reasoning |
| `config/redis.conf` | Persistence + eviction tuned for a box that can lose power without warning and has a fixed memory budget |
| `config/ollama.conf` | Environment-style config (Ollama has no config-file format; Docker Compose's `env_file` doesn't care about the `.conf` extension) controlling model concurrency/memory |
| `scripts/install.sh` | First-time setup on fresh hardware |
| `scripts/backup.sh` | Backs up Postgres + Neo4j data (the sync-agent's outbox means the cloud has a copy of most student-generated data, but curriculum content approved locally and anything not yet synced does not) |
| `scripts/update.sh` | Manual pull-and-restart (Watchtower already does this automatically nightly at 2 AM — see `docker-compose.yml` — this is for an operator who wants to update on demand, or on a box where Watchtower itself is disabled) |
| `scripts/reset.sh` | Destructive factory-reset — wipes all local data. Requires explicit confirmation; see the script's own safety checks. |
| `scripts/health-check.sh` | Checks every service (queries the sync-agent's own `/health/sync` endpoint — see `sync-agent/internal/health/health.go` — for sync status specifically) |
| `sync-agent/` | The offline-sync client (checklist 9.1) — drains the local `sync.outbox` to the cloud and applies pulled changes back. See its own package docs. |

## Deployment

```bash
cd school-box
cp ../.env.school-box.example ../.env.school-box
# edit .env.school-box: set real SCHOOL_BOX_ID, SCHOOL_ID, passwords, CLOUD_SYNC_URL
./scripts/install.sh
```

`install.sh` pulls images, brings the stack up, waits for Postgres/Neo4j
to report healthy, and runs a first health check. See the script itself
for exactly what it does — it's meant to be read, not just trusted.

## Offline behavior

Everything in `compose/docker-compose.yml` runs entirely on the LAN once
started — no service here reaches out to the internet during normal
operation. The only things that need connectivity at all:

- **`sync-agent`** — periodically (`SYNC_INTERVAL_MINUTES`, default 360)
  tries to reach `CLOUD_SYNC_ENDPOINT`. If it can't, it just tries again
  next interval — see `sync-agent/internal/sync/agent.go`, this was
  built and verified to never block local operation on connectivity.
- **`watchtower`** — checks for updated images nightly. Also non-blocking
  if unreachable; the box keeps running its current images.
- **First-time `ollama pull`** of the model weights — after that, all
  inference is local (see `docs/architecture/ai-models-local-vs-cloud.md`
  for the local-first LLM decision this box's AI features depend on).

A School Box that's never had internet access at all still works for
everything except the very first Ollama model pull, which must happen
at least once (pre-loaded onto the box before shipping it to a school
with no connectivity, if needed — see `install.sh`'s comments).
