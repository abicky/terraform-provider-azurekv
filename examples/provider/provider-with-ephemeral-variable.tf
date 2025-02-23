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

variable "secret_value" {
  ephemeral = true
  sensitive = true
}

resource "azurekv_secret" "example" {
  name             = "example-secret"
  key_vault_id     = var.key_vault_id
  value_wo         = var.secret_value
  value_wo_version = 1
}
