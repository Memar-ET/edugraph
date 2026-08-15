# ── Postgres / Supabase ───────────────────────────────────────────────────────
#
# EduGraph uses Supabase (hosted Postgres) rather than a self-managed RDS
# instance. This module does NOT create any AWS database resources.
#
# Why Supabase instead of RDS?
#   - Zero-ops managed Postgres with built-in pgvector support
#   - Free-tier sufficient for current scale; upgrading is a slider, not a
#     migration event
#   - pgvector (used for CLO/topic embeddings) is pre-installed on all plans
#   - Integrated auth studio and Storage product (used as `app_storage` schema)
#
# Connection requirements (see CLAUDE.md "Postgres is now Supabase"):
#   - Must use the SESSION pooler, NOT the direct host (direct host is IPv6-
#     only; most networks have no outbound IPv6 route)
#   - POSTGRES_USER must be postgres.<project-ref>, NOT plain postgres
#   - POSTGRES_SSLMODE must be "require"
#   - POSTGRES_MAX_CONNS=15 (pooler budget is shared across all app instances)
#   - Passwords may contain @/?/& — DSN construction must percent-encode them
#
# The AWS Secrets Manager secret created by this module stores the Supabase
# credentials so the EKS pods can fetch them at runtime via IRSA, avoiding
# hardcoded secrets in Helm values or Kubernetes Secrets.

resource "aws_secretsmanager_secret" "postgres" {
  name                    = "edugraph/${var.environment}/postgres"
  description             = "Supabase connection credentials for ${var.environment}"
  recovery_window_in_days = 7
}

resource "aws_secretsmanager_secret_version" "postgres" {
  secret_id = aws_secretsmanager_secret.postgres.id
  secret_string = jsonencode({
    POSTGRES_HOST     = var.supabase_host
    POSTGRES_USER     = var.supabase_user
    POSTGRES_PASSWORD = var.supabase_password
    POSTGRES_DB       = var.supabase_db
    POSTGRES_SSLMODE  = "require"
    POSTGRES_MAX_CONNS = "15"
  })
}
