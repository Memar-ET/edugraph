variable "environment" {
  type = string
}

variable "alb_arn" {
  description = "ARN of the Application Load Balancer to associate the WAF WebACL with"
  type        = string
}
