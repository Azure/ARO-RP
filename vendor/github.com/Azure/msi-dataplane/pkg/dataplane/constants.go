package dataplane

type contextKey string

const (
	// Cloud Environments
	AzurePublicCloud = "AZUREPUBLICCLOUD"
	AzureUSGovCloud  = "AZUREUSGOVERNMENTCLOUD"

	// MSI Headers - exported so frontend RP can reuse
	MsiIdentityURLHeader = "x-ms-identity-url"
	MsiPrincipalIDHeader = "x-ms-identity-principal-id"
	MsiTenantHeader      = "x-ms-home-tenant-id"

	identityURLKey contextKey = MsiIdentityURLHeader

	// Identity URL policy
	apiVersionParameter   = "api-version"
	headerAuthorization   = "authorization"
	headerWWWAuthenticate = "WWW-Authenticate"

	// MSI Endpoints sub domain
	publicMSIEndpoint = "identity.azure.net"
	usGovMSIEndpoint  = "identity.usgovcloudapi.net"

	https = "https"
)
