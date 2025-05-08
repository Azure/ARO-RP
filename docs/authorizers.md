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
in development mode as part of the ARM helper.

## localFPAuthorizer

The first party application, in the AME tenant, for use against ARM. Used in
steady state: manage ACR tokens, DNS zone records, private endpoints.

## localFPKVAuthorizer

The first party application, in the AME tenant, for use against the cluster key
vault. Used in steady state: manage cluster serving certificates.

## msiAuthorizer

The managed identity attached to the RP VM, in the AME tenant, for use against
ARM. Used for bootstrapping: finding the CosmosDB key, populating the SKU list.

## msiRefresherAuthorizer

The managed identity attached to the RP VM, in the AME tenant, for use against
the database token service. Used for retrieving Cosmos DB tokens.

## msiKVAuthorizer

The managed identity attached to the RP VM, in the AME tenant, for use against
the service key vault. Used for bootstrapping: retrieving keys and secrets from
the service key vault, including the first party certificate + key.

## armAuthorizer

The ARM helper application.  Used in development mode only.

## spAuthorizer

The cluster's AAD application, in the customer's tenant, for use against ARM.
Used in the cluster context, and by the RP to validate its setup.

## spGraphAuthorizer

The cluster's AAD application, in the customer's tenant, for use against AAD.
Used to discover the object ID of the service principal associated with the
cluster's AAD application in order to populate deny assignments.
