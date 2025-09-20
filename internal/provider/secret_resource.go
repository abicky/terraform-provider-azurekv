package provider

import (
	"context"
	"fmt"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/security/keyvault/azsecrets"
	"github.com/hashicorp/terraform-plugin-framework-timetypes/timetypes"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/identityschema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

const (
	LogKeyResourceID = "resource_id"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = (*SecretResource)(nil)
var _ resource.ResourceWithConfigure = (*SecretResource)(nil)
var _ resource.ResourceWithModifyPlan = (*SecretResource)(nil)
var _ resource.ResourceWithImportState = (*SecretResource)(nil)
var _ resource.ResourceWithIdentity = (*SecretResource)(nil)

func NewSecretResource() resource.Resource {
	return &SecretResource{}
}

// SecretResource defines the resource implementation.
type SecretResource struct {
	client Client
}

type SecretResourceModel struct {
	SecretDataSourceModel
	ValueWO        types.String `tfsdk:"value_wo"`
	ValueWOVersion types.Int32  `tfsdk:"value_wo_version"`
}

var _ SecretModel = (*SecretResourceModel)(nil)

type SecretResourceIdentityModel struct {
	Name       types.String `tfsdk:"name"`
	KeyVaultID types.String `tfsdk:"key_vault_id"`
}

func (r *SecretResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_secret"
}

func (r *SecretResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a Key Vault secret. This resource provides the same interface as [`azurerm_key_vault_secret`](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/key_vault_secret), but does not require the `Microsoft.KeyVault/vaults/secrets/getSecret/action` permission.",

		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				MarkdownDescription: "Specifies the name of the Key Vault Secret. Changing this forces a new resource to be created.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"key_vault_id": schema.StringAttribute{
				MarkdownDescription: "The ID of the Key Vault where the Secret should be created. Changing this forces a new resource to be created.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.RegexMatches(keyVaultIDRegex, ""),
				},
			},
			"id": schema.StringAttribute{
				MarkdownDescription: "The Key Vault Secret ID.",
				Computed:            true,
			},
			"versionless_id": schema.StringAttribute{
				MarkdownDescription: "The Base ID of the Key Vault Secret.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"value_wo": schema.StringAttribute{
				MarkdownDescription: "Specifies the value of the Key Vault Secret. Changing this will create a new version of the Key Vault Secret.",
				Required:            true,
				Sensitive:           true,
				WriteOnly:           true,
			},
			"value_wo_version": schema.Int32Attribute{
				MarkdownDescription: "An integer value used to trigger an update for `value_wo`. This property should be incremented when updating `value_wo`.",
				Required:            true,
			},
			"content_type": schema.StringAttribute{
				MarkdownDescription: "Specifies the content type for the Key Vault Secret.",
				Optional:            true,
				Computed:            true,
				// The default value of azurerm provider
				Default: stringdefault.StaticString(""),
			},
			"not_before_date": schema.StringAttribute{
				MarkdownDescription: "Key not usable before the provided UTC datetime (Y-m-d'T'H:M:S'Z').",
				Optional:            true,
				CustomType:          timetypes.RFC3339Type{},
			},
			"expiration_date": schema.StringAttribute{
				MarkdownDescription: "Expiration UTC datetime (Y-m-d'T'H:M:S'Z').",
				Optional:            true,
				CustomType:          timetypes.RFC3339Type{},
			},
			"version": schema.StringAttribute{
				MarkdownDescription: "The current version of the Key Vault Secret.",
				Computed:            true,
			},
			"resource_id": schema.StringAttribute{
				MarkdownDescription: "The (Versioned) ID for this Key Vault Secret. This property points to a specific version of a Key Vault Secret, as such using this won't auto-rotate values if used in other Azure Services.",
				Computed:            true,
			},
			"resource_versionless_id": schema.StringAttribute{
				MarkdownDescription: "The Versionless ID of the Key Vault Secret. This property allows other Azure Services (that support it) to auto-rotate their value when the Key Vault Secret is updated.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"tags": schema.MapAttribute{
				MarkdownDescription: "A mapping of tags to assign to the resource.",
				ElementType:         types.StringType,
				Optional:            true,
				Computed:            true,
				Default:             mapdefault.StaticValue(types.MapValueMust(types.StringType, map[string]attr.Value{})),
			},
		},
	}
}

func (r *SecretResource) IdentitySchema(_ context.Context, _ resource.IdentitySchemaRequest, resp *resource.IdentitySchemaResponse) {
	resp.IdentitySchema = identityschema.Schema{
		Attributes: map[string]identityschema.Attribute{
			"name": identityschema.StringAttribute{
				Description:       "The name of the Key Vault Secret.",
				RequiredForImport: true,
			},
			"key_vault_id": identityschema.StringAttribute{
				Description:       "The ID of the Key Vault where the Secret is managed.",
				RequiredForImport: true,
			},
		},
	}
}

