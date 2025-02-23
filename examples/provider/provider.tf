terraform {
  required_version = ">=1.11"
  required_providers {
    azurekv = {
      source = "abicky/azurekv"
    }
  }
}

provider "azurekv" {
  subscription_id = var.subscription_id
}

resource "azurekv_secret" "example" {
  name             = "example-secret"
  key_vault_id     = var.key_vault_id
  value_wo         = "This value is manged outside of Terraform"
  value_wo_version = 1
}
