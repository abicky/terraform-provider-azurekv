package provider_test

import (
	"fmt"
	"math/rand"
	"os"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/abicky/terraform-provider-azurekv/internal/provider"
)

var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

type keyVault struct {
	id             string
	subscriptionID string
	name           string
}

// testAccProtoV6ProviderFactories is used to instantiate a provider during acceptance testing.
// The factory function is called for each Terraform CLI command to create a provider
// server that the CLI can connect to and interact with.
var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"azurekv": providerserver.NewProtocol6WithError(provider.New("test")()),
}

func testAccPreCheck(t *testing.T) {
	for _, envVar := range []string{"ARM_SUBSCRIPTION_ID"} {
		if v := os.Getenv(envVar); v == "" {
			t.Fatal(envVar + " must be set for acceptance tests")
		}
	}
}

func testCheckResourceAttrPairs(nameFirst, nameSecond string, keys []string) resource.TestCheckFunc {
	funcs := make([]resource.TestCheckFunc, 0, len(keys))
	for _, key := range keys {
		funcs = append(funcs, resource.TestCheckResourceAttrPair(nameFirst, key, nameSecond, key))
	}
	return resource.ComposeAggregateTestCheckFunc(funcs...)
}

func providersConfig(randomName string) string {
	config := `
provider "azurerm" {
  features {}
}

provider "azurekv" {}
`

	if keyVaultID := os.Getenv("KEY_VAULT_ID"); keyVaultID != "" {
		// Using existing Key Vault shortens test time, since destroying a Key Vault takes about 10 minutes
		return fmt.Sprintf(`%s

locals {
  key_vault_id = %q
}
`, config, keyVaultID)
	}

	return fmt.Sprintf(`%s

locals {
  key_vault_id = azurerm_key_vault.test.id
}

data "azurerm_client_config" "current" {}

resource "azurerm_resource_group" "test" {
  name     = "azurekv-acctest-%[2]s"
  location = "East US"
}

resource "azurerm_key_vault" "test" {
  name                       = %[2]q
  location                   = azurerm_resource_group.test.location
  resource_group_name        = azurerm_resource_group.test.name
  tenant_id                  = data.azurerm_client_config.current.tenant_id
  sku_name                   = "standard"

  enable_rbac_authorization = true
}
`, config, randomName)
}

func generateRandomName(n int) string {
	prefix := time.Now().UTC().Format("F20060102T150405")
	b := make([]rune, n-len(prefix))
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return prefix + string(b)
}