func (r *SecretResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	c, ok := req.ProviderData.(Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	r.client = c
}

func (r *SecretResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var model SecretResourceModel
	var secretValue string

	resp.Diagnostics.Append(req.Plan.Get(ctx, &model)...)
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("value_wo"), &secretValue)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = tflog.SetField(ctx, LogKeyResourceID, model.ID.ValueString())

	tags, diags := toMap(model.Tags)
	resp.Diagnostics.Append(diags...)

	attrs, diags := buildSecretAttributes(model)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	setResp, err := r.client.SetSecret(ctx, model.KeyVaultID.ValueString(), model.Name.ValueString(), azsecrets.SetSecretParameters{
		Value:            to.Ptr(secretValue),
		ContentType:      model.ContentType.ValueStringPointer(),
		SecretAttributes: attrs,
		Tags:             tags,
	}, nil)
	if err != nil {
		// TODO: Handle ObjectIsDeletedButRecoverable
		resp.Diagnostics.AddError(
			"Failed to Set Secret",
			"An unexpected error occurred while setting a secret: "+err.Error(),
		)
		return
	}

	resp.Diagnostics.Append(setSecretData(&model, setResp.ID, setResp.Attributes, setResp.ContentType, setResp.Tags)...)

	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)

	identity := SecretResourceIdentityModel{
		Name:       model.Name,
		KeyVaultID: model.KeyVaultID,
	}
	resp.Diagnostics.Append(resp.Identity.Set(ctx, identity)...)
}

func (r *SecretResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var model SecretResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = tflog.SetField(ctx, LogKeyResourceID, model.ID.ValueString())

	secretProperties, err := r.client.GetSecretProperties(ctx, model.KeyVaultID.ValueString(), model.Name.ValueString(), "", nil)
	if err != nil {
		resp.Diagnostics.AddError("Failed to Get Secret Properties", err.Error())
		return
	}

	resp.Diagnostics.Append(setSecretData(&model, secretProperties.ID, secretProperties.Attributes, secretProperties.ContentType, secretProperties.Tags)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)

	identity := SecretResourceIdentityModel{
		Name:       model.Name,
		KeyVaultID: model.KeyVaultID,
	}
	resp.Diagnostics.Append(resp.Identity.Set(ctx, identity)...)
}

func (r *SecretResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var model SecretResourceModel
	var secretValue string
	var valueWOVersion int32

	resp.Diagnostics.Append(req.Plan.Get(ctx, &model)...)
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("value_wo"), &secretValue)...)
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("value_wo_version"), &valueWOVersion)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = tflog.SetField(ctx, LogKeyResourceID, model.ID.ValueString())

	tags, diags := toMap(model.Tags)
	resp.Diagnostics.Append(diags...)

	attrs, diags := buildSecretAttributes(model)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	keyVaultID := model.KeyVaultID.ValueString()
	name := model.Name.ValueString()

	if valueWOVersion != model.ValueWOVersion.ValueInt32() {
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}

		setResp, err := r.client.SetSecret(ctx, keyVaultID, name, azsecrets.SetSecretParameters{
			Value:            to.Ptr(secretValue),
			ContentType:      model.ContentType.ValueStringPointer(),
			SecretAttributes: attrs,
			Tags:             tags,
		}, nil)
		if err != nil {
			resp.Diagnostics.AddError(
				"Failed to Set Secret",
				"An unexpected error occurred while setting a secret: "+err.Error(),
			)
			return
		}

		resp.Diagnostics.Append(setSecretData(&model, setResp.ID, setResp.Attributes, setResp.ContentType, setResp.Tags)...)
	} else {
		updateResp, err := r.client.UpdateSecretProperties(ctx, keyVaultID, name, model.Version.ValueString(), azsecrets.UpdateSecretPropertiesParameters{
			ContentType:      model.ContentType.ValueStringPointer(),
			SecretAttributes: attrs,
			Tags:             tags,
		}, nil)
		if err != nil {
			resp.Diagnostics.AddError(
				"Failed to Update Secret Properties",
				"An unexpected error occurred while updating secret properties: "+err.Error(),
			)
			return
		}

		resp.Diagnostics.Append(setSecretData(&model, updateResp.ID, updateResp.Attributes, updateResp.ContentType, updateResp.Tags)...)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}

func (r *SecretResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state SecretResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = tflog.SetField(ctx, LogKeyResourceID, state.ID.ValueString())

	if _, err := r.client.DeleteSecret(ctx, state.KeyVaultID.ValueString(), state.Name.ValueString(), nil); err != nil {
		resp.Diagnostics.AddError(
			"Failed to Delete Secret",
			"An unexpected error occurred while deleting a secret: "+err.Error(),
		)
		return
	}
}

