output "postgres_secret_arn" {
  value = aws_secretsmanager_secret.postgres.arn
}

output "redis_secret_arn" {
  value = aws_secretsmanager_secret.redis.arn
}

output "neo4j_secret_arn" {
  value = aws_secretsmanager_secret.neo4j.arn
}

output "app_secret_arn" {
  value = aws_secretsmanager_secret.app.arn
}

output "secrets_read_policy_arn" {
  value = aws_iam_policy.secrets_read.arn
}
