# Authorizers used by the RP:

Throughout the RP codebase you can find various authorizers that serve different
purposes. The following will help shed some light on what each of them are for,
referring to each by the naming schema which should be consistent throughout the
codebase.

## fpAuthorizer

The first party application, in the customer's tenant, for use against ARM. Used
in steady state: manage cluster resources in the customer's subscription.

## fpGraphAuthorizer

The first party application, in the customer's tenant, for use against AAD. Used
in development mode to emulate ARM.

## localFPAuthorizer

The first party application, in the AME tenant, for use against ARM. Used in
steady state: manage ACR tokens, DNS zone records, private endpoints.

## rpAuthorizer

The managed identity attached to the RP VM, in the AME tenant, for use against
ARM. Used for bootstrapping: finding the CosmosDB account and key, finding the
DNS zone, finding the key vaults, populating the SKU list.

## rpKVAuthorizer

The managed identity attached to the RP VM, in the AME tenant, for use against
the service and cluster key vault. Used for bootstrapping: retrieving keys and
secrets from the service key vault, including the first party certificate + key.
Used in steady state: manage cluster serving certificates.

## spAuthorizer

The cluster's AAD application, in the customer's tenant, for use against ARM.
Used in the cluster context, and by the RP to validate its setup.

## spGraphAuthorizer

The cluster's AAD application, in the customer's tenant, for use against AAD.
Used to discover the object ID of the service principal associated with the
cluster's AAD application in order to populate deny assignments.
