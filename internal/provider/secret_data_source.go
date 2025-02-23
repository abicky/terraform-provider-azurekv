package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-timetypes/timetypes"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = (*SecretDataSource)(nil)
var _ datasource.DataSourceWithConfigure = (*SecretDataSource)(nil)

func NewSecretDataSource() datasource.DataSource {
	return &SecretDataSource{}
}

// SecretDataSource defines the data source implementation.
type SecretDataSource struct {
	client Client
}

type SecretDataSourceModel struct {
	Name                  types.String      `tfsdk:"name"`
	KeyVaultID            types.String      `tfsdk:"key_vault_id"`
	ID                    types.String      `tfsdk:"id"`
	VersionlessID         types.String      `tfsdk:"versionless_id"`
	ContentType           types.String      `tfsdk:"content_type"`
	NotBeforeDate         timetypes.RFC3339 `tfsdk:"not_before_date"`
	ExpirationDate        timetypes.RFC3339 `tfsdk:"expiration_date"`
	Version               types.String      `tfsdk:"version"`
	ResourceID            types.String      `tfsdk:"resource_id"`
	ResourceVersionlessID types.String      `tfsdk:"resource_versionless_id"`
	Tags                  types.Map         `tfsdk:"tags"`
}

var _ SecretModel = (*SecretDataSourceModel)(nil)

func (s *SecretDataSourceModel) GetKeyVaultID() string {
	return s.KeyVaultID.ValueString()
}

func (s *SecretDataSourceModel) SetID(id types.String) {
	s.ID = id
}

func (s *SecretDataSourceModel) SetVersionlessID(id types.String) {
	s.VersionlessID = id
}

func (s *SecretDataSourceModel) SetVersion(version types.String) {
	s.Version = version
}

func (s *SecretDataSourceModel) SetResourceVersionlessID(id types.String) {
	s.ResourceVersionlessID = id
}

func (s *SecretDataSourceModel) SetResourceID(id types.String) {
	s.ResourceID = id
}

func (s *SecretDataSourceModel) SetContentType(contentType types.String) {
	s.ContentType = contentType
}

func (s *SecretDataSourceModel) SetNotBeforeDate(date timetypes.RFC3339) {
	s.NotBeforeDate = date
}

func (s *SecretDataSourceModel) SetExpirationDate(date timetypes.RFC3339) {
	s.ExpirationDate = date
}

func (s *SecretDataSourceModel) SetTags(tags types.Map) {
	s.Tags = tags
}

func (d *SecretDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_secret"
}

func (d *SecretDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Use this data source to access information about an existing Key Vault Secret excluding the secret value.\n\n" +
			"-> If you need the secret value, use the [`azurerm_key_vault_secret`](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/ephemeral-resources/key_vault_secret) ephemeral resource " +
			"or the [`azurerm_key_vault_secret`](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/data-sources/key_vault_secret) data source instead.",

		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				MarkdownDescription: "Specifies the name of the Key Vault Secret.",
				Required:            true,
			},
			"key_vault_id": schema.StringAttribute{
				MarkdownDescription: "Specifies the ID of the Key Vault instance to fetch secret names from, available on the `azurerm_key_vault` Data Source / Resource.",
				Required:            true,
			},
			"version": schema.StringAttribute{
				MarkdownDescription: "Specifies the version of the Key Vault Secret. Defaults to the current version of the Key Vault Secret.",
				Optional:            true,
			},
			"id": schema.StringAttribute{
				MarkdownDescription: "The Key Vault Secret ID.",
				Computed:            true,
			},
			"versionless_id": schema.StringAttribute{
				MarkdownDescription: "The Versionless ID of the Key Vault Secret. This can be used to always get latest secret value, and enable fetching automatically rotating secrets.",
				Computed:            true,
			},
			"content_type": schema.StringAttribute{
				MarkdownDescription: "The content type for the Key Vault Secret.",
				Computed:            true,
			},
			"not_before_date": schema.StringAttribute{
				MarkdownDescription: "The earliest date at which the Key Vault Secret can be used.",
				Computed:            true,
				CustomType:          timetypes.RFC3339Type{},
			},
			"expiration_date": schema.StringAttribute{
				MarkdownDescription: "The date and time at which the Key Vault Secret expires and is no longer valid.",
				Computed:            true,
				CustomType:          timetypes.RFC3339Type{},
			},
			"resource_id": schema.StringAttribute{
				MarkdownDescription: "The (Versioned) ID for this Key Vault Secret. This property points to a specific version of a Key Vault Secret, as such using this won't auto-rotate values if used in other Azure Services.",
				Computed:            true,
			},
			"resource_versionless_id": schema.StringAttribute{
				MarkdownDescription: "The Versionless ID of the Key Vault Secret. This property allows other Azure Services (that support it) to auto-rotate their value when the Key Vault Secret is updated.",
				Computed:            true,
			},
			"tags": schema.MapAttribute{
				MarkdownDescription: "Any tags assigned to this resource.",
				Computed:            true,
				ElementType:         types.StringType,
			},
		},
	}
}

func (d *SecretDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	c, ok := req.ProviderData.(Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Configure Type",
			fmt.Sprintf("Expected Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	d.client = c
}

func (d *SecretDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var model SecretDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = tflog.SetField(ctx, LogKeyResourceID, model.ID.ValueString())

	secretProperties, err := d.client.GetSecretProperties(
		ctx,
		model.KeyVaultID.ValueString(),
		model.Name.ValueString(),
		model.Version.ValueString(),
		nil,
	)
	if err != nil {
		resp.Diagnostics.AddError("Failed to Get Secret Properties", err.Error())
		return
	}

	resp.Diagnostics.Append(setSecretData(&model, secretProperties.ID, secretProperties.Attributes, secretProperties.ContentType, secretProperties.Tags)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}
