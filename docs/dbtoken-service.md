# DB token service

## Introduction

Cosmos DB access control is described
[https://docs.microsoft.com/en-us/azure/cosmos-db/secure-access-to-data](here).
In brief, there are three options:

1. use r/w or r/o primary keys, which grant access to the whole database account
2. implement a service which transforms (1) into scoped resource tokens
3. a third AAD RBAC-based model is in preview.

Currently, the RP, monitoring and portal service share the same security
boundary (the RP VM) and use option 1.  The dbtoken service, which also runs on
the RP VM, is our implementation of option 2.  As and when option 3 goes GA, it
may be possible to retire the dbtoken service.

The purpose of the dbtoken service at its implementation time is to enable the
gateway component (which handles end-user traffic) to access the service Cosmos
DB without recourse to using root credentials.  This provides a level of defence
in depth in the face of an attack on the gateway component.


## Workflow

* An AAD application is manually created at rollout, registering the
  https://dbtoken.aro.azure.com resource.

* The dbtoken service receives POST requests from any client wishing to receive
  a scoped resource token at its /token?permission=<permission> endpoint.

* The dbtoken service validates that the POST request includes a valid
  AAD-signed bearer JWT for the https://dbtoken.aro.azure.com resource.  The
  subject UUID is retrieved from the JWT.

* In the case of the gateway service, the JWT subject UUID is the UUID of the
  service principal corresponding to the gateway VMSS MSI.

* Using its primary key Cosmos DB credential, the dbtoken requests a scoped
  resource token for the given user UUID and <permission> from Cosmos DB and
  proxies it to the caller.

* Clients may use the dbtoken.Refresher interface to handle regularly refreshing
  the resource token and injecting it into the database client used by the rest
  of the client codebase.


## Setup

* Create the application and set `requestedAccessTokenVersion`

   ```bash
   AZURE_DBTOKEN_CLIENT_ID="$(az ad app create --display-name dbtoken \
      --oauth2-allow-implicit-flow false \
      --query appId \
      -o tsv)"

   OBJ_ID="$(az ad app show --id $AZURE_DBTOKEN_CLIENT_ID --query id)"

   # NOTE: the graph API requires this to be done from a managed machine
   az rest --method PATCH \
      --uri https://graph.microsoft.com/v1.0/applications/$OBJ_ID/ \
      --body '{"api":{"requestedAccessTokenVersion": 2}}'
   ```

   * Add the `AZURE_DBTOKEN_CLIENT_ID` to the RP config for the respective environment.

* The dbtoken service is responsible for creating database users and permissions

  * see the ConfigurePermissions function.


