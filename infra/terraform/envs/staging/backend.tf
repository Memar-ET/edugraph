terraform {
  backend "s3" {
    bucket         = "edugraph-terraform-state"
    key            = "staging/terraform.tfstate"
    region         = "af-south-1"
    encrypt        = true
    kms_key_id     = "alias/edugraph-terraform"
    dynamodb_table = "edugraph-terraform-lock"
  }
}
