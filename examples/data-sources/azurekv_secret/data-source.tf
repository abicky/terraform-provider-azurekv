data "azurekv_vault_secret" "example" {
  name         = "secret-sauce"
  key_vault_id = data.azurerm_key_vault.existing.id
}

output "secret_resource_id" {
  value = data.azurekv_secret.example.resource_id
}
