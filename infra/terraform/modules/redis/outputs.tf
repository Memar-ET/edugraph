output "primary_endpoint" {
  value = aws_elasticache_replication_group.main.primary_endpoint_address
}

output "port" {
  value = 6379
}

output "redis_url" {
  value = "redis://${aws_elasticache_replication_group.main.primary_endpoint_address}:6379"
}
