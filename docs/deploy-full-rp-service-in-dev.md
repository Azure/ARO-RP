# Deploy an entire RP Development Service

## When to use it?

- Test the gateway and clusters' interactions with it.
- Test the actual VMSS instances we run the RP and gateway on.
- Test our deployments (since full-service uses mostly the same deployment setup as production).
- Test any Hive cluster changes without messing with the shared development Hive clusters.

## Prerequisites

1. Your development environment is prepared according to the steps outlined in [Prepare Your Dev Environment](./prepare-your-dev-environment.md)
2. During the deployment, it's recommended to avoid editing files in your
   ARO-RP repository so that `git status` reports a clean working tree.
   Otherwise aro container image will have `-dirty` suffix, which can be
   problematic:
    - if the working tree becomes dirty during the process (eg. because you
      create a temporary helper script to run some of the setup), you could end
      up with different image tag pushed in the azure image registry compared
      to the tag expected by aro deployer
    - with a dirty tag, it's not clear what's actually in the image

## Deploying an int-like Development RP

1. Fetch the most up-to-date secrets specifying `SECRET_SA_ACCOUNT_NAME` to the
   name of the storage account containing your shared development environment
   secrets, eg.:

   ```bash
   SECRET_SA_ACCOUNT_NAME=rharosecretsdev make secrets
   ```

1. Copy and tweak your environment file:

   ```bash
   cp env.example env
   vi env
   ```

   You don't need to change anything in the env file, unless you plan on using
   hive to install the cluster. In that case add the following hive environment
   variables into your env file:

   ```bash
   export ARO_INSTALL_VIA_HIVE=true
   export ARO_ADOPT_BY_HIVE=true
   ```

1. Create a full environment file, which overrides some defaults from `./env` options when sourced

    ```bash
    cp env-int.example env-int
    vi env-int
    ```

    What to change in `env-int` file:

    * if using a public key separate from `~/.ssh/id_rsa.pub` (for ssh access to RP and Gateway vmss instances), source it with `export SSH_PUBLIC_KEY=~/.ssh/id_separate.pub`
    - Modify `AZURE_PREFIX` environment variable for a different prefix of the created Azure resources.
    * set tag of `FLUENTBIT_IMAGE` value to match the default from `pkg/util/version/const.go`,
      eg. `FLUENTBIT_IMAGE=${AZURE_PREFIX}aro.azurecr.io/fluentbit:1.9.10-cm20230426`
    * if you actually care about fluentbit image version, you need to change the default both in the env-int file and for ARO Deployer, which is out of scope of this guide

1. And finally source the env:

    ```bash
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
        ./hack/hive/hive-generate-config.sh
        KUBECONFIG=$(pwd)/aks.kubeconfig ./hack/hive/hive-dev-install.sh
        ```

