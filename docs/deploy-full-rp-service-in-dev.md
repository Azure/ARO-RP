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
    ```

    ```bash
    . ./env
    ```

1. Run `make deploy`
    > __NOTE:__ This will fail on the first attempt to run due to certificate and container mirroring requirements.

1. Update the certificates in keyvault
<!-- TODO: this is almost duplicated elsewhere.  Would be nice to move to common area -->
    ```bash
    az keyvault certificate import \
        --vault-name "$KEYVAULT_PREFIX-svc" \
        --name rp-mdm \
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
    ```


1. Mirror the OpenShift images to your new ACR
    <!-- TODO (bv) allow mirroring through a pipeline would be faster and a nice to have -->

    1. Setup mirroring environment variables
        ```bash
        export DST_ACR_NAME=${USER}aro
        export SRC_AUTH_QUAY=FILL_IN # Get quay auth https://cloud.redhat.com/openshift/create/local -> Download Pull Secret
        export SRC_AUTH_REDHAT=$(echo $USER_PULL_SECRET | jq -r '.auths."registry.redhat.io".auth')
        export DST_AUTH=$(echo -n '00000000-0000-0000-0000-000000000000:'$(az acr login -n ${DST_ACR_NAME} --expose-token | jq -r .accessToken) | base64 -w0)

    1. Login to the Azure Container Registry
        ```bash
        docker login -u 00000000-0000-0000-0000-000000000000 -p "$(echo $DST_AUTH | base64 -d | cut -d':' -f2)" "${DST_ACR_NAME}.azurecr.io"
        ```

    1. Run the mirroring
        > The `latest` argument will take the InstallStream from `pkg/util/version/const.go` and mirror that version
        ```bash
        go run ./cmd/aro mirror latest
        ```

    1. Push the ARO image to your ACR
        ```bash
        make publish-image-aro-multistage
        ```

1. Delete the existing VMSS
    > __NOTE:__ This needs to be deleted as deploying won't recreate the VMSS if the commit hash is the same.
    ```bash
    az vmss delete -g ${RESOURCEGROUP} --name rp-vmss-$(git rev-parse --short=7 HEAD)$([[ $(git status --porcelain) = "" ]] || echo -dirty)
    ```

1. Run `make deploy`

## Deploying a cluster

1. Setup a local tunnel to the RP
    ```bash
    make tunnel
    ```

1. Deploy your cluster
    ```bash
    RESOURCEGROUP=v4-$LOCATION CLUSTER=bvesel go run ./hack/cluster create
    ```

    > __NOTE:__ The cluster will not be accessible via DNS unless you update the parent domain of the cluster.


## SSHing into RP VMSS Instance

1. Update the RP NSG to allow SSH
    ```bash
    az network nsg rule create \
        --name ssh-to-rp \
        --resource-group $RESOURCEGROUP \
        --nsg-name rp-nsg \
        --access Allow \
        --priority 140 \
        --source-address-prefixes "$(curl --silent ipecho.net/plain)/32" \
        --protocol Tcp \
        --destination-port-ranges 22
    ```

1. SSH into the VM
    ```bash
    VMSS_PIP=$(az vmss list-instance-public-ips -g $RESOURCEGROUP --name rp-vmss-$(git rev-parse --short=7 HEAD)$([[ $(git status --porcelain) = "" ]] || echo -dirty) | jq -r '.[0].ipAddress')

    ssh cloud-user@${VMSS_PIP}
    ```
