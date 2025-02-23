package provider_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccSecretDataSource_basic(t *testing.T) {
	t.Parallel()

	rn := generateRandomName(23)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		ExternalProviders: map[string]resource.ExternalProvider{
			"azurerm": {
				Source:            "hashicorp/azurerm",
				VersionConstraint: "~> 4.0",
			},
		},
		Steps: []resource.TestStep{
			{
				Config: basicDataSourceConfig(rn),
				Check: testCheckResourceAttrPairs("data.azurekv_secret.test", "azurerm_key_vault_secret.test", []string{
					"content_type",
					"expiration_date",
					"id",
					"key_vault_id",
					"name",
					"not_before_date",
					"resource_id",
					"resource_versionless_id",
					"tags",
					"version",
					"versionless_id",
				}),
			},
		},
	})
}

func TestAccSecretDataSource_complete(t *testing.T) {
	t.Parallel()

	rn := generateRandomName(23)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		ExternalProviders: map[string]resource.ExternalProvider{
			"azurerm": {
				Source:            "hashicorp/azurerm",
				VersionConstraint: "~> 4.0",
			},
		},
		Steps: []resource.TestStep{
			{
				Config: completeDataSourceConfig(rn),
				Check: testCheckResourceAttrPairs("data.azurekv_secret.test", "azurerm_key_vault_secret.test", []string{
					"content_type",
					"expiration_date",
					"id",
					"key_vault_id",
					"name",
					"not_before_date",
					"resource_id",
					"resource_versionless_id",
					"tags",
					"version",
					"versionless_id",
				}),
			},
		},
	})
}

func basicDataSourceConfig(resourceSuffix string) string {
	return fmt.Sprintf(`%s

resource "azurerm_key_vault_secret" "test" {
  name         = "secret-name-%s"
  key_vault_id = local.key_vault_id
  value        = "secret-value"
}

data "azurekv_secret" "test" {
  name = azurerm_key_vault_secret.test.name
  key_vault_id = azurerm_key_vault_secret.test.key_vault_id
}
`, providersConfig(resourceSuffix), resourceSuffix)
}

func completeDataSourceConfig(resourceSuffix string) string {
	return fmt.Sprintf(`%s

resource "azurerm_key_vault_secret" "test" {
  name         = "secret-name-%s"
  key_vault_id = local.key_vault_id
  value        = "secret-value"

  not_before_date = "2025-01-23T01:23:45Z"
  expiration_date = "2026-01-23T01:23:45Z"

  tags = {
    environment = "test"
  }
}

data "azurekv_secret" "test" {
  name = azurerm_key_vault_secret.test.name
  key_vault_id = azurerm_key_vault_secret.test.key_vault_id
}
`, providersConfig(resourceSuffix), resourceSuffix)
}
