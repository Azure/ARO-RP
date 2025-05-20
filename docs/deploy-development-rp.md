# Deploy development RP

## Why to use it?
This is the **preferred** and fast way to have your own local development RP setup, while also having a functional cluster.
It uses hacks scripts around a lot of the setup to make things easier to bootstrap and be more sensible for running off of your local laptop.

- Check the specific use-case examples where [deploying full RP service](https://github.com/Azure/ARO-RP/blob/master/docs/deploy-full-rp-service-in-dev.md) can be a better match.

## Prerequisites

1. Your development environment is prepared according to the steps outlined in [Prepare Your Dev Environment](./prepare-your-dev-environment.md)

## Installing the extension

1. Check the `env.example` file and copy it by creating your own:

   ```bash
   cp env.example env
   ```

2. Build the development `az aro` extension:

   ```bash
   . ./env
   make az
   ```

3. Verify the ARO extension is registered:

   ```bash
   az -v
   ...
   Extensions:
   aro                                0.4.0 (dev) /path/to/rp/python/az/aro
   ...
   Development extension sources:
       /path/to/rp/python
   ...
   ```

   Note: you will be able to update your development `az aro` extension in the
   future by simply running `git pull`. If you need to use the "prod" extension,
   what is bundled in `az` natively rather than your `./python`, you can
   `unset AZURE_EXTENSION_DEV_SOURCES` (found in your `./env` file).

## Prepare your environment

1. If you don't have access to a shared development environment and secrets,
   follow [prepare a shared RP development
   environment](prepare-a-shared-rp-development-environment.md).

1. If you have multiple subscriptions in your account, verify that "ARO SRE Team - InProgress (EA Subscription 2)" is your active subscription:
   ```bash
   az account set --subscription "ARO SRE Team - InProgress (EA Subscription 2)"
   ```
1. Set SECRET_SA_ACCOUNT_NAME to the name of the storage account containing your
   shared development environment secrets and save them in `secrets`:

   ```bash
   SECRET_SA_ACCOUNT_NAME=rharosecretsdev make secrets
   ```

1. Create your own RP database (if you don't already have one in the $LOCATION):

    * The following command can be used to check whether a DB already exists
        ```bash
        az deployment group list -g "$RESOURCEGROUP" -o table | grep "databases-development-${AZURE_PREFIX:-$USER}"
        ```

    * This is how you create one, if needed
      ```bash
      az deployment group create \
        -g "$RESOURCEGROUP" \
        -n "databases-development-${AZURE_PREFIX:-$USER}" \
        --template-file pkg/deploy/assets/databases-development.json \
        --parameters \
          "databaseAccountName=$DATABASE_ACCOUNT_NAME" \
          "databaseName=$DATABASE_NAME" \
        1>/dev/null
      ```

### Mock MSI setup required for MIWI installs

1. Run [msi.sh](../hack/devtools/msi.sh) to create a service principal and self-signed certificate to
mock a cluster MSI. This script will also create the platform identities, platform identity role assignments, and role assignment on mock cluster MSI to federate the platform identities. Platform identities will be created in resource group `RESOURCEGROUP` and subscription `SUBSCRIPTION`. Save the output values for cluster MSI `Client ID`, `Base64 Encoded Certificate`, and `Tenant`. Additionally, save the value for `Platform workload identity role sets`.

1. Copy, edit (if necessary) and source your environment file. The required
   environment variable configuration is documented immediately below:

   ```bash
   cp env.example env
   vi env
   . ./env
   ```

   - `LOCATION`: Location of the shared RP development environment (default:
     `eastus`).
   - `RP_MODE`: Set to `development` to use a development RP running at
     https://localhost:8443/.
   
### MIWI setup

1. Create a resource group for your cluster and managed identities

1. Source the local dev script and run the command to set up the wimi env file for you

   ```bash
   source ./hack/devtools/local_dev_env.sh
   CLUSTER_RESOURCEGROUP=<your cluster resourcegroup> create_miwi_env_file
   ```

1. Ensure that the following environment variables were set in your env file, and re-source it:

   - `MOCK_MSI_CLIENT_ID`: Client ID for service principal that mocks cluster MSI (see previous step).
   - `MOCK_MSI_OBJECT_ID`: Object ID for service principal that mocks cluster MSI (see previous step).
   - `MOCK_MSI_CERT`: Base64 encoded certificate for service principal that mocks cluster MSI (see previous step).
   - `MOCK_MSI_TENANT_ID`: Tenant ID for service principal that mocks cluster MSI (see previous step).
   - `PLATFORM_WORKLOAD_IDENTITY_ROLE_SETS`: The platform workload identity role sets (see previous step or value in `local_dev_env.sh`).

1. Connect to the VPN and populate the platform workload identity role set definitions to your CosmosDB instance

   **Note** if installing a version other than 4.14 you will need to change your local `PLATFORM_WORKLOAD_IDENTITY_ROLE_SETS` env var to point to your desired version

   - `go run ./cmd/aro update-role-sets`

1. Add a new installable OCP version to your local RP instance. This version should be a 4.14.38+ or 4.15.35+ version and use one of the current aro-installer images in our INT repo

   - for 4.16
   ```
   curl -X PUT -k "https://localhost:8443/admin/versions" --header "Content-Type: application/json" -d '{ "properties": { "version": "4.16.30", "enabled": true, "openShiftPullspec": "quay.io/openshift-release-dev/ocp-release@sha256:7aacace57ab6ec468dd98b0b3e0f3fc440b29afce21b90bd716fed0db487e9e9", "installerPullspec": "arosvc.azurecr.io/aro-installer:4.16@sha256:27871abbc88cdfda21c81ed1a00050e71df8c88b4bb53f96104f6d0661c0b9bf"}}'
   ```

   - for 4.15
   ```
   curl -X PUT -k "https://localhost:8443/admin/versions" --header "Content-Type: application/json" -d '{ "properties": { "version": "4.15.35", "enabled": true, "openShiftPullspec": "quay.io/openshift-release-dev/ocp-release@sha256:8c8433f95d09b051e156ff638f4ccc95543918c3aed92b8c09552a8977a2a1a2", "installerPullspec": "arointsvc.azurecr.io/aro-installer@sha256:e733a9b3fe549273098d7b6acd6b45a84819020f4170a6062a8185661417fe91"}}'
   ```

   - for 4.14
   ```
   curl -X PUT -k "https://localhost:8443/admin/versions" --header "Content-Type: application/json" -d '{ "properties": { "version": "4.14.38", "enabled": true, "openShiftPullspec": "quay.io/openshift-release-dev/ocp-release@sha256:98e43d1e848f0ad303ed4d8d427e92f7aaeaf2f670a3bfcdbeeeaa591b63fefd", "installerPullspec": "arointsvc.azurecr.io/aro-installer@sha256:e084ce2895fd1356d07e7f8a47f79ac43b75e3a146b211c843f58d5cb88d9c70"}}'
   ```

## Run the RP and create a cluster

1. Source your environment file.

   ```bash
   . ./env
   ```

1. Run the RP

    Option 1: using local compilation and binaries (requires local `go`/build dependencies/etc):
    ```bash
    make runlocal-rp
    ```

    Option 2: using containerized build and run (requires local `podman` and `openvpn`):
    ```bash
    # establish a VPN connection to the shared dev environment Hive cluster
    sudo openvpn secrets/vpn-${LOCATION}.ovpn &

    # build/run the RP as a container
    make run-rp
    ```

### Create a service principal cluster

1. To create a cluster, use one of the following methods:

   **NOTE:** clusters created by a local dev RP will not be represented by Azure resources in ARM

   1. Manually create the cluster using the public documentation.

      Before following the instructions in [Create, access, and manage an Azure Red Hat
      OpenShift 4 Cluster][1],
      you will need to add the OCP version you want to install to your DB by
      [put(ting) a new OpenShift installation version](#openshift-version),
      and you will also need to manually register your subscription to your local RP:

      ```bash
      curl -k -X PUT -H 'Content-Type: application/json' -d '{
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

      Note that, as there is no default version defined, you will need to provide the `--version` argument
      to `az aro create` with one of the versions you added to your DB.
      
      Note also that as long as the `RP_MODE` environment variable is set to `development`, the `az aro` client will
      connect to your local RP.

   1. Use the create utility:

      ```bash
      # Create the cluster
      CLUSTER=<cluster-name> go run ./hack/cluster create
      ```

      Later the cluster can be deleted as follows:

      ```bash
      CLUSTER=<cluster-name> go run ./hack/cluster delete
      ```

      By default, a public cluster will be created. In order to create a private cluster, set the `PRIVATE_CLUSTER` environment variable to `true` prior to creation. Internet access from the cluster can also be restricted by setting the `NO_INTERNET` environment variable to `true`.

   > **NOTE:** If the cluster creation fails with `unable to connect to Podman socket...dial unix ///run/user/1000/podman/podman.sock: connect: no such file or directory`, then you will need enable podman user socket by executing : `systemctl --user enable --now podman.socket`, and re-run the installation.

   [1]: https://docs.microsoft.com/en-us/azure/openshift/tutorial-create-cluster

### Create a MIWI cluster

1. Ensure the required environment variables are set:

   - `make aks.kubeconfig`
   - `export ARO_INSTALL_VIA_HIVE=true` : instructs the RP to use hive to install clusters
   - `export HIVE_KUBE_CONFIG_PATH=$PWD/aks.kubeconfig` : where to look for the kubeconfig

1. Ensure that the required platform workload identities were created in your resource group. If they haven't been, run `./hack/devtools/msi.sh`

   - aro-cloud-controller-manager
   - aro-ingress
   - aro-machine-api
   - aro-disk-csi-driver
   - aro-cloud-network-config
   - aro-image-registry
   - aro-file-csi-driver
   - aro-aro-operator
   - aro-Cluster

1. Create the cluster

   **Note** If the identities are not in the same resource group as the cluster, you can optionally use full resource IDs for each managed and cluster identity

   ```bash
   az aro create \
   --location ${LOCATION} \
   --resource-group ${CLUSTER_RESOURCEGROUP} \
   --name ${CLUSTER_NAME} \
   --vnet ${CLUSTER_VNET} \
   --master-subnet master-subnet \
   --worker-subnet worker-subnet \
   --version 4.15.35 \
   --master-vm-size Standard_D8s_v5 \
   --enable-managed-identity \
   --assign-cluster-identity aro-Cluster \
   --assign-platform-workload-identity file-csi-driver aro-file-csi-driver \
   --assign-platform-workload-identity cloud-controller-manager aro-cloud-controller-manager \
   --assign-platform-workload-identity ingress aro-ingress \
   --assign-platform-workload-identity image-registry aro-image-registry \
   --assign-platform-workload-identity machine-api aro-machine-api \
   --assign-platform-workload-identity cloud-network-config aro-cloud-network-config \
   --assign-platform-workload-identity aro-operator aro-aro-operator \
   --assign-platform-workload-identity disk-csi-driver aro-disk-csi-driver
   ```

## Interact with the cluster

1. Using `oc`:

   1. Get the KUBECONFIG:

      ```
      az aro get-admin-kubeconfig \
        --name <cluster-name> \
        --resource-group <resource_group_name> \
        --file dev.admin.kubeconfig;
      export KUBECONFIG=dev.admin.kubeconfig
      ```

      The cluster and resource group names can be found by running `az aro list -o table`

   1. Setup insecure-tls `oc`:

      To interact with the cluster using using `oc` you will need to add the `--insecure-skip-tls-verify` flag to all commands or use this handy alias:

      ```
        alias oc-dev='oc --insecure-skip-tls-verify'
      ```

      Note: This is because the cluster is using self-signed certificates.

1. The following additional RP endpoints are available but not exposed via `az
aro`:

   - Delete a subscription, cascading deletion to all its clusters:

     ```bash
     curl -k -X PUT \
       -H 'Content-Type: application/json' \
       -d '{"state": "Deleted", "properties": {"tenantId": "'"$AZURE_TENANT_ID"'"}}' \
       "https://localhost:8443/subscriptions/$AZURE_SUBSCRIPTION_ID?api-version=2.0"
     ```

   - List operations:

     ```bash
     curl -k \
       "https://localhost:8443/providers/Microsoft.RedHatOpenShift/operations?api-version=2020-04-30"
     ```

   - View RP logs in a friendly format:

     ```bash
     journalctl _COMM=aro -o json --since "15 min ago" -f | jq -r 'select (.COMPONENT != null and (.COMPONENT | contains("access"))|not) | .MESSAGE'
     ```

   - Optionally, create these aliases for viewing logs
     ```bash
     cat >>~/.bashrc <<'EOF'
     alias rp-logs='journalctl _COMM=aro -o json --since "15 min ago" -f | jq -r '\''select (.COMPONENT != null and (.COMPONENT | contains("access"))|not) | .MESSAGE'\'''
     alias rp-logs-all='journalctl _COMM=aro -o json -e | jq -r '\''select (.COMPONENT != null and (.COMPONENT | contains("access"))|not) | .MESSAGE'\'''
     EOF
     ```

### Use a custom installer

Sometimes you want to use a custom installer, for example, when you want to test a new OCP version's installer.
You can create a cluster with the new installer following these steps:

1. Push the installer image to somewhere accessible from Hive AKS.

   [quay.io](https://quay.io/) would be one of the options.
   You need pull-secret to use the repositories other than `arointsvc.azurecr.io`.
   It must be configured in the secrets.
   If you are using the hack script, you don't have to care about it because the script uses `USER_PULL_SECRET` automatically.

1. [Run the RP](#run-the-rp-and-create-a-cluster)
1. [Update the OpenShift installer version](#openshift-version)

1. Create a cluster with the version you updated.

   If you are using the hack script, you can specify the version with `OS_CLUSTER_VERSION` env var.

## Automatically run local RP

If you are already familiar with running the ARO RP locally, you can speed up the process executing the [local_dev_env.sh](../hack/devtools/local_dev_env.sh) script.

## Connect ARO-RP with a Hive development cluster

The env variables names defined in pkg/util/liveconfig/manager.go control the communication of the ARO-RP with Hive.

- If you want to use ARO-RP + Hive, set `HIVE_KUBE_CONFIG_PATH` to the path of the kubeconfig of the AKS Dev cluster. [Info](https://github.com/Azure/ARO-RP/blob/master/docs/deploy-development-rp.md#debugging-aks-cluster) about creating that kubeconfig (Step _Access the cluster via API_).
- If you want to create clusters using the local ARO-RP + Hive instead of doing the standard cluster creation process (which doesn't use Hive), set `ARO_INSTALL_VIA_HIVE` to _true_.
- If you want to enable the Hive adoption feature (which is performed during adminUpdate()), set `ARO_ADOPT_BY_HIVE` to _true_.

After setting the above environment variables (using _export_ directly in the terminal or including them in the _env_ file), connect to the [VPN](https://github.com/Azure/ARO-RP/blob/master/docs/deploy-development-rp.md#debugging-aks-cluster) (_Connect to the VPN_ section).

**Warning:** Hive do not support OpenShift image referenced by tag (like installer in container does) but only with sha, so make sure version you are installing is defined with OpenShiftPullSpec defined with sha and not tag.

Then proceed to [run](https://github.com/Azure/ARO-RP/blob/master/docs/deploy-development-rp.md#run-the-rp-and-create-a-cluster) the ARO-RP as usual.

After that, when you [create](https://github.com/Azure/ARO-RP/blob/master/docs/deploy-development-rp.md#run-the-rp-and-create-a-cluster) a cluster, you will be using Hive behind the scenes. You can check the created Hive objects following [Debugging OpenShift Cluster](https://github.com/Azure/ARO-RP/blob/master/docs/deploy-development-rp.md#debugging-openshift-cluster) and using the _oc_ command.

## Make Admin-Action API call(s) to a running local-rp

```bash
export CLUSTER=<cluster-name>
export AZURE_SUBSCRIPTION_ID=<subscription-id>
export RESOURCEGROUP=<resource-group-name>
  [OR]
. ./env
```

- Perform AdminUpdate on a dev cluster

  ```bash
  curl -X PATCH -k "https://localhost:8443/subscriptions/$AZURE_SUBSCRIPTION_ID/resourceGroups/$RESOURCEGROUP/providers/Microsoft.RedHatOpenShift/openShiftClusters/$CLUSTER?api-version=admin" --header "Content-Type: application/json" -d "{}"
  ```

- Get Cluster details of a dev cluster

  ```bash
  curl -X GET -k "https://localhost:8443/subscriptions/$AZURE_SUBSCRIPTION_ID/resourceGroups/$RESOURCEGROUP/providers/Microsoft.RedHatOpenShift/openShiftClusters/$CLUSTER?api-version=admin" --header "Content-Type: application/json" -d "{}"
  ```

- Get SerialConsole logs of a VM of dev cluster

  ```bash
  VMNAME="aro-cluster-qplnw-master-0"
  curl -X GET -k "https://localhost:8443/admin/subscriptions/$AZURE_SUBSCRIPTION_ID/resourceGroups/$RESOURCEGROUP/providers/Microsoft.RedHatOpenShift/openShiftClusters/$CLUSTER/serialconsole?vmName=$VMNAME" --header "Content-Type: application/json" -d "{}"
  ```

- Redeploy a VM in a dev cluster

  ```bash
  VMNAME="aro-cluster-qplnw-master-0"
  curl -X POST -k "https://localhost:8443/admin/subscriptions/$AZURE_SUBSCRIPTION_ID/resourceGroups/$RESOURCEGROUP/providers/Microsoft.RedHatOpenShift/openShiftClusters/$CLUSTER/redeployvm?vmName=$VMNAME" --header "Content-Type: application/json" -d "{}"
  ```

- Stop a VM in a dev cluster

  ```bash
  VMNAME="aro-cluster-qplnw-master-0"
  curl -X POST -k "https://localhost:8443/admin/subscriptions/$AZURE_SUBSCRIPTION_ID/resourceGroups/$RESOURCEGROUP/providers/Microsoft.RedHatOpenShift/openShiftClusters/$CLUSTER/stopvm?vmName=$VMNAME" --header "Content-Type: application/json" -d "{}"
  ```

- Stop and [deallocate a VM](https://learn.microsoft.com/en-us/azure/virtual-machines/states-billing) in a dev cluster

  ```bash
  VMNAME="aro-cluster-qplnw-master-0"
  curl -X POST -k "https://localhost:8443/admin/subscriptions/$AZURE_SUBSCRIPTION_ID/resourceGroups/$RESOURCEGROUP/providers/Microsoft.RedHatOpenShift/openShiftClusters/$CLUSTER/stopvm?vmName=$VMNAME&deallocateVM=True" --header "Content-Type: application/json" -d "{}"
  ```

- Start a VM in a dev cluster

  ```bash
  VMNAME="aro-cluster-qplnw-master-0"
  curl -X POST -k "https://localhost:8443/admin/subscriptions/$AZURE_SUBSCRIPTION_ID/resourceGroups/$RESOURCEGROUP/providers/Microsoft.RedHatOpenShift/openShiftClusters/$CLUSTER/startvm?vmName=$VMNAME" --header "Content-Type: application/json" -d "{}"
  ```

- List VM Resize Options for a master node of dev cluster

  ```bash
  curl -X GET -k "https://localhost:8443/admin/subscriptions/$AZURE_SUBSCRIPTION_ID/resourceGroups/$RESOURCEGROUP/providers/Microsoft.RedHatOpenShift/openShiftClusters/$CLUSTER/skus" --header "Content-Type: application/json" -d "{}"
  ```

- Resize master node of a dev cluster

  ```bash
  VMNAME="aro-cluster-qplnw-master-0"
  VMSIZE="Standard_D16s_v3"
  curl -X POST -k "https://localhost:8443/admin/subscriptions/$AZURE_SUBSCRIPTION_ID/resourceGroups/$RESOURCEGROUP/providers/Microsoft.RedHatOpenShift/openShiftClusters/$CLUSTER/resize?vmName=$VMNAME&vmSize=$VMSIZE" --header "Content-Type: application/json" -d "{}"
  ```

- List Clusters of a local-rp

  ```bash
  curl -X GET -k "https://localhost:8443/admin/providers/microsoft.redhatopenshift/openshiftclusters"
  ```

- List cluster Azure Resources of a dev cluster

  ```bash
  curl -X GET -k "https://localhost:8443/admin/subscriptions/$AZURE_SUBSCRIPTION_ID/resourceGroups/$RESOURCEGROUP/providers/Microsoft.RedHatOpenShift/openShiftClusters/$CLUSTER/resources"
  ```

- Perform Cluster Upgrade on a dev cluster

  ```bash
  curl -X POST -k "https://localhost:8443/admin/subscriptions/$AZURE_SUBSCRIPTION_ID/resourceGroups/$RESOURCEGROUP/providers/Microsoft.RedHatOpenShift/openShiftClusters/$CLUSTER/upgrade"
  ```

- Get container logs from an OpenShift pod in a cluster

  ```bash
  NAMESPACE=<namespace-name>
  POD=<pod-name>
  CONTAINER=<container-name>
  curl -X GET -k "https://localhost:8443/admin/subscriptions/$AZURE_SUBSCRIPTION_ID/resourceGroups/$RESOURCEGROUP/providers/Microsoft.RedHatOpenShift/openShiftClusters/$CLUSTER/kubernetespodlogs?podname=$POD&namespace=$NAMESPACE&container=$CONTAINER"
  ```

- List Supported VM Sizes

  ```bash
  VMROLE=<master or worker>
  curl -X GET -k "https://localhost:8443/admin/supportedvmsizes?vmRole=$VMROLE"
  ```

- Perform Etcd Recovery Operation on a cluster

  ```bash
  curl -X PATCH -k "https://localhost:8443/admin/subscriptions/$AZURE_SUBSCRIPTION_ID/resourceGroups/$RESOURCEGROUP/providers/Microsoft.RedHatOpenShift/openShiftClusters/$CLUSTER/etcdrecovery"
  ```

- Delete a managed resource
  ```bash
  MANAGED_RESOURCEID=<id of managed resource to delete>
  curl -X POST -k "https://localhost:8443/admin/subscriptions/$AZURE_SUBSCRIPTION_ID/resourceGroups/$RESOURCEGROUP/providers/Microsoft.RedHatOpenShift/openShiftClusters/$CLUSTER/deletemanagedresource?managedResourceID=$MANAGED_RESOURCEID"
  ```

## OpenShift Version

- We have a cosmos container which contains supported installable OCP versions, more information on the definition in `pkg/api/openshiftversion.go`.

- Admin - List OpenShift installation versions

  ```bash
  curl -X GET -k "https://localhost:8443/admin/versions"
  ```

- Admin - Put a new OpenShift installation version

This command adds the image to your cosmosDB. **openShiftPullspec** comes from [quay.io/repository/openshift-release-dev](https://quay.io/repository/openshift-release-dev/ocp-release?tab=tags) (in production we must use sha tag, but in dev we can use tag for simplicity) ; and **installerPullspec** from int to work in dev without having to set any secret, but you can use repo for installer if you configured secret for it.

  ```bash
  OCP_VERSION=<x.y.z>
  curl -X PUT -k "https://localhost:8443/admin/versions" --header "Content-Type: application/json" -d '
    {
      "name": "'${OCP_VERSION}'",
      "type": "Microsoft.RedHatOpenShift/OpenShiftVersion",
      "properties":
      {
        "version": "'${OCP_VERSION}'",
        "enabled": true,
        "openShiftPullspec": "quay.io/openshift-release-dev/ocp-release:'${OCP_VERSION}'-x86_64",
        "installerPullspec": "arointsvc.azurecr.io/aro-installer:release-'${OCP_VERSION%.*}'"
      }
    }
  '
  ```

If you want to run the installer version via hive and not in container, you will need to use sha instead of tag for OCP image, and you can use your docker connection for this:
  ```bash
  docker login quay.io                                                                                                                                                                                                                                                                              16:36:10
  OCP_VERSION=<x.y.z>
  docker pull quay.io/openshift-release-dev/ocp-release:${OCP_VERSION}-x86_64
  curl -X PUT -k "https://localhost:8443/admin/versions" --header "Content-Type: application/json" -d '
    {
      "name": "'${OCP_VERSION}'",
      "type": "Microsoft.RedHatOpenShift/OpenShiftVersion",
      "properties":
      {
        "version": "'${OCP_VERSION}'",
        "enabled": true,
        "openShiftPullspec": "'$(docker inspect --format='{{index .RepoDigests 0}}' quay.io/openshift-release-dev/ocp-release:${OCP_VERSION}-x86_64)'",
        "installerPullspec": "arointsvc.azurecr.io/aro-installer:release-'${OCP_VERSION%.*}'"
      }
    }
  '
  ```

- List the enabled OpenShift installation versions within a region
  ```bash
  curl -X GET -k "https://localhost:8443/subscriptions/$AZURE_SUBSCRIPTION_ID/providers/Microsoft.RedHatOpenShift/locations/$LOCATION/openshiftversions?api-version=2022-09-04"
  ```

## Debugging OpenShift Cluster

- SSH to the bootstrap node:

  > **NOTE:** If you have a password-based `sudo` command, you must first authenticate before running `sudo` in the background

  ```bash
  sudo openvpn secrets/vpn-$LOCATION.ovpn &
  CLUSTER=cluster hack/ssh-agent.sh bootstrap
  ```

- Get an admin kubeconfig:

  ```bash
  CLUSTER=cluster make admin.kubeconfig
  export KUBECONFIG=admin.kubeconfig
  ```

- "SSH" to a cluster node:

  - Get the admin kubeconfig and `export KUBECONFIG` as detailed above.
  - Run the ssh-agent.sh script. This takes the argument is the name of the NIC attached to the VM you are trying to ssh to.
  - Given the following nodes these commands would be used to connect to the respective node

  ```
  $ oc get nodes
  NAME                                     STATUS     ROLES    AGE   VERSION
  aro-dev-abc123-master-0               Ready      master   47h   v1.19.0+2f3101c
  aro-dev-abc123-master-1               Ready      master   47h   v1.19.0+2f3101c
  aro-dev-abc123-master-2               Ready      master   47h   v1.19.0+2f3101c
  aro-dev-abc123-worker-eastus1-2s5rb   Ready      worker   47h   v1.19.0+2f3101c
  aro-dev-abc123-worker-eastus2-php82   Ready      worker   47h   v1.19.0+2f3101c
  aro-dev-abc123-worker-eastus3-cbqs2   Ready      worker   47h   v1.19.0+2f3101c


  CLUSTER=cluster hack/ssh-agent.sh master0 # master node aro-dev-abc123-master-0
  CLUSTER=cluster hack/ssh-agent.sh aro-dev-abc123-worker-eastus1-2s5rb # worker aro-dev-abc123-worker-eastus1-2s5rb
  CLUSTER=cluster hack/ssh-agent.sh eastus1 # worker aro-dev-abc123-worker-eastus1-2s5rb
  CLUSTER=cluster hack/ssh-agent.sh 2s5rb  # worker aro-dev-abc123-worker-eastus1-2s5rb
  CLUSTER=cluster hack/ssh-agent.sh bootstrap # the bootstrap node used to provision cluster
  ```

# Debugging AKS Cluster

- Connect to the VPN:

To access the cluster for oc / kubectl or SSH'ing into the cluster you need to connect to the VPN first.

> **NOTE:** If you have a password-based `sudo` command, you must first authenticate before running `sudo` in the background

```bash
sudo openvpn secrets/vpn-aks-$LOCATION.ovpn &
```

- Access the cluster via API (oc / kubectl):

  ```bash
  make aks.kubeconfig
  export KUBECONFIG=aks.kubeconfig

  $ oc get nodes
  NAME                                 STATUS   ROLES   AGE   VERSION
  aks-systempool-99744725-vmss000000   Ready    agent   9h    v1.23.5
  aks-systempool-99744725-vmss000001   Ready    agent   9h    v1.23.5
  aks-systempool-99744725-vmss000002   Ready    agent   9h    v1.23.5
  ```

- "SSH" into a cluster node:

  - Run the ssh-aks.sh script, specifying the cluster name and the node number of the VM you are trying to ssh to.

  ```bash
  hack/ssh-aks.sh aro-aks-cluster 0 # The first VM node in 'aro-aks-cluster'
  hack/ssh-aks.sh aro-aks-cluster 1 # The second VM node in 'aro-aks-cluster'
  hack/ssh-aks.sh aro-aks-cluster 2 # The third VM node in 'aro-aks-cluster'
  ```

- Access via Azure Portal

Due to the fact that the AKS cluster is private, you need to be connected to the VPN in order to view certain AKS cluster properties, because the UI interrogates k8s via the VPN.

### Metrics

To run fake metrics socket:

```bash
go run ./hack/monitor
```
### Run the RP and create a Hive cluster

**Steps to perform on Mac**
1. Mount your local MacOS filesystem into the podman machine:
```bash
podman machine init --now --cpus=4 --memory=4096 -v $HOME:$HOME
```
2. Use the openvpn config file (which is now mounted inside the podman machine) to start the VPN connection:
```bash
podman machine ssh
sudo rpm-ostree install openvpn
sudo systemctl reboot
podman machine ssh
sudo openvpn --config /Users/<user_name>/go/src/github.com/Azure/ARO-RP/secrets/vpn-aks-westeurope.ovpn --daemon --writepid vpnpid
ps aux | grep openvpn
```
### Instructions for Modifying Environment File
**Update the env File**
- Open the `env` file.
-  Update env file instructions: set `OPENSHIFT_VERSION`, update `INSTALLER_PULLSPEC` and `OCP_PULLSPEC`, mention quay.io for SHA256 hash.
-  Update INSTALLER_PULLSPEC with the appropriate name and tag, typically matching the OpenShift version, e.g., `release-4.13.`(for more detail see the `env.example`)
* Source the environment file before creating the cluster using the `setup_resources.sh` script(Added the updated env in the PR)
```bash
cd /hack
./setup_resources.sh
```
* Once the cluster create verify connectivity with the ARO cluster:
- Download the admin kubeconfig file
```bash
az aro get-admin-kubeconfig --name <cluster_name> --resource-group v4-westeurope --file ~/.kube/aro-admin-kubeconfig
```
- Set the KUBECONFIG environment variable
```bash
export KUBECONFIG=~/.kube/aro-admin-kubeconfig
```
- Verify connectivity with the ARO cluster
```bash
kubectl get nodes
```
```bash
kubectl get nodes
NAME                                                  STATUS   ROLES                  AGE   VERSION
shpaitha-aro-cluster-4sp5c-master-0                   Ready    control-plane,master   39m   v1.25.11+1485cc9
shpaitha-aro-cluster-4sp5c-master-1                   Ready    control-plane,master   39m   v1.25.11+1485cc9
shpaitha-aro-cluster-4sp5c-master-2                   Ready    control-plane,master   39m   v1.25.11+1485cc9
shpaitha-aro-cluster-4sp5c-worker-westeurope1-j9c76   Ready    worker                 29m   v1.25.11+1485cc9
shpaitha-aro-cluster-4sp5c-worker-westeurope2-j9zrs   Ready    worker                 27m   v1.25.11+1485cc9
shpaitha-aro-cluster-4sp5c-worker-westeurope3-56tk7   Ready    worker                 28m   v1.25.11+1485cc9
```

## Troubleshooting

1. Trying to use `az aro` CLI in Production, fails with:
```
(NoRegisteredProviderFound) No registered resource provider found for location '$LOCATION' and API version '2024-08-12-preview'
```
- Check if`~/.azure/config` there is a block `extensions.dev_sources`. If yes, comment it.
- Check if env var `AZURE_EXTENSION_DEV_SOURCES` is set. If yes, unset it.

- Installation fails with authorization errors:
```bash
Message="authorization.RoleAssignmentsClient#Create: Failure responding to request: StatusCode=403 -- Original Error: autorest/azure: Service returned an error. Status=403 Code=\"AuthorizationFailed\" Message=\"The client '$SP_ID' with object id '$SP_ID' does not have authorization to perform action 'Microsoft.Authorization/roleAssignments/write' over scope '/subscriptions/$SRE_SUBSCRIPTION/resourceGroups/$myresourcegroup/providers/Microsoft.Authorization/roleAssignments/b5a083aa-f555-466e-a268-4352b3b8394d' or the scope is invalid. If access was recently granted, please refresh your credentials.\"" Target="encountered error"
exit status 1
```

To resolve, check if it has the `User Access Administrator` role assigned.
```
az role assignment list --assignee $SP_ID --output json --query '[].{principalId:principalId, roleDefinitionName:roleDefinitionName, scope:scope}'
```
