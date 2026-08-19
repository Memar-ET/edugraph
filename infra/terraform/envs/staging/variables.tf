variable "vpc_cidr" {
  default = "10.1.0.0/16"
}

variable "azs" {
  type    = list(string)
  default = ["af-south-1a", "af-south-1b", "af-south-1c"]
}

variable "cluster_name" {
  default = "edugraph-staging"
}

variable "cluster_version" {
  default = "1.30"
}

variable "s3_bucket_name" {
  default = "edugraph-staging-files"
}

variable "cdn_origin_domain" {
  description = "ALB DNS name for CDN origin; leave empty to skip CDN"
  default     = ""
}

variable "cdn_aliases" {
  type    = list(string)
  default = []
}

variable "cdn_acm_certificate_arn" {
  default = ""
}
