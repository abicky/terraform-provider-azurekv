package provider

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/security/keyvault/azsecrets"
	"github.com/hashicorp/terraform-plugin-framework-timetypes/timetypes"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	idRegex         = regexp.MustCompile(`\Ahttps://([^/]+)\.vault.azure.net/secrets/([^/]+)`)
	keyVaultIDRegex = regexp.MustCompile(`\A/subscriptions/[^/]+/resourceGroups/[^/]+/providers/Microsoft.KeyVault/vaults/([^/]+)\z`)
)

type SecretModel interface {
	GetKeyVaultID() string
	SetID(types.String)
	SetVersionlessID(types.String)
	SetVersion(types.String)
	SetResourceVersionlessID(types.String)
	SetResourceID(types.String)
	SetContentType(types.String)
	SetNotBeforeDate(timetypes.RFC3339)
	SetExpirationDate(timetypes.RFC3339)
	SetTags(types.Map)
}

func setSecretData(s SecretModel, id *azsecrets.ID, attrs *azsecrets.SecretAttributes, contentType *string, tags map[string]*string) diag.Diagnostics {
	s.SetID(types.StringValue(string(*id)))
	s.SetVersionlessID(types.StringValue(strings.TrimSuffix(string(*id), "/"+id.Version())))
	s.SetVersion(types.StringValue(id.Version()))

	resourceVersionlessID := s.GetKeyVaultID() + "/secrets/" + id.Name()
	s.SetResourceVersionlessID(types.StringValue(resourceVersionlessID))
	s.SetResourceID(types.StringValue(resourceVersionlessID + "/versions/" + id.Version()))

	if contentType != nil {
		s.SetContentType(types.StringValue(*contentType))
	}

	if attrs.NotBefore != nil {
		s.SetNotBeforeDate(timetypes.NewRFC3339TimePointerValue(to.Ptr(attrs.NotBefore.UTC())))
	}
	if attrs.Expires != nil {
		s.SetExpirationDate(timetypes.NewRFC3339TimePointerValue(to.Ptr(attrs.Expires.UTC())))
	}

	if tags != nil {
		attrTags, diags := types.MapValueFrom(context.Background(), types.StringType, tags)
		if diags.HasError() {
			return diags
		}

		s.SetTags(attrTags)
	}

	return nil
}

func extractVaultName(keyVaultID string) (string, error) {
	matches := keyVaultIDRegex.FindStringSubmatch(keyVaultID)
	if len(matches) == 0 {
		return "", fmt.Errorf("invalid key vault ID: %q doesn't match %q", keyVaultID, keyVaultIDRegex)
	}

	return matches[1], nil
}

func extractVaultNameAndName(id string) (string, string, error) {
	matches := idRegex.FindStringSubmatch(id)
	if len(matches) == 0 {
		return "", "", fmt.Errorf("invalid ID: %q doesn't match %q", id, idRegex)
	}

	return matches[1], matches[2], nil
}
