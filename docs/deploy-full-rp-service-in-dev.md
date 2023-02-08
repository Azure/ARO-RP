# Deploy an Entire RP Development Service

## Prerequisites

1. Your development environment is prepared according to the steps outlined in [Prepare Your Dev Environment](./prepare-your-dev-environment.md)


## Deploying an int-like Development RP

1. Fetch the most up-to-date secrets with `make secrets`

1. Copy and source your environment file.
    ```bash
    cp env.example env
    vi env
    . ./env
    ```

1. Create a full environment file, which overrides some default `./env` options when sourced
    * if using a public key separate from `~/.ssh/id_rsa.pub`, source it with `export SSH_PUBLIC_KEY=~/.ssh/id_separate.pub`
    ```bash
    cp env-int.example env-int
    vi env-int
    . ./env-int
    ```

1. Generate the development RP configuration
    ```bash
    make dev-config.yaml
    ```

1. Run `make deploy`. This will fail on the first attempt to run due to AKS not being installed, so after the first failure, please skip to the next step to deploy the VPN Gateway and then deploy AKS.
    > __NOTE:__ If the deployment fails with `InvalidResourceReference` due to the RP Network Security Groups not found, delete the "gateway-production-predeploy" deployment in the gateway resource group, and re-run `make deploy`.

    > __NOTE:__ If the deployment fails with `A vault with the same name already exists in deleted state`, then you will need to recover the deleted keyvaults from a previous deploy using: `az keyvault recover --name <KEYVAULT_NAME>` for each keyvault, and re-run.

1. Deploy a VPN Gateway
    This is required in order to be able to connect to AKS from your local machine:
    ```bash
    source ./hack/devtools/deploy-shared-env.sh
    deploy_vpn_for_dedicated_rp
    ```

1. Deploy AKS by running these commands from the ARO-RP root directory:
    ```bash
    source ./hack/devtools/deploy-shared-env.sh
    deploy_aks_dev
    ```
    > __NOTE:__ If the AKS deployment fails with missing RP VNETs, delete the "gateway-production-predeploy" deployment in the gateway resource group, and re-run `make deploy` and then re-run `deploy_aks_dev`.

1. Install Hive into AKS
    1. Download the VPN config. Please note that this action will _**OVER WRITE**_ the `secrets/vpn-$LOCATION.ovpn` on your local machine. **DO NOT** run `make secrets-update` after doing this, as you will overwrite existing config, until such time as you have run `make secrets` to get the config restored.
        ```bash
        vpn_configuration
        ```

    1. Connect to the Dev VPN in a new terminal:
        ```bash
        sudo openvpn secrets/vpn-$LOCATION.ovpn
        ```

    1. Now that your machine is able access the AKS cluster, you can deploy Hive:
        ```bash
        make aks.kubeconfig
        ./hack/hive-generate-config.sh
        KUBECONFIG=$(pwd)/aks.kubeconfig ./hack/hive-dev-install.sh
        ```

1. Mirror the OpenShift images to your new ACR
    <!-- TODO (bv) allow mirroring through a pipeline would be faster and a nice to have -->
    > __NOTE:__ Running the mirroring through a VM in Azure rather than a local workstation is recommended for better performance.

    1. Setup mirroring environment variables
        ```bash
        export DST_ACR_NAME=${USER}aro
        export SRC_AUTH_QUAY=$(echo $USER_PULL_SECRET | jq -r '.auths."quay.io".auth')
        export SRC_AUTH_REDHAT=$(echo $USER_PULL_SECRET | jq -r '.auths."registry.redhat.io".auth')
        export DST_AUTH=$(echo -n '00000000-0000-0000-0000-000000000000:'$(az acr login -n ${DST_ACR_NAME} --expose-token | jq -r .accessToken) | base64 -w0)
        ```

    1. Login to the Azure Container Registry
        ```bash
        docker login -u 00000000-0000-0000-0000-000000000000 -p "$(echo $DST_AUTH | base64 -d | cut -d':' -f2)" "${DST_ACR_NAME}.azurecr.io"
        ```

    1. Run the mirroring
        > The `latest` argument will take the InstallStream from `pkg/util/version/const.go` and mirror that version
        ```bash
        go run -tags aro ./cmd/aro mirror latest
        ```
        If you are going to test or work with multi-version installs, then you should mirror any additional versions as well, for example for 4.11.21 it would be
        ```bash
        go run -tags aro ./cmd/aro mirror 4.11.21
        ```

    1. Push the ARO and Fluentbit images to your ACR

        > If running this step from a VM separate from your workstation, ensure the commit tag used to build the image matches the commit tag where `make deploy` is run.

        > Due to security compliance requirements, `make publish-image-*` targets pull from `arointsvc.azurecr.io`. You can either authenticate to this registry using `az acr login --name arointsvc` to pull the image, or modify the $RP_IMAGE_ACR environment variable locally to point to `registry.access.redhat.com` instead.

        ```bash
        make publish-image-aro-multistage
        make publish-image-fluentbit
        ```

