output "postgres_secret_arn" {
  value = aws_secretsmanager_secret.postgres.arn
}

output "postgres_host" {
  value     = var.supabase_host
  sensitive = true
}

output "postgres_user" {
  value     = var.supabase_user
  sensitive = true
}
