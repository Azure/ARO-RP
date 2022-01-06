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

1. Generate the development RP configuration
    ```bash
    make dev-config.yaml
    ```

1. Update and resource your environment file
    > It should look something like below once completed
    ```bash
    export LOCATION=eastus
    export ARO_IMAGE=arointsvc.azurecr.io/aro:latest

    . secrets/env

    export RESOURCEGROUP=$USER-aro-$LOCATION
    export DATABASE_ACCOUNT_NAME=$USER-aro-$LOCATION
    export DATABASE_NAME=ARO
    export KEYVAULT_PREFIX=$USER-aro-$LOCATION
    export ARO_IMAGE=${USER}aro.azurecr.io/aro:$(git rev-parse --short=7 HEAD)$([[ $(git status --porcelain) = "" ]] || echo -dirty)
    export FLUENTBIT_IMAGE=${USER}aro.azurecr.io/fluentbit:latest
    ```

    ```bash
    . ./env
    ```

1. Run `make deploy`
    > __NOTE:__ This will fail on the first attempt to run due to certificate and container mirroring requirements.
    > __NOTE:__ If you reuse an old name, you might run into soft-delete of the keyvaults. `az keyvault recover --name` to fix this.

<!-- TODO: this is almost duplicated elsewhere.  Would be nice to move to common area -->
1. Update the certificates in keyvault

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

1. Mirror the OpenShift images to your new ACR
    <!-- TODO (bv) allow mirroring through a pipeline would be faster and a nice to have -->

    1. Setup mirroring environment variables
        ```bash
        export DST_ACR_NAME=${USER}aro
        export SRC_AUTH_QUAY=$(echo $USER_PULL_SECRET | jq -r '.auths."quay.io".auth')
        export SRC_AUTH_REDHAT=$(echo $USER_PULL_SECRET | jq -r '.auths."registry.redhat.io".auth')
        export DST_AUTH=$(echo -n '00000000-0000-0000-0000-000000000000:'$(az acr login -n ${DST_ACR_NAME} --expose-token | jq -r .accessToken) | base64 -w0)

    1. Login to the Azure Container Registry
        ```bash
        docker login -u 00000000-0000-0000-0000-000000000000 -p "$(echo $DST_AUTH | base64 -d | cut -d':' -f2)" "${DST_ACR_NAME}.azurecr.io"
        ```

    1. Run the mirroring
        > The `latest` argument will take the InstallStream from `pkg/util/version/const.go` and mirror that version
        ```bash
        go run -tags aro ./cmd/aro mirror latest
        ```

    1. Push the ARO and Fluentbit images to your ACR
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

1. Delete the existing VMSS
    > __NOTE:__ This needs to be deleted as deploying won't recreate the VMSS if the commit hash is the same.
    ```bash
    az vmss delete -g ${RESOURCEGROUP} --name rp-vmss-$(git rev-parse --short=7 HEAD)$([[ $(git status --porcelain) = "" ]] || echo -dirty)

    az vmss delete -g $USER-gwy-$LOCATION --name gateway-vmss-$(git rev-parse --short=7 HEAD)$([[ $(git status --porcelain) = "" ]] || echo -dirty)
    ```

1. Run `make deploy`

## Deploying a cluster

1. Setup a local tunnel to the RP
    ```bash
    make tunnel
    ```

1. Deploy your cluster
    ```bash
    RESOURCEGROUP=v4-$LOCATION CLUSTER=$USER-cluster go run ./hack/cluster create
    ```

    > __NOTE:__ The cluster will not be accessible via DNS unless you update the parent domain of the cluster.


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

1. Run the tunnel program to tunnel to the RP
    ```bash
    make tunnel
    ```

1. Update environment variable to deploy in a different resource group
    ```bash
    export RESOURCEGROUP=myResourceGroup
    ```

1. Create the resource group if it doesn't exist
    ```bash
    az group create $RESOURCEGROUP --location $LOCATION
    ```

1. Create VNets / Subnets
    ```bash
    az network vnet create -g $RESOURCEGROUP -n dev-vnet --address-prefixes 10.0.0.0/8
    ```

1. Create the cluster
    ```bash
    export NO_INTERNET=true
    CLUSTER=$USER go run ./hack/cluster create
    ```
