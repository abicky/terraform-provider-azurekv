package provider

import (
	"net/http"
	"reflect"
	"testing"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	azfake "github.com/Azure/azure-sdk-for-go/sdk/azcore/fake"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/security/keyvault/azsecrets"
	azsecretsfake "github.com/Azure/azure-sdk-for-go/sdk/security/keyvault/azsecrets/fake"
)

const (
	vaultName = "vault-name"
)

var (
	testKeyVaultID = "/subscriptions/sub/resourceGroups/rg/providers/Microsoft.KeyVault/vaults/" + vaultName
	testVaultURL   = "https://" + vaultName + ".vault.azure.net"
)

func newTestClient(t *testing.T, fakeServer *azsecretsfake.Server) *client {
	t.Helper()

	secretClient, err := azsecrets.NewClient(
		testVaultURL,
		&azfake.TokenCredential{},
		&azsecrets.ClientOptions{
			ClientOptions: azcore.ClientOptions{
				Transport: azsecretsfake.NewServerTransport(fakeServer),
			},
		},
	)
	if err != nil {
		t.Fatalf("azsecrets.NewClient() error = %v", err)
	}

	return &client{
		secretClients: map[string]*azsecrets.Client{
			vaultName: secretClient,
		},
	}
}

func secretProperties(version string, created int64) *azsecrets.SecretProperties {
	createdAt := time.Unix(created, 0)

	return &azsecrets.SecretProperties{
		ID: to.Ptr(azsecrets.ID(testVaultURL + "/secrets/secret-name/" + version)),
		Attributes: &azsecrets.SecretAttributes{
			Created: &createdAt,
		},
	}
}

func TestClientGetSecretProperties(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		pages          [][]*azsecrets.SecretProperties
		version        string
		wantProperties *azsecrets.SecretProperties
		wantErr        bool
	}{
		{
			name: "requested version on a later page",
			pages: [][]*azsecrets.SecretProperties{
				{secretProperties("version-1", 100)},
				{secretProperties("version-2", 200), secretProperties("version-3", 300)},
			},
			version:        "version-2",
			wantProperties: secretProperties("version-2", 200),
		},
		{
			name: "requested version after a non-matching version on the same page",
			pages: [][]*azsecrets.SecretProperties{
				{secretProperties("version-1", 100), secretProperties("version-2", 200)},
			},
			version:        "version-2",
			wantProperties: secretProperties("version-2", 200),
		},
		{
			name: "requested version not found",
			pages: [][]*azsecrets.SecretProperties{
				{secretProperties("version-1", 100), secretProperties("version-2", 200)},
			},
			version: "missing-version",
			wantErr: true,
		},
		{
			name: "latest version",
			pages: [][]*azsecrets.SecretProperties{
				{
					secretProperties("version-2", 200),
					secretProperties("version-3", 300),
					secretProperties("version-1", 100),
				},
			},
			version:        "",
			wantProperties: secretProperties("version-3", 300),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			fakeServer := azsecretsfake.Server{
				NewListSecretPropertiesVersionsPager: func(
					_ string,
					_ *azsecrets.ListSecretPropertiesVersionsOptions,
				) (resp azfake.PagerResponder[azsecrets.ListSecretPropertiesVersionsResponse]) {
					for _, secrets := range tt.pages {
						page := azsecrets.ListSecretPropertiesVersionsResponse{
							SecretPropertiesListResult: azsecrets.SecretPropertiesListResult{
								Value: secrets,
							},
						}
						resp.AddPage(http.StatusOK, page, nil)
					}
					return
				},
			}
			c := newTestClient(t, &fakeServer)

			got, err := c.GetSecretProperties(t.Context(), testKeyVaultID, "secret-name", tt.version, nil)
			if tt.wantErr {
				if err == nil {
					t.Errorf("GetSecretProperties() = %v, want error", got)
				}
				if got != nil {
					t.Errorf("GetSecretProperties() = %v, want nil", got)
				}
				return
			}

			if err != nil {
				t.Fatalf("GetSecretProperties() error = %v", err)
			}
			if !reflect.DeepEqual(got, tt.wantProperties) {
				t.Errorf("GetSecretProperties() = %v, want %v", got, tt.wantProperties)
			}
		})
	}
}
