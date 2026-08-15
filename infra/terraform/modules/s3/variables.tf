variable "environment" {
  type = string
}

variable "bucket_name" {
  description = "S3 bucket name (must be globally unique)"
  type        = string
}

variable "versioning_enabled" {
  type    = bool
  default = true
}

variable "lifecycle_expire_days" {
  description = "Days before non-current versions expire (0 = disabled)"
  type        = number
  default     = 90
}
