# EduGraph Infrastructure

AWS + Kubernetes deployment for EduGraph AI. Region: **af-south-1** (Cape Town).

## Layout

```
infra/
├── terraform/
│   ├── shared/          # Provider + version pins (shared across envs)
│   ├── modules/         # Reusable modules: vpc, eks, redis, s3, cdn, vault, waf
│   └── envs/
│       ├── staging/     # staging stack (backend: S3 key staging/)
│       └── production/  # production stack (backend: S3 key production/)
├── helm/
│   └── edugraph/        # Single Helm chart for api + ai-service + frontend
└── scripts/             # setup.sh, rotate-secrets.sh, destroy.sh
```

## Terraform Quick Start

Prerequisites: Terraform ≥ 1.7, AWS CLI authenticated for af-south-1.

```bash
# One-time: create S3 state bucket and DynamoDB lock table
bash infra/scripts/setup.sh

# Staging
cd infra/terraform/envs/staging
terraform init
terraform plan
terraform apply

# Production
cd infra/terraform/envs/production
terraform init
terraform plan -var="cdn_origin_domain=<alb-dns>" -var="alb_arn=<alb-arn>"
terraform apply
```

### Key design decisions

| Decision | Rationale |
|---|---|
| No RDS — Supabase session pooler | Zero-ops Postgres with pgvector and free-tier; IPv6-only direct connect blocked on most networks so session pooler required |
| Single-node ElastiCache (no cluster mode) | `BRPOP` workers require a single Redis endpoint; cluster mode breaks this |
| SPOT EKS nodes (t3.medium, min 2 / max 6) | 60-70% cost saving vs. on-demand; stateless pods tolerate spot interruptions |
| No HashiCorp Vault | AWS Secrets Manager + IRSA (EKS pod IAM) is sufficient and avoids operating another service |
| af-south-1 region | Closest AWS region to Ethiopia; data-sovereignty preference |

## Helm Chart

The `edugraph` chart deploys three Deployments (api, ai-service, frontend), Services, HPAs, Ingress, NetworkPolicies, and ServiceAccounts.

```bash
# Staging
helm upgrade --install edugraph infra/helm/edugraph \
  -n edugraph --create-namespace \
  -f infra/helm/edugraph/values.staging.yaml \
  --set image.api.tag=$SHA \
  --set image.ai.tag=$SHA \
  --set image.frontend.tag=$SHA \
  --atomic --timeout 8m

# Production (canary → full, as CI does it)
helm upgrade --install edugraph infra/helm/edugraph \
  -n edugraph \
  -f infra/helm/edugraph/values.production.yaml \
  --set image.api.tag=$SHA --set canary.weight=25 --atomic --timeout 10m
# ... smoke tests ...
helm upgrade edugraph infra/helm/edugraph \
  -n edugraph \
  -f infra/helm/edugraph/values.production.yaml \
  --set image.api.tag=$SHA --set canary.weight=100 --atomic --timeout 10m
```

## Secrets

Secrets live in AWS Secrets Manager under `edugraph/<env>/{postgres,redis,neo4j,app}`. Pods read them via IRSA (IAM role bound to Kubernetes ServiceAccount via OIDC). In practice: use the External Secrets Operator or AWS Secrets & Config Provider (ASCP) CSI driver to project them as a Kubernetes Secret named `edugraph-secrets`.

```bash
# Rotate all secrets
bash infra/scripts/rotate-secrets.sh <env>
```

## CI/CD

- **`.github/workflows/ci.yml`** — runs on every PR: Go lint/test, frontend lint/type-check/test, Python lint/test, Docker build.
- **`.github/workflows/deploy-staging.yml`** — deploys on push to `develop`.
- **`.github/workflows/deploy-production.yml`** — canary (25%) → wait 10 min → promote (100%) on push to `main`.

Flyway migrations run as a Kubernetes Job before `helm upgrade` in both pipelines.
