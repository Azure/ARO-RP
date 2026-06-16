using Azure.ResourceManager.CommonTypes;

namespace Azure.ResourceManager;

// customer-managed-keys
alias InfrastructureEncryption = CommonTypes.InfrastructureEncryption;
alias KeyEncryptionIdentity = CommonTypes.KeyEncryptionKeyIdentity;
alias KeyEncryptionKeyIdentity = CommonTypes.KeyEncryptionKeyIdentity;
alias CustomerManagedKeyEncryption = CommonTypes.CustomerManagedKeyEncryption;
alias EncryptionConfiguration = CommonTypes.Encryption;

// private-links
alias PrivateEndpoint = CommonTypes.PrivateEndpoint;
alias PrivateEndpointConnection = CommonTypes.PrivateEndpointConnection;
alias PrivateEndpointConnectionProperties = CommonTypes.PrivateEndpointConnectionProperties;
alias PrivateLinkServiceConnectionState = CommonTypes.PrivateLinkServiceConnectionState;
alias PrivateEndpointConnectionProvisioningState = CommonTypes.PrivateEndpointConnectionProvisioningState;
alias PrivateEndpointServiceConnectionStatus = CommonTypes.PrivateEndpointServiceConnectionStatus;
alias PrivateEndpointConnectionParameter = CommonTypes.PrivateEndpointConnectionParameter;
alias PrivateLinkResource = CommonTypes.PrivateLinkResource;
alias PrivateLinkResourceProperties = CommonTypes.PrivateLinkResourceProperties;
alias PrivateLinkResourceParameter<
  Segment extends valueof string = "privateLinkResources",
  ResourceNamePattern extends valueof string = string("^[a-z][a-zA-Z0-9]*$"),
  MinLength extends valueof int32 = 3,
  MaxLength extends valueof int32 = 63
> = CommonTypes.PrivateLinkResourceNameParameter<
  Segment,
  ResourceNamePattern,
  MinLength,
  MaxLength
>;
alias PrivateEndpointConnectionResourceListResult = CommonTypes.PrivateEndpointConnectionListResult;
#suppress "deprecated" "Need for back compatibility in common-types"
alias PrivateEndpointConnectionResourceListResultV5 = CommonTypes.PrivateEndpointConnectionListResultV5;
alias PrivateLinkResourceListResult = CommonTypes.PrivateLinkResourceListResult;
#suppress "deprecated" "Need for back compatibility in common-types"
alias PrivateLinkResourceListResultV5 = CommonTypes.PrivateLinkResourceListResultV5;

// types
alias ErrorResponse = CommonTypes.ErrorResponse;
