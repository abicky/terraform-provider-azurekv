package provider

import (
	"context"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure AzurekvProvider satisfies various provider interfaces.
var _ provider.Provider = (*AzurekvProvider)(nil)

// AzurekvProvider defines the provider implementation.
type AzurekvProvider struct {
	// version is set to the provider version on release, "dev" when the
	// provider is built and run locally, and "test" when running acceptance
	// testing.
	version string
}

// AzurekvProviderModel describes the provider data model.
type AzurekvProviderModel struct {
	SubscriptionID types.String `tfsdk:"subscription_id"`
}

func (p *AzurekvProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "azurekv"
	resp.Version = p.version
}

func (p *AzurekvProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "The Azure Key Vault provider allows you to manage Key Vault secrets without requiring the `Microsoft.KeyVault/vaults/secrets/getSecret/action` permission, by leveraging [write-only arguments](https://developer.hashicorp.com/terraform/language/resources/ephemeral/write-only).",
		Attributes: map[string]schema.Attribute{
			"subscription_id": schema.StringAttribute{
				MarkdownDescription: "The subscription ID which should be used. This can also be sourced from the `ARM_SUBSCRIPTION_ID` environment variable and is only required for import.",
				Optional:            true,
			},
		},
	}
}

func (p *AzurekvProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var model AzurekvProviderModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if model.SubscriptionID.IsNull() {
		if v := os.Getenv("ARM_SUBSCRIPTION_ID"); v != "" {
			model.SubscriptionID = types.StringValue(v)
		}
	}

	c, err := NewClient(model.SubscriptionID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to Create Azure Client", err.Error())
		return
	}

	resp.DataSourceData = c
	resp.ResourceData = c
}

func (p *AzurekvProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewSecretResource,
	}
}

func (p *AzurekvProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewSecretDataSource,
	}
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &AzurekvProvider{
			version: version,
		}
	}
}