1. Mirror the OpenShift images to your new Azure Container Registry (ACR)
    <!-- TODO (bv) allow mirroring through a pipeline would be faster and a nice to have -->
    > __NOTE:__ Running the mirroring through a VM in Azure rather than a local workstation is recommended for better performance.
    > __NOTE:__ Value of `USER_PULL_SECRET` variable comes from the secrets, which are sourced via `env-int` file
    > __NOTE:__ `DST_AUTH` token or the login to the registry expires after some time

    1. Setup mirroring environment variables
        ```bash
        export DST_ACR_NAME=${AZURE_PREFIX}aro
        export SRC_AUTH_QUAY=$(echo $USER_PULL_SECRET | jq -r '.auths."quay.io".auth')
        export SRC_AUTH_REDHAT=$(echo $USER_PULL_SECRET | jq -r '.auths."registry.redhat.io".auth')
        export DST_AUTH=$(echo -n '00000000-0000-0000-0000-000000000000:'$(az acr login -n ${DST_ACR_NAME} --expose-token | jq -r .accessToken) | base64 -w0)
        ```

    1. Login to the Azure Container Registry
        ```bash
        docker login -u 00000000-0000-0000-0000-000000000000 -p "$(echo $DST_AUTH | base64 -d | cut -d':' -f2)" "${DST_ACR_NAME}.azurecr.io"
        ```

   1. Run the mirroring

      > The `latest` argument will take the DefaultInstallStream from `pkg/util/version/const.go` and mirror that version

        ```bash
        go run ./cmd/aro mirror latest
        ```

      > __Troubleshooting:__ There could be some issues when mirroring the images to the ACR related to missing _devmapper_ or _btrfs_ (usually with "fatal error: btrfs/ioctl.h: No such file or directory" error) packages.
      If respectively installing _device-mapper-devel_ or _btrfs-progs-devel_ packages won't help, then you may ignore them as follows:

        ```bash
        go run -tags=exclude_graphdriver_devicemapper,exclude_graphdriver_btrfs ./cmd/aro mirror latest
        ```

        If you are going to test or work with multi-version installs, then you should mirror any additional versions as well, for example for 4.11.21 it would be

        ```bash
        go run ./cmd/aro mirror 4.11.21
        ```

   1. Mirror upstream distroless Geneva MDM/MDSD images to your ACR

        Run the following commands to mirror two Microsoft Geneva images based on the tags from [pkg/util/version/const.go](https://github.com/Azure/ARO-RP/blob/master/pkg/util/version/const.go) (e.g., 2.2024.517.533-b73893-20240522t0954 and mariner_20240524.1).

        ```bash
            source hack/devtools/rp_dev_helper.sh
            mdm_image_tag=$(get_digest_tag "MdmImage")
            az acr import --name $DST_ACR_NAME.azurecr.io$mdm_image_tag --source linuxgeneva-microsoft.azurecr.io$mdm_image_tag
            mdsd_image_tag=$(get_digest_tag "MdsdImage")
            az acr import --name $DST_ACR_NAME.azurecr.io$mdsd_image_tag --source linuxgeneva-microsoft.azurecr.io$mdsd_image_tag
        ```

   1. Push the ARO image to your ACR

        > If running this step from a VM separate from your workstation, ensure the commit tag used to build the image matches the commit tag where `make deploy` is run.

        > For local builds and CI builds without `RP_IMAGE_ACR` environment
        > variable set, `make publish-image-*` targets will pull from
        > `registry.access.redhat.com`.
        > If you need to use Azure container registry instead due to security
        > compliance requirements, modify the `RP_IMAGE_ACR` environment
        > variable to point to `arointsvc` or `arosvc` instead. You will need
        > authenticate to this registry using `az acr login --name arointsvc`
        > to pull the images.

        > If the push fails on error like `unable to retrieve auth token:
        > invalid username/password: unauthorized: authentication required`,
        > try to create `DST_AUTH` variable and login to the container
        > registry (as explained in steps above) again. It will resolve the
        > failure in case of an expired auth token.

        ```bash
        make publish-image-aro-multistage
        ```

    1. Copy the Fluentbit image from arointsvc ACR to your ACR

        ```bash
        source hack/devtools/rp_dev_helper.sh
        fluentbit_image_tag=$(get_digest_tag "FluentbitImage")
        copy_digest_tag $PULL_SECRET "arointsvc" $DST_ACR_NAME $fluentbit_image_tag
        ```
    1. Copy the Mise image from arointsvc ACR to your ACR

        > Mise is not enabled as of now for dev or full rp service but
        > its image is required in your acr for the deploy to not fail
        > while trying to pull it as required by systemd service. 

        ```bash
        source hack/devtools/rp_dev_helper.sh
        mise_image_tag=$(get_digest_tag "MiseImage")
        copy_digest_tag $PULL_SECRET "arointsvc" $DST_ACR_NAME $mise_image_tag
        ```
    

1. Update the DNS Child Domains
    ```bash
    export PARENT_DOMAIN_NAME=osadev.cloud
    export PARENT_DOMAIN_RESOURCEGROUP=dns
    export GLOBAL_RESOURCEGROUP=$AZURE_PREFIX-global

    for DOMAIN_NAME in $AZURE_PREFIX-clusters.$PARENT_DOMAIN_NAME $AZURE_PREFIX-rp.$PARENT_DOMAIN_NAME; do
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

1. Update the certificates in keyvault
    <!-- TODO: this is almost duplicated elsewhere.  Would be nice to move to common area -->
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
    ```

1. Delete the existing VMSS
    > __NOTE:__ This needs to be deleted as deploying won't recreate the VMSS if the commit hash is the same.

    ```bash
    az vmss delete -g ${RESOURCEGROUP} --name rp-vmss-$(git rev-parse --short=7 HEAD)$([[ $(git status --porcelain) = "" ]] || echo -dirty) && az vmss delete -g $AZURE_PREFIX-gwy-$LOCATION --name gateway-vmss-$(git rev-parse --short=7 HEAD)$([[ $(git status --porcelain) = "" ]] || echo -dirty)
    ```

1. Run `make deploy`. When the command finishes, there should be one VMSS for
   the RP with a single vm instance, and another VMSS with a single vm for
   Gateway.

1. Create additional infrastructure required for workload identity clusters
    ```
    source ./hack/devtools/deploy-shared-env.sh
    deploy_miwi_infra_for_dedicated_rp
    ```

1. If you are going to use multiversion, you can now update the OpenShiftVersions DB as per [OpenShift Version insttructions](./deploy-development-rp.md#openshift-version)

## SSH to RP VMSS Instance

1. Update the RP NSG to allow SSH
    ```bash
    az network nsg rule create \
        --name ssh-to-rp \
        --resource-group $RESOURCEGROUP \
        --nsg-name rp-nsg \
        --access Allow \
        --priority 500 \
        --source-address-prefixes "$(curl --silent -4 ipecho.net/plain)/32" \
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
        --resource-group $AZURE_PREFIX-gwy-$LOCATION \
        --nsg-name gateway-nsg \
        --access Allow \
        --priority 500 \
        --source-address-prefixes "$(curl --silent -4 ipecho.net/plain)/32" \
        --protocol Tcp \
        --destination-port-ranges 22
    ```


1. SSH into the VM
    ```bash
    VMSS_PIP=$(az vmss list-instance-public-ips -g $AZURE_PREFIX-gwy-$LOCATION --name gateway-vmss-$(git rev-parse --short=7 HEAD)$([[ $(git status --porcelain) = "" ]] || echo -dirty) | jq -r '.[0].ipAddress')

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
        --source-address-prefixes "$(curl --silent -4 ipecho.net/plain)/32" \
        --protocol Tcp \
        --destination-port-ranges 443
    ```


1. Run the tunnel program to tunnel to the RP
    ```bash
    make tunnel
    ```

    > __NOTE:__ `make tunnel` will print the public IP of your new RP VM NIC. Ensure that it's correct.

1. Update the versions present available to install (run this as many times as you need for versions)
    ```bash
    curl -X PUT -k "https://localhost:8443/admin/versions" --header "Content-Type: application/json" -d '{ "properties": { "version": "4.x.y", "enabled": true, "openShiftPullspec": "quay.io/openshift-release-dev/ocp-release@sha256:<sha256>", "installerPullspec": "<name>.azurecr.io/installer:release-4.x" }}'
    ```

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

## Recover VPN access

Since setting up your own VPN in an earlier step will overwrite your local secrets, you will lose access to the vpn / vnet gateway that you provisioned in an earlier step if you run `make secrets`. If you don't have a secrets/* backup, you can recover your access using the following steps. Please note that this action will _**OVER WRITE**_ the `secrets/vpn-$LOCATION.ovpn` on your local machine. **DO NOT** run `make secrets-update` after doing this, as you will overwrite the shared secrets for all users.

1. Source all environment variables from earlier, and run the VPN configuration step again:

    ```bash
    . ./env
    . ./env-int

    source ./hack/devtools/deploy-shared-env.sh
    vpn_configuration
    ```

1. Create new VPN certificates locally:

    ```bash
    go run ./hack/genkey -ca vpn-ca
    mv vpn-ca.* secrets
    go run ./hack/genkey -client -keyFile secrets/vpn-ca.key -certFile secrets/vpn-ca.crt vpn-client
    mv vpn-client.* secrets
    ```

1. Update the VPN configuration locally:
    - Add the new cert and key created above (located in `secrets/vpn-client.pem`) to `secrets/vpn-eastus.ovpn`, replacing the existing configuration.

1. Add the newly created secrets to the `dev-vpn` vnet gateway in `$USER-aro-$LOCATION` resource group:
    - In portal, navigate to `dev-vpn`, Point-to-site configuration > Root certificates.
    - Add the new `secrets/vpn-ca.pem` data created above to this configuration.

1. Connect to the VPN:
    ```bash
    sudo openvpn secrets/vpn-$LOCATION.ovpn
    ```