1. Update the DNS Child Domains
    ```bash
    export PARENT_DOMAIN_NAME=osadev.cloud
    export PARENT_DOMAIN_RESOURCEGROUP=dns
    export GLOBAL_RESOURCEGROUP=$USER-global

    for DOMAIN_NAME in $USER-clusters.$PARENT_DOMAIN_NAME $USER-rp.$PARENT_DOMAIN_NAME; do
        CHILD_DOMAIN_PREFIX="$(cut -d. -f1 <<<$DOMAIN_NAME)"
        echo "########## Creating NS record to DNS Zone $CHILD_DOMAIN_PREFIX ##########"
        az network dns record-set ns create \
            --resource-group "$PARENT_DOMAIN_RESOURCEGROUP" \
            --zone "$PARENT_DOMAIN_NAME" \
            --name "$CHILD_DOMAIN_PREFIX" >/dev/null
        for ns in $(az network dns zone show \
            --resource-group "$GLOBAL_RESOURCEGROUP" \
            --name "$DOMAIN_NAME" \
            --query nameServers -o tsv); do
            az network dns record-set ns add-record \
            --resource-group "$PARENT_DOMAIN_RESOURCEGROUP" \
            --zone "$PARENT_DOMAIN_NAME" \
            --record-set-name "$CHILD_DOMAIN_PREFIX" \
            --nsdname "$ns" >/dev/null
        done
    done
    ```

<!-- TODO: this is almost duplicated elsewhere.  Would be nice to move to common area -->
1. Update the certificates in keyvault
    > __NOTE:__ If you reuse an old name, you might run into soft-delete of the keyvaults. Run `az keyvault recover --name` to fix this.

    > __NOTE:__ Check to ensure that the $KEYVAULT_PREFIX environment variable set on workstation matches the prefix deployed into the resource group.

    ```bash
    az keyvault certificate import \
        --vault-name "$KEYVAULT_PREFIX-svc" \
        --name rp-mdm \
        --file secrets/rp-metrics-int.pem >/dev/null
    az keyvault certificate import \
        --vault-name "$KEYVAULT_PREFIX-gwy" \
        --name gwy-mdm \
        --file secrets/rp-metrics-int.pem >/dev/null
    az keyvault certificate import \
        --vault-name "$KEYVAULT_PREFIX-svc" \
        --name rp-mdsd \
        --file secrets/rp-logging-int.pem >/dev/null
    az keyvault certificate import \
        --vault-name "$KEYVAULT_PREFIX-gwy" \
        --name gwy-mdsd \
        --file secrets/rp-logging-int.pem >/dev/null
    az keyvault certificate import \
        --vault-name "$KEYVAULT_PREFIX-svc" \
        --name cluster-mdsd \
        --file secrets/cluster-logging-int.pem >/dev/null
    az keyvault certificate import \
        --vault-name "$KEYVAULT_PREFIX-svc" \
        --name dev-arm \
        --file secrets/arm.pem >/dev/null
    az keyvault certificate import \
        --vault-name "$KEYVAULT_PREFIX-svc" \
        --name rp-firstparty \
        --file secrets/firstparty.pem >/dev/null
    az keyvault certificate import \
        --vault-name "$KEYVAULT_PREFIX-svc" \
        --name rp-server \
        --file secrets/localhost.pem >/dev/null
    az keyvault certificate import \
        --vault-name "$KEYVAULT_PREFIX-por" \
        --name portal-server \
        --file secrets/localhost.pem >/dev/null
    az keyvault certificate import \
        --vault-name "$KEYVAULT_PREFIX-por" \
        --name portal-client \
        --file secrets/portal-client.pem >/dev/null
    az keyvault certificate import \
        --vault-name "$KEYVAULT_PREFIX-dbt" \
        --name dbtoken-server \
        --file secrets/localhost.pem >/dev/null
    ```

1. Delete the existing VMSS
    > __NOTE:__ This needs to be deleted as deploying won't recreate the VMSS if the commit hash is the same.

    ```bash
    az vmss delete -g ${RESOURCEGROUP} --name rp-vmss-$(git rev-parse --short=7 HEAD)$([[ $(git status --porcelain) = "" ]] || echo -dirty) && az vmss delete -g $USER-gwy-$LOCATION --name gateway-vmss-$(git rev-parse --short=7 HEAD)$([[ $(git status --porcelain) = "" ]] || echo -dirty)
    ```