func (r *SecretResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	var keyVaultID string

	if !req.Identity.Raw.IsNull() {
		var identity SecretResourceIdentityModel
		resp.Diagnostics.Append(req.Identity.Get(ctx, &identity)...)
		if resp.Diagnostics.HasError() {
			return
		}

		name := identity.Name.ValueString()
		keyVaultID = identity.KeyVaultID.ValueString()
		secretProperties, err := r.client.GetSecretProperties(ctx, keyVaultID, name, "", nil)
		if err != nil {
			resp.Diagnostics.AddError("Failed to Get Secret Properties", err.Error())
			return
		}

		ctx = tflog.SetField(ctx, LogKeyResourceID, secretProperties.ID)
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), secretProperties.ID)...)
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("name"), name)...)
		if resp.Diagnostics.HasError() {
			return
		}
	} else {
		ctx = tflog.SetField(ctx, LogKeyResourceID, req.ID)

		resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)

		if r.client.GetSubscriptionID() == "" {
			resp.Diagnostics.AddError(
				"Missing Configuration",
				"Subscription ID is required to import a secret",
			)
			return
		}

		// Set key_vault_id manually because the configuration value is not accessible
		// cf. https://discuss.hashicorp.com/t/access-resource-configuration-in-plugin-framework-read/57440
		vaultName, name, err := extractVaultNameAndName(req.ID)
		if err != nil {
			resp.Diagnostics.AddError(
				"Invalid ID",
				err.Error(),
			)
			return
		}

		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("name"), name)...)
		if resp.Diagnostics.HasError() {
			return
		}

		keyVaultID, err = r.client.GetKeyVaultID(ctx, vaultName)
		if err != nil {
			resp.Diagnostics.AddError(
				"Failed to Get KeyVaults",
				err.Error(),
			)
		}
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("key_vault_id"), keyVaultID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("value_wo_version"), 1)...)
}

func (r *SecretResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.Config.Raw.IsNull() || // This resource will be deleted
		req.State.Raw.IsNull() { // This resource will be created
		return
	}

	var config, state SecretResourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = tflog.SetField(ctx, LogKeyResourceID, state.ID.ValueString())

	if config.ValueWO.IsNull() || config.ValueWOVersion.IsNull() {
		tflog.Debug(ctx, "The secret value will not be be updated because the change of the value_wo or value_wo_version seem to be ignored by the lifecycle")
		return
	}

	if config.ValueWO.IsUnknown() {
		tflog.Debug(ctx, "The secret value will be updated because the value is unknown")
		markValueWillChange(ctx, resp)
		return
	}

	if config.ValueWOVersion != state.ValueWOVersion {
		tflog.Debug(ctx, "The secret value will be updated because the value_wo_version changes")
		markValueWillChange(ctx, resp)
		return
	}

	resp.Plan.SetAttribute(ctx, path.Root("id"), state.ID.ValueString())
	resp.Plan.SetAttribute(ctx, path.Root("resource_id"), state.ResourceID.ValueString())
	resp.Plan.SetAttribute(ctx, path.Root("version"), state.Version.ValueString())
}

func toMap(m types.Map) (map[string]*string, diag.Diagnostics) {
	ret := make(map[string]*string)
	diags := m.ElementsAs(context.Background(), &ret, false)
	return ret, diags
}

func buildSecretAttributes(model SecretResourceModel) (*azsecrets.SecretAttributes, diag.Diagnostics) {
	var diags diag.Diagnostics
	var expires, notBefore time.Time
	var attrs azsecrets.SecretAttributes

	if !model.ExpirationDate.IsNull() {
		var expiresDiags diag.Diagnostics
		expires, expiresDiags = model.ExpirationDate.ValueRFC3339Time()
		diags.Append(expiresDiags...)
		attrs.Expires = to.Ptr(expires)
	}

	if !model.NotBeforeDate.IsNull() {
		var notBeforeDiags diag.Diagnostics
		notBefore, notBeforeDiags = model.NotBeforeDate.ValueRFC3339Time()
		diags.Append(notBeforeDiags...)
		attrs.NotBefore = to.Ptr(notBefore)
	}

	return &attrs, diags
}

func markValueWillChange(ctx context.Context, resp *resource.ModifyPlanResponse) {
	// When the value changes, these attributes also change
	resp.Plan.SetAttribute(ctx, path.Root("id"), types.StringUnknown())
	resp.Plan.SetAttribute(ctx, path.Root("resource_id"), types.StringUnknown())
	resp.Plan.SetAttribute(ctx, path.Root("version"), types.StringUnknown())
}
