module "vpc" {
  source      = "../../modules/vpc"
  environment = "staging"
  vpc_cidr    = var.vpc_cidr
  azs         = var.azs
}

module "eks" {
  source          = "../../modules/eks"
  environment     = "staging"
  cluster_name    = var.cluster_name
  vpc_id          = module.vpc.vpc_id
  subnet_ids      = module.vpc.private_subnet_ids
  node_sg_id      = module.vpc.eks_nodes_sg_id
  cluster_version = var.cluster_version
}

module "redis" {
  source      = "../../modules/redis"
  environment = "staging"
  subnet_ids  = module.vpc.private_subnet_ids
  sg_id       = module.vpc.redis_sg_id
  node_type   = "cache.t3.micro"
}

module "s3" {
  source      = "../../modules/s3"
  environment = "staging"
  bucket_name = var.s3_bucket_name
}

module "vault" {
  source      = "../../modules/vault"
  environment = "staging"
}

# CDN is optional in staging — set var.cdn_origin_domain to enable
module "cdn" {
  count       = var.cdn_origin_domain != "" ? 1 : 0
  source      = "../../modules/cdn"
  environment = "staging"

  origin_domain_name  = var.cdn_origin_domain
  aliases             = var.cdn_aliases
  acm_certificate_arn = var.cdn_acm_certificate_arn
  price_class         = "PriceClass_100"
}
