variable "environment" {
  type = string
}

variable "origin_domain_name" {
  description = "ALB DNS name or S3 bucket regional domain name to use as CloudFront origin"
  type        = string
}

variable "aliases" {
  description = "Custom domain aliases (requires ACM certificate in us-east-1)"
  type        = list(string)
  default     = []
}

variable "acm_certificate_arn" {
  description = "ACM certificate ARN in us-east-1 (required if aliases are set)"
  type        = string
  default     = ""
}

variable "price_class" {
  description = "CloudFront price class"
  type        = string
  default     = "PriceClass_100" # US, Canada, Europe — cheapest; add Africa when needed
}
