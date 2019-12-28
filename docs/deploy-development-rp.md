# Deploy development RP

## Prerequisites - all

1. Install [Go 1.13](https://golang.org/dl) or later, if you haven't already.

1. Install a supported version of [Python](https://www.python.org/downloads), if
   you don't have one installed already.  The `az` client supports Python 2.7
   and Python 3.5+.  A recent Python 3.x version is recommended.

1. Install the
   [`az`](https://docs.microsoft.com/en-us/cli/azure/install-azure-cli) client,
   if you haven't already.

1. Log in to Azure:

   ```
   az login
   ```

1. Git clone this repository to your local machine:

   ```
   go get github.com/Azure/ARO-RP/...
   ```

1. You will need the `Contributor` and `User Access Administrator` roles on your
   subscription.


## Configuration - Red Hat ARO engineering

1. Fetch the development environment secrets:

   ```
   make secrets
   ```

1. Edit and source your environment file.  The required environment variable
   configuration is documented immediately below:

   ```
   cp env.example env
   vi env
   . ./env
   ```

   * LOCATION: Location of the resource group where the development RP will run
     (default: `eastus`).

   * RESOURCEGROUP: Name of the new resource group into which you will deploy
     your RP assets.

   * DATABASE_NAME: Name of the database to use within the CosmosDB database
     account (default: `$USER`).

   * RP_MODE: Set to `development` to enable the RP to read its development
     configuration, and the `az aro` client to connect to the development RP.


## Configuration - non-Red Hat ARO engineering

1. Edit and source your environment file.  The required environment variable
   configuration is documented immediately below:

   ```
   cp env.example env
   vi env
   . ./env
   ```

   * LOCATION: Location of the resource group where the development RP will run
     (default: `eastus`).

   * RESOURCEGROUP: Name of the new resource group into which you will deploy
     your RP assets.

   * DATABASE_NAME: Name of the database to use within the CosmosDB database
     account (default: `$USER`).

   * RP_MODE: Set to `development` to enable the RP to read its development
     configuration, and the `az aro` client to connect to the development RP.

   * AZURE_TENANT_ID: Azure tenant UUID.

   * AZURE_SUBSCRIPTION_ID: Azure subscription UUID.

   * AZURE_ARM_CLIENT_{ID,SECRET}: Credentials of an AAD application which fakes
     up the ARM layer.

     Later it will be granted:

     * `User Access Administrator` on your subscription.

   * AZURE_FP_CLIENT_ID: Client ID of an AAD application which fakes up the
     first party application.

     Later it will be granted:

     * `ARO v4 FP Subscription` on your subscription.

     This application requires client certificate authentication to be enabled.
     A suitable key/certificate file can be generated using the following helper
     utility; then configure it in AAD.

     ```
     go run ./hack/genkey -client firstparty-development
     ```

   * AZURE_CLIENT_{ID,SECRET}: Credentials of an AAD application which fakes up
     the RP identity.

     Later it will be granted:

     * `Reader` on RESOURCEGROUP.
     * `Secrets / Get` on the key vault in RESOURCEGROUP.
     * `DocumentDB Account Contributor` on the CosmosDB resource in RESOURCEGROUP.
     * `DNS Zone Contributor` on the DNS zone in RESOURCEGROUP.

   * PULL_SECRET: A cluster pull secret retrieved from [Red Hat OpenShift
     Cluster
     Manager](https://cloud.redhat.com/openshift/install/azure/installer-provisioned)

   * ADMIN_OBJECT_ID: AAD object ID (e.g. an AAD group, or your AAD user) for
     key vault admin(s)

   * DOMAIN_RESOURCEGROUP, DOMAIN_NAME: Resource group and name of a publicly
     resolvable parent DNS zone resource in your Azure subscription.

1. Set up the RP role definitions and assignments in your Azure subscription.
   This mimics the RBAC that ARM sets up.  With at least `User Access
   Administrator` permissions on your subscription, do:

   ```
   az deployment create \
     -l $LOCATION \
     --template-file deploy/rbac-development.json \
     --parameters \
       "armServicePrincipalId=$ARM_SERVICEPRINCIPAL_ID" \
       "fpServicePrincipalId=$FP_SERVICEPRINCIPAL_ID"
   ```

1. Create an RP serving key/certificate.  A suitable key/certificate file
   can be generated using the following helper utility:

   ```
   go run ./hack/genkey localhost
   ```


## Deploy development RP - all

1. Create the resource group and deploy the RP resources:

   ```
   az group create -g "$RESOURCEGROUP" -l "$LOCATION"

   az group deployment create \
     -g "$RESOURCEGROUP" \
     --template-file deploy/rp-development-nsg.json

   az group deployment create \
     -g "$RESOURCEGROUP" \
     --template-file deploy/rp-development.json \
     --parameters \
       "adminObjectId=$ADMIN_OBJECT_ID" \
       "databaseAccountName=$COSMOSDB_ACCOUNT" \
       "domainName=$DOMAIN" \
       "keyvaultName=$KEYVAULT_NAME" \
       "rpServicePrincipalId=$SERVICEPRINCIPAL_ID"

   az group deployment create \
     -g "$RESOURCEGROUP" \
     --template-file deploy/databases-development.json \
     --parameters \
       "databaseAccountName=$COSMOSDB_ACCOUNT" \
       "databaseName=$DATABASE_NAME"
   ```

1. Load the keys/certificates into the key vault:

   ```
   az keyvault certificate import \
     --vault-name "$KEYVAULT_NAME" \
     --name rp-firstparty \
     --file "$FP_KEYFILE"
   az keyvault certificate import \
     --vault-name "$KEYVAULT_NAME" \
     --name rp-server \
     --file "$KEYFILE"
   ```

1. Create nameserver records in the parent DNS zone:

   ```
   az network dns record-set ns create --resource-group "$DOMAIN_RESOURCEGROUP" --zone "$(cut -d. -f2- <<<"$DOMAIN")" --name "$(cut -d. -f1 <<<"$DOMAIN")"

   for ns in "$(az network dns zone show --resource-group "$RESOURCEGROUP" --name "$DOMAIN" --query nameServers -o tsv)"; do
     az network dns record-set ns add-record \
       --resource-group "$DOMAIN_RESOURCEGROUP" \
       --zone "$(cut -d. -f2- <<<"$DOMAIN")" \
       --record-set-name "$(cut -d. -f1 <<<"$DOMAIN")" \
       --nsdname "$ns"
   done
   ```


## Running the RP and creating a cluster

1. Run the RP

   ```
   go run ./cmd/aro rp
   ```

1. Before creating a cluster, it is necessary to fake up the step of registering
   the development resource provider to the subscription:

   ```
   curl -k -X PUT \
     -H 'Content-Type: application/json'
     -d '{"state": "Registered", "properties": {"tenantId": "'"$AZURE_TENANT_ID"'"}}' \
     "https://localhost:8443/subscriptions/$AZURE_SUBSCRIPTION_ID?api-version=2.0"
   ```

1. To create a cluster, follow the instructions in [using `az
   aro`](docs/using-az-aro.md).  Note that as long as the RP_MODE environment
   variable is set to development, the `az aro` client will connect to your
   local RP.

1. The following additional RP endpoints are available but not exposed via `az
   aro`:

   * Delete a subscription, cascading deletion to all its clusters:

     ```
     curl -k -X PUT \
       -H 'Content-Type: application/json' \
       -d '{"state": "Deleted", "properties": {"tenantId": "'"$AZURE_TENANT_ID"'"}}' \
       "https://localhost:8443/subscriptions/$AZURE_SUBSCRIPTION_ID?api-version=2.0"
     ```

   * List operations:

     ```
     curl -k \
       "https://localhost:8443/providers/Microsoft.RedHatOpenShift/operations?api-version=2019-12-31-preview"
     ```


## Debugging

* SSH to the bootstrap node:

  ```
  hack/ssh-bootstrap.sh "/subscriptions/$AZURE_SUBSCRIPTION_ID/resourceGroups/$RESOURCEGROUP/providers/Microsoft.RedHatOpenShift/openShiftClusters/$CLUSTER"
  ```

* Get an admin kubeconfig:

  ```
  hack/get-admin-kubeconfig.sh "/subscriptions/$AZURE_SUBSCRIPTION_ID/resourceGroups/$RESOURCEGROUP/providers/Microsoft.RedHatOpenShift/openShiftClusters/$CLUSTER"
  export KUBECONFIG=admin.kubeconfig
  oc version
  ```

* "SSH" to a cluster node:

  ```
  hack/ssh.sh [aro-master-0]
  ```
