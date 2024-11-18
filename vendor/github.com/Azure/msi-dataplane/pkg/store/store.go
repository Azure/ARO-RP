package store

import (
	"context"
	"errors"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/runtime"
	"github.com/Azure/azure-sdk-for-go/sdk/security/keyvault/azsecrets"
	"github.com/Azure/msi-dataplane/pkg/dataplane"
)

var (
	errNilSecretValue = errors.New("secret value is nil")
)

type DeletedSecretProperties struct {
	Name          string
	RecoveryLevel string
	DeletedDate   time.Time
}

type DeletedSecretResponse struct {
	CredentialsObject dataplane.CredentialsObject
	Properties        DeletedSecretProperties
}

type MsiKeyVaultStore struct {
	kvClient KeyVaultClient
}

type SecretProperties struct {
	Enabled   bool
	Expires   time.Time
	Name      string
	NotBefore time.Time
}

type SecretResponse struct {
	CredentialsObject dataplane.CredentialsObject
	Properties        SecretProperties
}

func NewMsiKeyVaultStore(kvClient KeyVaultClient) *MsiKeyVaultStore {
	return &MsiKeyVaultStore{kvClient: kvClient}
}

// Delete a credentials object from key vault using the specified secret name.
// Delete applies to all versions of the secret.
func (s *MsiKeyVaultStore) DeleteCredentialsObject(ctx context.Context, secretName string) error {
	if _, err := s.kvClient.DeleteSecret(ctx, secretName, nil); err != nil {
		return err
	}

	return nil
}

// Get a credentials object from the key vault using the specified secret name.
// The latest version of the secret will always be returned.
func (s *MsiKeyVaultStore) GetCredentialsObject(ctx context.Context, secretName string) (*SecretResponse, error) {
	// https://github.com/Azure/azure-sdk-for-go/blob/3fab729f1bd43098837ddc34931fec6c342fa3ef/sdk/security/keyvault/azsecrets/client.go#L197
	latestSecretVersion := ""
	secret, err := s.kvClient.GetSecret(ctx, secretName, latestSecretVersion, nil)
	if err != nil {
		return nil, err
	}

	if secret.Value == nil {
		return nil, errNilSecretValue
	}
	var credentialsObject dataplane.CredentialsObject
	if err := credentialsObject.UnmarshalJSON([]byte(*secret.Value)); err != nil {
		return nil, err
	}

	secretProperties := SecretProperties{
		Name:      secretName,
		Enabled:   true, // Default to true
		Expires:   time.Time{},
		NotBefore: time.Time{},
	}

	if secret.Attributes != nil {
		// Override defaults if values are present
		if secret.Attributes.Enabled != nil {
			secretProperties.Enabled = *secret.Attributes.Enabled
		}
		if secret.Attributes.Expires != nil {
			secretProperties.Expires = *secret.Attributes.Expires
		}
		if secret.Attributes.NotBefore != nil {
			secretProperties.NotBefore = *secret.Attributes.NotBefore
		}
	}

	return &SecretResponse{CredentialsObject: credentialsObject, Properties: secretProperties}, nil
}

// Get a deleted credentials object from the key vault using the specified secret name.
func (s *MsiKeyVaultStore) GetDeletedCredentialsObject(ctx context.Context, secretName string) (*DeletedSecretResponse, error) {
	response, err := s.kvClient.GetDeletedSecret(ctx, secretName, nil)
	if err != nil {
		return nil, err
	}

	if response.Value == nil {
		return nil, errNilSecretValue
	}

	var credentialsObject dataplane.CredentialsObject
	if err := credentialsObject.UnmarshalJSON([]byte(*response.Value)); err != nil {
		return nil, err
	}

	deletedSecretProperties := DeletedSecretProperties{
		Name:          secretName,
		RecoveryLevel: "",
		DeletedDate:   time.Time{},
	}

	if response.DeletedDate != nil {
		deletedSecretProperties.DeletedDate = *response.DeletedDate
	}

	if response.Attributes != nil {
		// Override defaults if values are present
		if response.Attributes.RecoveryLevel != nil {
			deletedSecretProperties.RecoveryLevel = *response.Attributes.RecoveryLevel
		}
	}

	return &DeletedSecretResponse{CredentialsObject: credentialsObject, Properties: deletedSecretProperties}, nil
}

// Get a pager for listing credentials objects from the key vault.
func (s *MsiKeyVaultStore) GetCredentialsObjectPager() *runtime.Pager[azsecrets.ListSecretPropertiesResponse] {
	return s.kvClient.NewListSecretPropertiesPager(nil)
}

// Get a pager for listing deleted credentials objects from the key vault.
func (s *MsiKeyVaultStore) GetDeletedCredentialsObjectPager() *runtime.Pager[azsecrets.ListDeletedSecretPropertiesResponse] {
	return s.kvClient.NewListDeletedSecretPropertiesPager(nil)
}

// Purge a deleted credentials object from the key vault using the specified secret name.
// This operation is only applicable in vaults enabled for soft-delete.
func (s *MsiKeyVaultStore) PurgeDeletedCredentialsObject(ctx context.Context, secretName string) error {
	if _, err := s.kvClient.PurgeDeletedSecret(ctx, secretName, nil); err != nil {
		return err
	}

	return nil
}

// Set a credentials object in the key vault using the specified secret name.
// If the secret already exists, key vault will create a new version of the secret.
func (s *MsiKeyVaultStore) SetCredentialsObject(ctx context.Context, properties SecretProperties, credentialsObject dataplane.CredentialsObject) error {
	credentialsObjectBuffer, err := credentialsObject.MarshalJSON()
	if err != nil {
		return err
	}

	credentialsObjectString := string(credentialsObjectBuffer)
	setSecretParameters := azsecrets.SetSecretParameters{
		Value: &credentialsObjectString,
		SecretAttributes: &azsecrets.SecretAttributes{
			Enabled:   &properties.Enabled,
			Expires:   &properties.Expires,
			NotBefore: &properties.NotBefore,
		},
	}
	if _, err := s.kvClient.SetSecret(ctx, properties.Name, setSecretParameters, nil); err != nil {
		return err
	}

	return nil
}
