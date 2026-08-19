variable "vpc_cidr" {
  default = "10.0.0.0/16"
}

variable "azs" {
  type    = list(string)
  default = ["af-south-1a", "af-south-1b", "af-south-1c"]
}

variable "cluster_name" {
  default = "edugraph-production"
}

variable "cluster_version" {
  default = "1.30"
}

variable "s3_bucket_name" {
  default = "edugraph-production-files"
}

variable "cdn_origin_domain" {
  description = "ALB DNS name for CDN origin"
}

variable "cdn_aliases" {
  type    = list(string)
  default = []
}

variable "cdn_acm_certificate_arn" {
  description = "ACM certificate ARN in us-east-1 for CDN aliases"
  default     = ""
}

variable "alb_arn" {
  description = "ARN of the ALB for WAF association"
}
