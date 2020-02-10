# Prepare a shared RP development environment

Follow these steps to build a shared RP development environment and secrets
file.  A single RP development environment can be shared across multiple
developers and/or CI flows.  It may include multiple resource groups in multiple
locations.

## Prerequisites

1. You will need `Contributor` and `User Access Administrator` roles on your
   Azure subscription, as well as the ability to create and configure AAD
   applications.

1. You will need a publicly resolvable DNS Zone resource in your Azure
   subscription.  Set PARENT_DOMAIN_NAME and PARENT_DOMAIN_RESOURCEGROUP to the name and
   resource group of the DNS Zone resource:

   ```
   PARENT_DOMAIN_NAME=osadev.cloud
   PARENT_DOMAIN_RESOURCEGROUP=dns
   ```

1. You will need an AAD object (this could be your AAD user, or an AAD group of
   which you are a member) which will be able to administer certificates in the
   development environment key vault(s).  Set ADMIN_OBJECT_ID to the object ID.

   ```
   ADMIN_OBJECT_ID="$(az ad group show -g Engineering --query objectId -o tsv)"
   ```

1. You will need the ARO RP-specific pull secret (ask one of the
   @azure-red-hat-openshift GitHub team for this):

   ```
   PULL_SECRET=...
   ```

1. Install [Go 1.13](https://golang.org/dl) or later, if you haven't already.

1. Install the [Azure
   CLI](https://docs.microsoft.com/en-us/cli/azure/install-azure-cli), if you
   haven't already.

1. Log in to Azure:

   ```
   az login

   AZURE_TENANT_ID=$(az account show --query tenantId -o tsv)
   AZURE_SUBSCRIPTION_ID=$(az account show --query id -o tsv)
   ```

1. Git clone this repository to your local machine:

   ```
   go get -u github.com/Azure/ARO-RP/...
   cd ${GOPATH:-$HOME/go}/src/github.com/Azure/ARO-RP
   ```

1. Prepare the secrets directory:

   ```
   mkdir -p secrets
   ```


## AAD applications

1. Create an AAD application which will fake up the ARM layer:

   ```
   AZURE_ARM_CLIENT_SECRET="$(uuidgen)"
   AZURE_ARM_CLIENT_ID="$(az ad app create \
     --display-name aro-v4-arm-shared \
     --end-date '2299-12-31T11:59:59+00:00' \
     --identifier-uris "https://$(uuidgen)/" \
     --key-type password \
     --password "$AZURE_ARM_CLIENT_SECRET" \
     --query appId \
     -o tsv)"
   az ad sp create --id "$AZURE_ARM_CLIENT_ID" >/dev/null
   ```

   Later this application will be granted:

   * `User Access Administrator` on your subscription.

1. Create an AAD application which will fake up the first party application.

   This application requires client certificate authentication to be enabled.  A
   suitable key/certificate file can be generated using the following helper
   utility:

   ```
   go run ./hack/genkey -client firstparty-development
   mv firstparty-development.* secrets
   ```

   Now create the application:

   ```
   AZURE_FP_CLIENT_ID="$(az ad app create \
     --display-name aro-v4-fp-shared \
     --identifier-uris "https://$(uuidgen)/" \
     --query appId \
     -o tsv)"
   az ad app credential reset \
     --id "$AZURE_FP_CLIENT_ID" \
     --cert "$(base64 -w0 <secrets/firstparty-development.crt)" >/dev/null
   az ad sp create --id "$AZURE_FP_CLIENT_ID" >/dev/null
   ```

   Later this application will be granted:

   * `ARO v4 FP Subscription` on your subscription.
   * `DNS Zone Contributor` on the DNS zone in RESOURCEGROUP.
   * `Network Contributor` on RESOURCEGROUP.

1. Create an AAD application which will fake up the RP identity.

   ```
   AZURE_CLIENT_SECRET="$(uuidgen)"
   AZURE_CLIENT_ID="$(az ad app create \
     --display-name aro-v4-rp-shared \
     --end-date '2299-12-31T11:59:59+00:00' \
     --identifier-uris "https://$(uuidgen)/" \
     --key-type password \
     --password "$AZURE_CLIENT_SECRET" \
     --query appId \
     -o tsv)"
   az ad sp create --id "$AZURE_CLIENT_ID" >/dev/null
   ```

   Later this application will be granted:

   * `Reader` on RESOURCEGROUP.
   * `Secrets / Get` on the key vault in RESOURCEGROUP.
   * `DocumentDB Account Contributor` on the CosmosDB resource in RESOURCEGROUP.

1. Set up the RP role definitions and subscription role assignments in your
   Azure subscription. This mimics the RBAC that ARM sets up.  With at least
   `User Access Administrator` permissions on your subscription, do:

   ```
   az deployment create \
     -l eastus \
     --template-file deploy/rbac-development.json \
     --parameters \
       "armServicePrincipalId=$(az ad sp list --filter "appId eq '$AZURE_ARM_CLIENT_ID'" --query '[].objectId' -o tsv)" \
       "fpServicePrincipalId=$(az ad sp list --filter "appId eq '$AZURE_FP_CLIENT_ID'" --query '[].objectId' -o tsv)" \
     >/dev/null
   ```


## Certificates

1. Create the VPN CA key/certificate.  A suitable key/certificate file can be
   generated using the following helper utility:

   ```
   go run ./hack/genkey -ca vpn-ca
   mv vpn-ca.* secrets
   ```

1. Create the VPN client key/certificate.  A suitable key/certificate file can be
   generated using the following helper utility:

   ```
   go run ./hack/genkey -client -keyFile secrets/vpn-ca.key -certFile secrets/vpn-ca.crt vpn-client
   mv vpn-client.* secrets
   ```

1. Create the proxy serving key/certificate.  A suitable key/certificate file
   can be generated using the following helper utility:

   ```
   go run ./hack/genkey proxy
   mv proxy.* secrets
   ```

1. Create the proxy client key/certificate.  A suitable key/certificate file can
   be generated using the following helper utility:

   ```
   go run ./hack/genkey -client proxy-client
   mv proxy-client.* secrets
   ```

1. Create the proxy ssh key/certificate.  A suitable key/certificate file can
   be generated using the following helper utility:

   ```
   ssh-keygen -f secrets/proxy_id_rsa -N ''
   ```

1. Create an RP serving key/certificate.  A suitable key/certificate file
   can be generated using the following helper utility:

   ```
   go run ./hack/genkey localhost
   mv localhost.* secrets
   ```


# Environment file

1. Choose the resource group prefix.  The resource group location will be
   appended to the prefix to make the resource group name.

   ```
   RESOURCEGROUP_PREFIX=v4
   ```

1. Choose the proxy domain name label.  This final proxy hostname will be of the
   form `vm0.$PROXY_DOMAIN_NAME_LABEL.$LOCATION.cloudapp.azure.com`.

   ```
   PROXY_DOMAIN_NAME_LABEL=aroproxy
   ```

1. Create the secrets/env file:

   ```
   cat >secrets/env <<EOF
   export AZURE_TENANT_ID='$AZURE_TENANT_ID'
   export AZURE_SUBSCRIPTION_ID='$AZURE_SUBSCRIPTION_ID'
   export AZURE_ARM_CLIENT_ID='$AZURE_ARM_CLIENT_ID'
   export AZURE_ARM_CLIENT_SECRET='$AZURE_ARM_CLIENT_SECRET'
   export AZURE_FP_CLIENT_ID='$AZURE_FP_CLIENT_ID'
   export AZURE_CLIENT_ID='$AZURE_CLIENT_ID'
   export AZURE_CLIENT_SECRET='$AZURE_CLIENT_SECRET'
   export RESOURCEGROUP="$RESOURCEGROUP_PREFIX-\$LOCATION"
   export PROXY_HOSTNAME="vm0.$PROXY_DOMAIN_NAME_LABEL.\$LOCATION.cloudapp.azure.com"
   export DATABASE_NAME="\$USER"
   export RP_MODE='development'
   export PULL_SECRET='$PULL_SECRET'
   ADMIN_OBJECT_ID='$ADMIN_OBJECT_ID'
   COSMOSDB_ACCOUNT="\$RESOURCEGROUP"
   DOMAIN_NAME="\$RESOURCEGROUP"
   KEYVAULT_PREFIX="\$RESOURCEGROUP"
   PARENT_DOMAIN_NAME='$PARENT_DOMAIN_NAME'
   PARENT_DOMAIN_RESOURCEGROUP='$PARENT_DOMAIN_RESOURCEGROUP'
   EOF
   ```


## Deploy shared RP development environment (once per location)

1. Copy, edit (if necessary) and source your environment file.  The required
   environment variable configuration is documented immediately below:

   ```
   cp env.example env
   vi env
   . ./env
   ```

   * LOCATION: Location of the shared RP development environment (default:
     `eastus`).

1. Create the resource group and deploy the RP resources:

   ```
   az group create -g "$RESOURCEGROUP" -l "$LOCATION" >/dev/null

   az group deployment create \
     -g "$RESOURCEGROUP" \
     --template-file deploy/rp-development-nsg.json \
     >/dev/null

   az group deployment create \
     -g "$RESOURCEGROUP" \
     --template-file deploy/rp-development.json \
     --parameters \
       "adminObjectId=$ADMIN_OBJECT_ID" \
       "databaseAccountName=$COSMOSDB_ACCOUNT" \
       "domainName=$DOMAIN_NAME.$PARENT_DOMAIN_NAME" \
       "fpServicePrincipalId=$(az ad sp list --filter "appId eq '$AZURE_FP_CLIENT_ID'" --query '[].objectId' -o tsv)" \
       "keyvaultPrefix=$KEYVAULT_PREFIX" \
       "rpServicePrincipalId=$(az ad sp list --filter "appId eq '$AZURE_CLIENT_ID'" --query '[].objectId' -o tsv)" \
     >/dev/null

   az group deployment create \
     -g "$RESOURCEGROUP" \
     --template-file deploy/env-development.json \
     --parameters \
       "proxyCert=$(base64 -w0 <secrets/proxy.crt)" \
       "proxyClientCert=$(base64 -w0 <secrets/proxy-client.crt)" \
       "proxyDomainNameLabel=$(cut -d. -f2 <<<$PROXY_HOSTNAME)" \
       "proxyImage=arosvc.azurecr.io/proxy:latest" \
       "proxyImageAuth=$(jq -r '.auths["arosvc.azurecr.io"].auth' <<<$PULL_SECRET)" \
       "proxyKey=$(base64 -w0 <secrets/proxy.key)" \
       "sshPublicKey=$(<secrets/proxy_id_rsa.pub)" \
       "vpnCACertificate=$(base64 -w0 <secrets/vpn-ca.crt)" \
     >/dev/null
   ```

1. Load the keys/certificates into the key vault:

   ```
   az keyvault certificate import \
     --vault-name "$KEYVAULT_PREFIX-service" \
     --name rp-firstparty \
     --file secrets/firstparty-development.pem \
     >/dev/null
   az keyvault certificate import \
     --vault-name "$KEYVAULT_PREFIX-service" \
     --name rp-server \
     --file secrets/localhost.pem \
     >/dev/null
   az keyvault secret set \
     --vault-name "$KEYVAULT_PREFIX-service" \
     --name encryption-key \
     --value "$(openssl rand -base64 32)" \
     >/dev/null
   ```

1. Create nameserver records in the parent DNS zone:

   ```
   az network dns record-set ns create \
     --resource-group "$PARENT_DOMAIN_RESOURCEGROUP" \
     --zone "$PARENT_DOMAIN_NAME" \
     --name "$DOMAIN_NAME" \
     >/dev/null

   for ns in $(az network dns zone show \
     --resource-group "$RESOURCEGROUP" \
     --name "$DOMAIN_NAME.$PARENT_DOMAIN_NAME" \
     --query nameServers -o tsv); do
     az network dns record-set ns add-record \
       --resource-group "$PARENT_DOMAIN_RESOURCEGROUP" \
       --zone "$PARENT_DOMAIN_NAME" \
       --record-set-name "$DOMAIN_NAME" \
       --nsdname "$ns" \
       >/dev/null
   done
   ```

1. Store the VPN client configuration:

   ```
   curl -so vpnclientconfiguration.zip "$(az network vnet-gateway vpn-client generate \
     -g "$RESOURCEGROUP" \
     -n dev-vpn \
     -o tsv)"
   export CLIENTCERTIFICATE="$(openssl x509 -inform der -in secrets/vpn-client.crt)"
   export PRIVATEKEY="$(openssl rsa -inform der -in secrets/vpn-client.key)"
   unzip -qc vpnclientconfiguration.zip 'OpenVPN\\vpnconfig.ovpn' \
     | envsubst \
     | grep -v '^log ' >"secrets/vpn-$LOCATION.ovpn"
   rm vpnclientconfiguration.zip
   ```
