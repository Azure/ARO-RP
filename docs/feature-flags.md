# Feature flags

Feature flags are used in a couple of different places in the ARO-RP codebase.

## Subscription feature flags

Azure has the capability to set feature flags on subscriptions.  Depending on
the feature flag, it may be settable directly via the end user, or solely via
Geneva.  See the ARM wiki for more details.

ARM advertises subscription flags to RPs as part of the [subscription
lifecycle](https://github.com/Azure/azure-resource-manager-rpc/blob/master/v1.0/subscription-lifecycle-api-reference.md);
the ARO RP responds to subscription PUTs by storing the subscription document
verbatim in Cosmos DB.  Thus subscription feature flags can be checked by
reading the subscription document from the database.

Subscription feature flags used by ARO-RP include:

* Microsoft.RedHatOpenShift/RedHatEngineering: generic feature flag for
  graduating features to production such that they only apply on our Red Hat
  engineering subscription.

Subscription feature flags are also used for API preview, INT and region
rollout. See the RP ARM manifest for more details.

## RP feature flags

The RP_FEATURES environment variable is a comma-delimited list of RP codebase
feature flags defined in pkg/env/env.go.  At the time of writing these include:

* DisableDenyAssignments: don't create a deny assignment on the cluster resource
  group.  Used everywhere except PROD, as this capability is not released
  outside of PROD.

* DisableSignedCertificates: don't integrate with Digicert to sign cluster
  certificates.  Used in development only.

* EnableDevelopmentAuthorizer: use the SubjectNameAndIssuer authorizer instead
  of the ARM authorizer to validate inbound ARM API calls.  Used in development
  only.

* RequireD2sWorkers: require cluster worker VMs to be Standard_D2s (v3, v4, or v5) SKU.
  Used in development only (to save money :-).

* RequireOIDCStorageWebEndpoint: Since Azure Front Door is only present for INT and PROD, there is a need to determine the web endpoint of the OIDC Storage Account after its creation.
Format of web endpoint(It uses Azure DNS Zone endpoint):- **https://[storage-account].z[00-99].web.storage.azure.net** .
Used in development only.

* UseMockMsiRp: The MSI RP is only present in PROD, so this feature flag is used
  in local development, full service development, and INT to tell the RP to use a
  mocked version of the MSI dataplane for the cluster MSI. Only relevant to
  clusters that have a cluster MSI.
