data "azurerm_client_config" "current" {}

resource "azurerm_resource_group" "example" {
  name     = "example-resources"
  location = "West Europe"
}

resource "azurerm_key_vault" "example" {
  name                       = "examplekeyvault"
  location                   = azurerm_resource_group.example.location
  resource_group_name        = azurerm_resource_group.example.name
  tenant_id                  = data.azurerm_client_config.current.tenant_id
  sku_name                   = "premium"
  soft_delete_retention_days = 7

  enable_rbac_authorization = true
}

resource "azurekv_secret" "example" {
  name             = "secret-sauce"
  key_vault_id     = azurerm_key_vault.example.id
  value_wo         = "szechuan"
  value_wo_version = 1
}
