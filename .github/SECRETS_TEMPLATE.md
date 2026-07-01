# Required GitHub Secrets

Configure these in: Settings → Secrets and variables → Actions

## AWS
- `AWS_ACCESS_KEY_ID`        — IAM key with ECR push + EKS deploy permissions
- `AWS_SECRET_ACCESS_KEY`    — IAM secret
- `ECR_REGISTRY`             — e.g. 123456789.dkr.ecr.af-south-1.amazonaws.com

## Kubernetes / EKS  
- `KUBECONFIG_STAGING`       — base64-encoded kubeconfig for staging cluster
- `KUBECONFIG_PRODUCTION`    — base64-encoded kubeconfig for prod cluster (protected)

## Notifications
- `SLACK_WEBHOOK_URL`        — Slack incoming webhook for deploy notifications

## Code Quality
- `CODECOV_TOKEN`            — Codecov upload token

## Security
- `SNYK_TOKEN`               — Snyk vulnerability scan token

## Environments
Create two GitHub Environments (Settings → Environments):
1. `staging`    — auto-deploy from develop branch
2. `production` — requires manual approval from senior engineer