1. Run `make deploy`

## SSH to RP VMSS Instance

1. Update the RP NSG to allow SSH
    ```bash
    az network nsg rule create \
        --name ssh-to-rp \
        --resource-group $RESOURCEGROUP \
        --nsg-name rp-nsg \
        --access Allow \
        --priority 500 \
        --source-address-prefixes "$(curl --silent ipecho.net/plain)/32" \
        --protocol Tcp \
        --destination-port-ranges 22
    ```

1. SSH into the VM
    ```bash
    VMSS_PIP=$(az vmss list-instance-public-ips -g $RESOURCEGROUP --name rp-vmss-$(git rev-parse --short=7 HEAD)$([[ $(git status --porcelain) = "" ]] || echo -dirty) | jq -r '.[0].ipAddress')

    ssh cloud-user@${VMSS_PIP}
    ```


## SSH to Gateway VMSS Instance

1. Update the Gateway NSG to allow SSH
    ```bash
    az network nsg rule create \
        --name ssh-to-gwy \
        --resource-group $USER-gwy-$LOCATION \
        --nsg-name gateway-nsg \
        --access Allow \
        --priority 500 \
        --source-address-prefixes "$(curl --silent ipecho.net/plain)/32" \
        --protocol Tcp \
        --destination-port-ranges 22
    ```


1. SSH into the VM
    ```bash
    VMSS_PIP=$(az vmss list-instance-public-ips -g $USER-gwy-$LOCATION --name gateway-vmss-$(git rev-parse --short=7 HEAD)$([[ $(git status --porcelain) = "" ]] || echo -dirty) | jq -r '.[0].ipAddress')

    ssh cloud-user@${VMSS_PIP}
    ```


## Deploy a Cluster

1. Add a NSG rule to allow tunneling to the RP instance

    ```bash
    az network nsg rule create \
        --name tunnel-to-rp \
        --resource-group $RESOURCEGROUP \
        --nsg-name rp-nsg \
        --access Allow \
        --priority 499 \
        --source-address-prefixes "$(curl --silent ipecho.net/plain)/32" \
        --protocol Tcp \
        --destination-port-ranges 443
    ```


1. Run the tunnel program to tunnel to the RP
    ```bash
    make tunnel
    ```

    > __NOTE:__ `make tunnel` will print the public IP of your new RP VM NIC. Ensure that it's correct.

1. Update environment variable to deploy in a different resource group
    ```bash
    export RESOURCEGROUP=myResourceGroup
    ```

1. Create the resource group if it doesn't exist
    ```bash
    az group create --resource-group $RESOURCEGROUP --location $LOCATION
    ```

1. Create VNets / Subnets
    ```bash
    az network vnet create \
        --resource-group $RESOURCEGROUP \
        --name aro-vnet \
        --address-prefixes 10.0.0.0/22
    ```

    ```bash
    az network vnet subnet create \
        --resource-group $RESOURCEGROUP \
        --vnet-name aro-vnet \
        --name master-subnet \
        --address-prefixes 10.0.0.0/23 \
        --service-endpoints Microsoft.ContainerRegistry
    ```

    ```bash
    az network vnet subnet create \
        --resource-group $RESOURCEGROUP \
        --vnet-name aro-vnet \
        --name worker-subnet \
        --address-prefixes 10.0.2.0/23 \
        --service-endpoints Microsoft.ContainerRegistry
    ```

1. Register your subscription with the resource provider (post directly to subscription cosmosdb container)
    ```bash
    curl -k -X PUT   -H 'Content-Type: application/json'   -d '{
        "state": "Registered",
        "properties": {
            "tenantId": "'"$AZURE_TENANT_ID"'",
            "registeredFeatures": [
                {
                    "name": "Microsoft.RedHatOpenShift/RedHatEngineering",
                    "state": "Registered"
                }
            ]
        }
    }' "https://localhost:8443/subscriptions/$AZURE_SUBSCRIPTION_ID?api-version=2.0"
    ```

1. Create the cluster
    ```bash
    export CLUSTER=$USER

    az aro create \
        --resource-group $RESOURCEGROUP \
        --name $CLUSTER \
        --vnet aro-vnet \
        --master-subnet master-subnet \
        --worker-subnet worker-subnet
    ```

    > __NOTE:__ The `az aro` CLI extension must be registered in order to run `az aro` commands against a local or tunneled RP. The usual hack script used to create clusters does not work due to keyvault mirroring requirements. The name of the cluster depends on the DNS zone that was created in an earlier step.
