# AWS Secrets Manager module
# EduGraph does not run HashiCorp Vault (see CLAUDE.md "Don't build unused production infra").
# All secrets are stored in AWS Secrets Manager and fetched by pods via IRSA.

resource "aws_secretsmanager_secret" "postgres" {
  name                    = "edugraph/${var.environment}/postgres"
  description             = "Supabase session-pooler credentials"
  recovery_window_in_days = var.environment == "production" ? 30 : 0
}

resource "aws_secretsmanager_secret" "redis" {
  name                    = "edugraph/${var.environment}/redis"
  description             = "ElastiCache Redis endpoint and auth token"
  recovery_window_in_days = var.environment == "production" ? 30 : 0
}

resource "aws_secretsmanager_secret" "neo4j" {
  name                    = "edugraph/${var.environment}/neo4j"
  description             = "Neo4j bolt URI, username, and password"
  recovery_window_in_days = var.environment == "production" ? 30 : 0
}

resource "aws_secretsmanager_secret" "app" {
  name                    = "edugraph/${var.environment}/app"
  description             = "Application secrets: JWT keys, Gemini API key"
  recovery_window_in_days = var.environment == "production" ? 30 : 0
}

# IAM policy that allows EKS pods (via IRSA) to read these secrets
resource "aws_iam_policy" "secrets_read" {
  name        = "edugraph-${var.environment}-secrets-read"
  description = "Allow EKS service accounts to read EduGraph secrets from Secrets Manager"

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Action = [
          "secretsmanager:GetSecretValue",
          "secretsmanager:DescribeSecret",
        ]
        Resource = [
          aws_secretsmanager_secret.postgres.arn,
          aws_secretsmanager_secret.redis.arn,
          aws_secretsmanager_secret.neo4j.arn,
          aws_secretsmanager_secret.app.arn,
        ]
      }
    ]
  })
}
