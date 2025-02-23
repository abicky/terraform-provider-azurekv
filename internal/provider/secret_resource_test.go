package provider_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

func TestAccSecretResource_basic(t *testing.T) {
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
			buildTestStep(basicResourceConfig(rn, 1)),
			{
				ResourceName:      "azurekv_secret.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccSecretResource_complete(t *testing.T) {
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
			buildTestStep(completeResourceConfig(rn, 1)),
			{
				ResourceName:      "azurekv_secret.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccSecretResource_update(t *testing.T) {
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
			buildTestStep(basicResourceConfig(rn, 1)),
			{
				ResourceName:      "azurekv_secret.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update the secret value
			buildTestStep(basicResourceConfig(rn, 2)),
			// Update the properties
			buildTestStep(completeResourceConfig(rn, 2)),
		},
	})
}

func buildTestStep(config string) resource.TestStep {
	return resource.TestStep{
		Config: config,
		Check: testCheckResourceAttrPairs("azurekv_secret.test", "data.azurerm_key_vault_secret.test", []string{
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
		ConfigStateChecks: []statecheck.StateCheck{
			statecheck.ExpectKnownValue(
				"azurekv_secret.test",
				tfjsonpath.New("value_wo"),
				knownvalue.Null(),
			),
		},
	}
}

func basicResourceConfig(resourceSuffix string, version int) string {
	return fmt.Sprintf(`%s

resource "azurekv_secret" "test" {
  name         = "secret-name-%s"
  key_vault_id = local.key_vault_id

  value_wo         = "secret-value"
  value_wo_version = %d
}

data "azurerm_key_vault_secret" "test" {
  name         = azurekv_secret.test.name
  key_vault_id = azurekv_secret.test.key_vault_id
}
`, providersConfig(resourceSuffix), resourceSuffix, version)
}

func completeResourceConfig(resourceSuffix string, version int) string {
	return fmt.Sprintf(`%s

resource "azurekv_secret" "test" {
  name         = "secret-name-%s"
  key_vault_id = local.key_vault_id

  value_wo         = "secret-value"
  value_wo_version = %d

  not_before_date = "2025-01-23T01:23:45Z"
  expiration_date = "2026-01-23T01:23:45Z"

  tags = {
    environment = "test"
  }
}

data "azurerm_key_vault_secret" "test" {
  name         = azurekv_secret.test.name
  key_vault_id = azurekv_secret.test.key_vault_id
}
`, providersConfig(resourceSuffix), resourceSuffix, version)
}
