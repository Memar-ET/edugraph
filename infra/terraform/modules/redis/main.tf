# Single-node ElastiCache Redis.
#
# Cluster mode (sharding) is deliberately disabled: the ai-service's BRPOP
# workers use Redis list queues where each job is destructively consumed by
# exactly one consumer. Redis Cluster routes commands by key-slot, so a BLPOP
# across a sharded cluster would require all keys to hash to the same slot,
# making horizontal sharding useless for this pattern. A single primary node
# (with optional replica for read HA) is the correct topology here.

resource "aws_elasticache_subnet_group" "main" {
  name       = "edugraph-${var.environment}-redis"
  subnet_ids = var.subnet_ids
}

resource "aws_elasticache_replication_group" "main" {
  replication_group_id       = "edugraph-${var.environment}"
  description                = "EduGraph Redis — job queues and refresh-token store"
  node_type                  = var.node_type
  engine_version             = var.engine_version
  port                       = 6379
  num_cache_clusters         = 1 # single primary, no read replicas
  automatic_failover_enabled = false
  multi_az_enabled           = false
  at_rest_encryption_enabled = true
  transit_encryption_enabled = false # TLS adds latency; clients are in same VPC
  subnet_group_name          = aws_elasticache_subnet_group.main.name
  security_group_ids         = var.security_group_ids

  log_delivery_configuration {
    destination      = "/edugraph/${var.environment}/redis/slow-log"
    destination_type = "cloudwatch-logs"
    log_format       = "text"
    log_type         = "slow-log"
  }
}
