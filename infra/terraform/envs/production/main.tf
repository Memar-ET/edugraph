module "vpc" {
  source      = "../../modules/vpc"
  environment = "production"
  vpc_cidr    = var.vpc_cidr
  azs         = var.azs
}

module "eks" {
  source          = "../../modules/eks"
  environment     = "production"
  cluster_name    = var.cluster_name
  vpc_id          = module.vpc.vpc_id
  subnet_ids      = module.vpc.private_subnet_ids
  node_sg_id      = module.vpc.eks_nodes_sg_id
  cluster_version = var.cluster_version
}

module "redis" {
  source      = "../../modules/redis"
  environment = "production"
  subnet_ids  = module.vpc.private_subnet_ids
  sg_id       = module.vpc.redis_sg_id
  node_type   = "cache.r7g.large"
}

module "s3" {
  source      = "../../modules/s3"
  environment = "production"
  bucket_name = var.s3_bucket_name
}

module "vault" {
  source      = "../../modules/vault"
  environment = "production"
}

module "cdn" {
  source      = "../../modules/cdn"
  environment = "production"

  origin_domain_name  = var.cdn_origin_domain
  aliases             = var.cdn_aliases
  acm_certificate_arn = var.cdn_acm_certificate_arn
  price_class         = "PriceClass_100"
}

module "waf" {
  source      = "../../modules/waf"
  environment = "production"
  alb_arn     = var.alb_arn
}
