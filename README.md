# Terraform Provider for Auzre Key Vault

The Terraform provider for Azure Key Vault allows you to manage Key Vault secrets without requiring the `Microsoft.KeyVault/vaults/secrets/getSecret/action` permission, by leveraging [write-only arguments](https://developer.hashicorp.com/terraform/language/resources/ephemeral/write-only).

Even if you use [`ignore_changes`](https://developer.hashicorp.com/terraform/language/meta-arguments/lifecycle#ignore_changes) to avoid storing secret values directly in Terraform files, the actual secret values are still written to the state file. Write-only arguments were introduced to solve this problem. However, the [`azurerm_key_vault_secret`](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/key_vault_secret) resource still requires the `Microsoft.KeyVault/vaults/secrets/getSecret/action` permission even if you use write-only arguments. This means you must grant that permission to anyone who runs `terraform plan`.
This provider addresses that limitation.

> [!WARNING]
> Once [hashicorp/terraform-provider-azurerm#29637](https://github.com/hashicorp/terraform-provider-azurerm/pull/29637), which provides the same functionality, is merged, this repository will be archived.


## Usage

See the [provider documentation](https://registry.terraform.io/providers/abicky/azurekv/latest/docs).

## Development

This provider is built on the [Terraform Plugin Framework](https://github.com/hashicorp/terraform-plugin-framework). If you have questions about the framework, see its documentation.

### Requirements

- [Terraform](https://developer.hashicorp.com/terraform/downloads) >= 1.11
- [Go](https://golang.org/doc/install) >= 1.24


### Build and use provider locally

First, build the provider:

```sh
make
```

Then, create a `~/.terraformrc` file:

```terraform
provider_installation {

  dev_overrides {
    "abicky/azurekv" = "/path/to/this-repository/dist"
  }

  # For all other providers, install them directly from their origin provider
  # registries as normal. If you omit this, Terraform will _only_ use
  # the dev_overrides block, and so no other providers will be available.
  direct {}
}
```

With the configuration file, Terraform will use your local build of the provider.


> [!NOTE]
> If you use other providers such as the Terraform provider for Azure, you must run `terraform init` before adding this provider to a Terraform file.
> For more details, see this related issue: [`init` fails for development overrides · Issue #27459 · hashicorp/terraform](https://github.com/hashicorp/terraform/issues/27459).

### Debugging

The following command builds the provider in debug mode and start [`delve`](https://github.com/go-delve/delve):

```sh
make debug
```

For details, see [Debugger-Based Debugging](https://developer.hashicorp.com/terraform/plugin/debugging#debugger-based-debugging).


### Testing

#### Required permissions

The tests depend on both this provider and the Terraform provider for Azure to create resource groups, Key Vaults, and Key Vault secrets.
For this reason, your user, service principal, or workload identity must have the following permissions in the subscription specified by the `ARM_SUBSCRIPTION_ID` environment variable:

* Actions
    - Microsoft.Resources/subscriptions/resourceGroups/delete
    - Microsoft.Resources/subscriptions/resourceGroups/read
    - Microsoft.Resources/subscriptions/resourceGroups/write
    - Microsoft.KeyVault/locations/deletedVaults/purge/action
    - Microsoft.KeyVault/locations/operationResults/read
    - Microsoft.KeyVault/vaults/delete
    - Microsoft.KeyVault/vaults/read
    - Microsoft.KeyVault/vaults/write
* DataActions
    - Microsoft.KeyVault/vaults/secrets/delete
    - Microsoft.KeyVault/vaults/secrets/getSecret/action
    - Microsoft.KeyVault/vaults/secrets/setSecret/action
    - Microsoft.KeyVault/vaults/secrets/readMetadata/action
    - Microsoft.KeyVault/vaults/secrets/setSecret/action


To use the user, service principal, or workload identity in the tests, set all the required configurations using environment variables.
Since this provider uses the [DefaultAzureCredential](https://pkg.go.dev/github.com/Azure/azure-sdk-for-go/sdk/azidentity#DefaultAzureCredential) for authorization, see its documentations for details.
For the Azure provider, see [Authenticating to Azure](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs#authenticating-to-azure).

For example, if you use a service principal using its client secret, set the following environment variables in addiiton to `ARM_SUBSCRIPTION_ID`:

* ARM_CLIENT_ID
* ARM_CLIENT_SECRET
* ARM_TENANT_ID
* AZURE_CLIENT_ID
* AZURE_CLIENT_SECRET
* AZURE_TENANT_ID


#### Run tests from scratch

```sh
export ARM_SUBSCRIPTION_ID=$YOUR_SUBSCRIPTION_ID
make testacc
```

You can run specific tests by setting `TESTARGS`:

```sh
make testacc TESTARGS='-run=TestAccSecretDataSource_basic'
```

#### Run tests with existing Key Vault

To shorten test time, use an existing Key Vault by setting the `KEY_VAULT_ID` environment variable:

```sh
export ARM_SUBSCRIPTION_ID=$YOUR_SUBSCRIPTION_ID
export KEY_VAULT_ID=/subscriptions/$ARM_SUBSCRIPTION_ID/resourceGroups/$resource_group_name/providers/Microsoft.KeyVault/vaults/$key_vault_name
make testacc
```

Using an existing Key Vault skips creating and deleting it, reducing test time by more than 10 minutes.
