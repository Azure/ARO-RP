package dataplane

import (
	"context"
	"errors"
	"fmt"
	"reflect"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/runtime"
	"github.com/Azure/msi-dataplane/pkg/dataplane/swagger"
	"github.com/go-playground/validator/v10"
)

//go:generate /bin/bash -c "../../hack/mockgen.sh mock_swagger_client/zz_generated_mocks.go client.go"

const (
	// TODO - Make module name configurable
	moduleName = "managedidentitydataplane.APIClient"
	// TODO - Tie the module version to update automatically with new releases
	moduleVersion = "v0.0.1"

	resourceIDsTag = "resource_ids"
)

type ManagedIdentityClient struct {
	swaggerClient msiClient
	cloud         string
}

type UserAssignedMSIRequest struct {
	IdentityURL string   `validate:"required,http_url"`
	ResourceIDs []string `validate:"required,resource_ids"`
	TenantID    string   `validate:"required,uuid"`
}

type msiClient interface {
	Getcreds(ctx context.Context, credRequest swagger.CredRequestDefinition, options *swagger.ManagedIdentityDataPlaneAPIClientGetcredsOptions) (swagger.ManagedIdentityDataPlaneAPIClientGetcredsResponse, error)
}

var _ msiClient = &swagger.ManagedIdentityDataPlaneAPIClient{}

var (
	// Errors returned by the Managed Identity Dataplane API client
	errGetCreds       = errors.New("failed to get credentials")
	errInvalidRequest = errors.New("invalid request")
	errNilMSI         = errors.New("expected non-nil user-assigned managed identity")
	errNumberOfMSIs   = errors.New("returned MSIs does not match number of requested MSIs")
)

// TODO - Add parameter to specify module name in azcore.NewClient()
// NewClient creates a new Managed Identity Dataplane API client
func NewClient(cloud string, authenticator policy.Policy, clientOpts *policy.ClientOptions) (*ManagedIdentityClient, error) {
	var perCallPolicies []policy.Policy
	if authenticator != nil {
		perCallPolicies = append(perCallPolicies, authenticator)
	}
	perCallPolicies = append(perCallPolicies, &injectIdentityURLPolicy{
		msiHost: getMsiHost(cloud),
	})
	plOpts := runtime.PipelineOptions{
		PerCall: perCallPolicies,
	}

	azCoreClient, err := azcore.NewClient(moduleName, moduleVersion, plOpts, clientOpts)
	if err != nil {
		return nil, err
	}
	swaggerClient := swagger.NewSwaggerClient(azCoreClient)

	return &ManagedIdentityClient{swaggerClient: swaggerClient, cloud: cloud}, nil
}

func (c *ManagedIdentityClient) GetUserAssignedIdentities(ctx context.Context, request UserAssignedMSIRequest) (*UserAssignedIdentities, error) {
	validate := validator.New(validator.WithRequiredStructEnabled())
	validate.RegisterValidation(resourceIDsTag, validateResourceIDs)
	if err := validate.Struct(request); err != nil {
		return nil, fmt.Errorf("%w: %w", errInvalidRequest, err)
	}

	identityIDs := make([]*string, len(request.ResourceIDs))
	for idx, r := range request.ResourceIDs {
		identityIDs[idx] = &r
	}

	ctx = context.WithValue(ctx, identityURLKey, request.IdentityURL)
	credRequestDef := swagger.CredRequestDefinition{
		IdentityIDs: identityIDs,
	}

	creds, err := c.swaggerClient.Getcreds(ctx, credRequestDef, nil)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", errGetCreds, err)
	}

	if err := validateUserAssignedMSIs(creds.ExplicitIdentities, request.ResourceIDs); err != nil {
		return nil, err
	}

	// Tenant ID is a header passed to RP frontend, so set it here if it's not set
	for _, identity := range creds.ExplicitIdentities {
		if *identity.TenantID == "" {
			*identity.TenantID = request.TenantID
		}
	}

	credentialsObject := CredentialsObject{CredentialsObject: creds.CredentialsObject}
	return NewUserAssignedIdentities(credentialsObject, c.cloud)
}

func validateResourceIDs(fl validator.FieldLevel) bool {
	field := fl.Field()

	// Confirm we have a slice of strings
	if field.Kind() != reflect.Slice {
		return false
	}

	if field.Type().Elem().Kind() != reflect.String {
		return false
	}

	// Check we have at least one element
	if field.Len() < 1 {
		return false
	}

	// Check that all elements are valid resource IDs
	for i := 0; i < field.Len(); i++ {
		resourceID := field.Index(i).String()
		if !isUserAssignedMSIResource(resourceID) {
			return false
		}
	}

	return true
}

func isUserAssignedMSIResource(resourceID string) bool {
	_, err := arm.ParseResourceID(resourceID)
	if err != nil {
		return false
	}

	resourceType, err := arm.ParseResourceType(resourceID)
	if err != nil {
		return false
	}

	const expectedNamespace = "Microsoft.ManagedIdentity"
	const expectedResourceType = "userAssignedIdentities"

	return resourceType.Namespace == expectedNamespace && resourceType.Type == expectedResourceType
}

func getMsiHost(cloud string) string {
	switch cloud {
	case AzureUSGovCloud:
		return usGovMSIEndpoint
	default:
		return publicMSIEndpoint
	}
}
