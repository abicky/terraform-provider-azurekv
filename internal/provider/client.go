package provider

import (
	"context"
	"fmt"
	"sync"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources"
	"github.com/Azure/azure-sdk-for-go/sdk/security/keyvault/azsecrets"
)

type Client interface {
	GetSubscriptionID() string
	GetSecretProperties(ctx context.Context, keyVaultID, name string, version string, options *azsecrets.ListSecretPropertiesVersionsOptions) (*azsecrets.SecretProperties, error)
	SetSecret(ctx context.Context, keyVaultID, name string, parameters azsecrets.SetSecretParameters, options *azsecrets.SetSecretOptions) (azsecrets.SetSecretResponse, error)
	UpdateSecretProperties(ctx context.Context, keyVaultID, name string, version string, parameters azsecrets.UpdateSecretPropertiesParameters, options *azsecrets.UpdateSecretPropertiesOptions) (azsecrets.UpdateSecretPropertiesResponse, error)
	DeleteSecret(ctx context.Context, keyVaultID, name string, options *azsecrets.DeleteSecretOptions) (azsecrets.DeleteSecretResponse, error)
	GetKeyVaultID(ctx context.Context, name string) (string, error)
}

type client struct {
	cred           azcore.TokenCredential
	subscriptionID string
	secretClients  map[string]*azsecrets.Client
	resourceClient *armresources.Client
	mutex          sync.Mutex
}

var _ Client = (*client)(nil)

func NewClient(subscriptionID string) (Client, error) {
	cred, err := azidentity.NewDefaultAzureCredential(&azidentity.DefaultAzureCredentialOptions{})
	if err != nil {
		return nil, err
	}

	resourceClient, err := armresources.NewClient(subscriptionID, cred, nil)
	if err != nil {
		return nil, err
	}

	return &client{
		cred:           cred,
		subscriptionID: subscriptionID,
		resourceClient: resourceClient,
		secretClients:  make(map[string]*azsecrets.Client),
	}, nil
}

func (c *client) GetSubscriptionID() string {
	return c.subscriptionID
}

func (c *client) GetSecretProperties(ctx context.Context, keyVaultID, name string, version string, options *azsecrets.ListSecretPropertiesVersionsOptions) (*azsecrets.SecretProperties, error) {
	secretClient, err := c.getSecretClient(keyVaultID)
	if err != nil {
		return nil, err
	}

	var latestSecretProperties *azsecrets.SecretProperties
	pager := secretClient.NewListSecretPropertiesVersionsPager(name, options)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, secret := range page.Value {
			if version != "" && version != secret.ID.Version() {
				return secret, nil
			}
			if latestSecretProperties == nil || secret.Attributes.Created.After(*latestSecretProperties.Attributes.Created) {
				latestSecretProperties = secret
			}
		}
	}

	if latestSecretProperties == nil {
		return nil, fmt.Errorf("the secret %q was not found in the key vault %q", name, keyVaultID)
	}

	return latestSecretProperties, nil
}

func (c *client) SetSecret(ctx context.Context, keyVaultID, name string, parameters azsecrets.SetSecretParameters, options *azsecrets.SetSecretOptions) (azsecrets.SetSecretResponse, error) {
	secretClient, err := c.getSecretClient(keyVaultID)
	if err != nil {
		return azsecrets.SetSecretResponse{}, err
	}

	return secretClient.SetSecret(ctx, name, parameters, options)
}

func (c *client) GetKeyVaultID(ctx context.Context, vaultName string) (string, error) {
	pager := c.resourceClient.NewListPager(&armresources.ClientListOptions{
		Filter: to.Ptr(fmt.Sprintf("resourceType eq 'Microsoft.KeyVault/vaults' and name eq '%s'", vaultName)),
	})
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return "", err
		}

		for _, keyVault := range page.Value {
			return *keyVault.ID, nil
		}
	}

	return "", fmt.Errorf("the key vault %q not found; make sure that the key vault name is correct and that you have the \"Microsoft.KeyVault/vaults/read\" permission", vaultName)
}

func (c *client) UpdateSecretProperties(ctx context.Context, keyVaultID, name string, version string, parameters azsecrets.UpdateSecretPropertiesParameters, options *azsecrets.UpdateSecretPropertiesOptions) (azsecrets.UpdateSecretPropertiesResponse, error) {
	secretClient, err := c.getSecretClient(keyVaultID)
	if err != nil {
		return azsecrets.UpdateSecretPropertiesResponse{}, err
	}

	return secretClient.UpdateSecretProperties(ctx, name, version, parameters, options)
}

func (c *client) DeleteSecret(ctx context.Context, keyVaultID, name string, options *azsecrets.DeleteSecretOptions) (azsecrets.DeleteSecretResponse, error) {
	secretClient, err := c.getSecretClient(keyVaultID)
	if err != nil {
		return azsecrets.DeleteSecretResponse{}, err
	}

	return secretClient.DeleteSecret(ctx, name, options)
}

func (c *client) getSecretClient(keyVaultID string) (*azsecrets.Client, error) {
	vaultName, err := extractVaultName(keyVaultID)
	if err != nil {
		return nil, err
	}

	c.mutex.Lock()
	defer c.mutex.Unlock()

	if secretClient, ok := c.secretClients[vaultName]; ok {
		return secretClient, nil
	}

	secretClient, err := azsecrets.NewClient("https://"+vaultName+".vault.azure.net", c.cred, nil)
	if err != nil {
		return nil, err
	}
	c.secretClients[vaultName] = secretClient

	return secretClient, nil
}
