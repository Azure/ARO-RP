package dataplane

import (
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	azcloud "github.com/Azure/azure-sdk-for-go/sdk/azcore/cloud"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/msi-dataplane/pkg/dataplane/swagger"
)

var (
	// Errors returned when processing idenities
	errDecodeClientSecret = errors.New("failed to decode client secret")
	errParseCertificate   = errors.New("failed to parse certificate")
	errParseResourceID    = errors.New("failed to parse resource ID")
	errNilField           = errors.New("expected non nil field in identity")
	errNoUserAssignedMSIs = errors.New("credentials object does not contain user-assigned managed identities")
	errResourceIDNotFound = errors.New("resource ID not found in user-assigned managed identity")
)

// CredentialsObject is a wrapper around the swagger.CredentialsObject to add additional functionality
// swagger.Credentials object can represent either system or user-assigned managed identity
type CredentialsObject struct {
	swagger.CredentialsObject
}

type UserAssignedIdentities struct {
	CredentialsObject
	cloud string
}

// Constructor for UserAssignedIdentities object
func NewUserAssignedIdentities(c CredentialsObject, cloud string) (*UserAssignedIdentities, error) {
	if !c.IsUserAssigned() {
		return nil, errNoUserAssignedMSIs
	}
	return &UserAssignedIdentities{CredentialsObject: c, cloud: cloud}, nil
}

// This method may be used by clients to check if they can use the object as a user-assigned managed identity
// Ex: get credentials object from key vault store and check if it is a user-assigned managed identity to call client for object refresh.
func (c CredentialsObject) IsUserAssigned() bool {
	return len(c.ExplicitIdentities) > 0
}

// Get an AzIdentity credential for the given user-assigned identity resource ID
// Clients can use the credential to get a token for the user-assigned identity
func (u UserAssignedIdentities) GetCredential(requestedResourceID string) (*azidentity.ClientCertificateCredential, error) {
	requestedARMResourceID, err := arm.ParseResourceID(requestedResourceID)
	if err != nil {
		return nil, fmt.Errorf("%w for requested resource ID %s: %w", errParseResourceID, requestedResourceID, err)
	}
	requestedResourceID = requestedARMResourceID.String()

	for _, id := range u.ExplicitIdentities {
		if id != nil && id.ResourceID != nil {
			idARMResourceID, err := arm.ParseResourceID(*id.ResourceID)
			if err != nil {
				return nil, fmt.Errorf("%w for identity resource ID %s: %w", errParseResourceID, *id.ResourceID, err)
			}
			if requestedResourceID == idARMResourceID.String() {
				return getClientCertificateCredential(*id, u.cloud)
			}
		}
	}

	return nil, errResourceIDNotFound
}

func getAzCoreCloud(cloud string) azcloud.Configuration {
	switch cloud {
	case AzureUSGovCloud:
		return azcloud.AzureGovernment
	default:
		return azcloud.AzurePublic
	}
}

func getClientCertificateCredential(identity swagger.NestedCredentialsObject, cloud string) (*azidentity.ClientCertificateCredential, error) {
	// Double check nil pointers so we don't panic
	fieldsToCheck := map[string]*string{
		"clientID":               identity.ClientID,
		"tenantID":               identity.TenantID,
		"clientSecret":           identity.ClientSecret,
		"authenticationEndpoint": identity.AuthenticationEndpoint,
	}
	missing := make([]string, 0)
	for field, val := range fieldsToCheck {
		if val == nil {
			missing = append(missing, field)
		}
	}
	if len(missing) > 0 {
		return nil, fmt.Errorf("%w: %s", errNilField, strings.Join(missing, ","))
	}

	opts := &azidentity.ClientCertificateCredentialOptions{
		ClientOptions: azcore.ClientOptions{
			Cloud: getAzCoreCloud(cloud),
		},

		// x5c header required: https://eng.ms/docs/products/arm/rbac/managed_identities/msionboardingrequestingatoken
		SendCertificateChain: true,

		// Disable instance discovery because MSI credential may have regional AAD endpoint that instance discovery endpoint doesn't support
		// e.g. when MSI credential has westus2.login.microsoft.com, it will cause instance discovery to fail with HTTP 400
		DisableInstanceDiscovery: true,
	}

	// Set the regional AAD endpoint
	// https://eng.ms/docs/products/arm/rbac/managed_identities/msionboardingcredentialapiversion2019-08-31
	opts.Cloud.ActiveDirectoryAuthorityHost = *identity.AuthenticationEndpoint

	// Parse the certificate and private key from the base64 encoded secret
	decodedSecret, err := base64.StdEncoding.DecodeString(*identity.ClientSecret)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", errDecodeClientSecret, err)
	}
	// Note - ParseCertificates does not currently support pkcs12 SHA256 MAC certs, so if
	// managed identity team changes the cert format, double check this code
	crt, key, err := azidentity.ParseCertificates(decodedSecret, nil)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", errParseCertificate, err)
	}
	return azidentity.NewClientCertificateCredential(*identity.TenantID, *identity.ClientID, crt, key, opts)
}

func validateUserAssignedMSIs(identities []*swagger.NestedCredentialsObject, resourceIDs []string) error {
	if len(identities) != len(resourceIDs) {
		return fmt.Errorf("%w, found %d identities instead", errNumberOfMSIs, len(identities))
	}

	resourceIDMap := make(map[string]interface{})
	for _, identity := range identities {
		if identity == nil {
			return errNilMSI
		}
		if identity.ResourceID == nil {
			return fmt.Errorf("%w, resource ID", errNilField)
		}
		armResourceID, err := arm.ParseResourceID(*identity.ResourceID)
		if err != nil {
			return fmt.Errorf("%w for received resource ID %s: %w", errParseResourceID, *identity.ResourceID, err)
		}

		resourceIDMap[armResourceID.String()] = true
	}

	for _, resourceID := range resourceIDs {
		armResourceID, err := arm.ParseResourceID(resourceID)
		if err != nil {
			return fmt.Errorf("%w for requested resource ID %s: %w", errParseResourceID, resourceID, err)
		}
		if _, ok := resourceIDMap[armResourceID.String()]; !ok {
			return fmt.Errorf("%w, resource ID %s", errResourceIDNotFound, resourceID)
		}
	}

	return nil
}
