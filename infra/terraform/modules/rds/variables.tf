variable "environment" {
  type = string
}
# NOTE: Postgres is hosted on Supabase — this module provisions NO RDS
# instances. It exists to document the Supabase connection parameters
# that the application expects and to export them as Terraform outputs
# so the env-level modules can wire them into the EKS secrets.

variable "supabase_host" {
  description = "Supabase session-pooler host (aws-0-<region>.pooler.supabase.com)"
  type        = string
  sensitive   = true
}

variable "supabase_user" {
  description = "Supabase pooler user (postgres.<project-ref>)"
  type        = string
  sensitive   = true
}

variable "supabase_password" {
  description = "Supabase database password"
  type        = string
  sensitive   = true
}

variable "supabase_db" {
  description = "Database name"
  type        = string
  default     = "edugraph"
}
