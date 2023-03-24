# Deploy development RP

## Prerequisites

1. Your development environment is prepared according to the steps outlined in [Prepare Your Dev Environment](./prepare-your-dev-environment.md)

## Installing the extension

1. Build the development `az aro` extension:

   `make az`

1. Verify that the ARO extension path is in your `az` configuration:

   ```bash
   grep -q 'dev_sources' ~/.azure/config || cat >>~/.azure/config <<EOF
   [extension]
   dev_sources = $PWD/python
   EOF
   ```

1. Verify the ARO extension is registered:

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
   future by simply running `git pull`.


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

1. Copy, edit (if necessary) and source your environment file.  The required
   environment variable configuration is documented immediately below:

   ```bash
   cp env.example env
   vi env
   . ./env
   ```

   * `LOCATION`: Location of the shared RP development environment (default:
     `eastus`).
   * `RP_MODE`: Set to `development` to use a development RP running at
     https://localhost:8443/.

1. Create your own RP database:

   ```bash
   az deployment group create \
     -g "$RESOURCEGROUP" \
     -n "databases-development-$USER" \
     --template-file pkg/deploy/assets/databases-development.json \
     --parameters \
       "databaseAccountName=$DATABASE_ACCOUNT_NAME" \
       "databaseName=$DATABASE_NAME" \
     >/dev/null
   ```


## Run the RP and create a cluster

1. Source your environment file.

   ```bash
   . ./env
   ```

1. Run the RP

   ```bash
   make runlocal-rp
   ```

1. To create a cluster, EITHER follow the instructions in [Create, access, and
   manage an Azure Red Hat OpenShift 4.3 Cluster][1].  Note that as long as the
   `RP_MODE` environment variable is set to `development`, the `az aro` client
   will connect to your local RP.

   OR use the create utility:

   ```bash
   CLUSTER=<cluster-name> go run ./hack/cluster create
   ```

   Later the cluster can be deleted as follows:

   ```bash
   CLUSTER=<cluster-name> go run ./hack/cluster delete
   ```

   By default, a public cluster will be created. In order to create a private cluster, set the `PRIVATE_CLUSTER` environment variable to `true` prior to creation. Internet access from the cluster can also be restricted by setting the `NO_INTERNET` environment variable to `true`.

   [1]: https://docs.microsoft.com/en-us/azure/openshift/tutorial-create-cluster

1. The following additional RP endpoints are available but not exposed via `az
   aro`:

   * Delete a subscription, cascading deletion to all its clusters:

     ```bash
     curl -k -X PUT \
       -H 'Content-Type: application/json' \
       -d '{"state": "Deleted", "properties": {"tenantId": "'"$AZURE_TENANT_ID"'"}}' \
       "https://localhost:8443/subscriptions/$AZURE_SUBSCRIPTION_ID?api-version=2.0"
     ```

   * List operations:

     ```bash
     curl -k \
       "https://localhost:8443/providers/Microsoft.RedHatOpenShift/operations?api-version=2020-04-30"
     ```

   * View RP logs in a friendly format:

     ```bash
     journalctl _COMM=aro -o json --since "15 min ago" -f | jq -r 'select (.COMPONENT != null and (.COMPONENT | contains("access"))|not) | .MESSAGE'
     ```

## Automatically run local RP
If you are already familiar with running the ARO RP locally, you can speed up the process executing the [local_dev_env.sh](../hack/devtools/local_dev_env.sh) script.

## Connect ARO-RP with a Hive development cluster
The env variables names defined in pkg/util/liveconfig/manager.go control the communication of the ARO-RP with Hive.
- If you want to use ARO-RP + Hive, set *HIVE_KUBE_CONFIG_PATH* to the path of the kubeconfig of the AKS Dev cluster. [Info](https://github.com/Azure/ARO-RP/blob/master/docs/deploy-development-rp.md#debugging-openshift-cluster) about creating that kubeconfig (Step *Get an admin kubeconfig:*).
- If you want to create clusters using the local ARO-RP + Hive instead of doing the standard cluster creation process (which doesn't use Hive), set *ARO_INSTALL_VIA_HIVE* to *true*.
- If you want to enable the Hive adoption feature (which is performed during adminUpdate()), set *ARO_ADOPT_BY_HIVE* to *true*.

After setting the above environment variables (using *export* direclty in the terminal or including them in the *env* file), connect to the [VPN](https://github.com/Azure/ARO-RP/blob/master/docs/deploy-development-rp.md#debugging-aks-cluster) (*Connect to the VPN* section).

Then proceed to [run](https://github.com/Azure/ARO-RP/blob/master/docs/deploy-development-rp.md#run-the-rp-and-create-a-cluster) the ARO-RP as usual. 

After that, when you [create](https://github.com/Azure/ARO-RP/blob/master/docs/deploy-development-rp.md#run-the-rp-and-create-a-cluster) a cluster, you will be using Hive behind the scenes. You can check the created Hive objects following [Debugging OpenShift Cluster](https://github.com/Azure/ARO-RP/blob/master/docs/deploy-development-rp.md#debugging-openshift-cluster) and using the *oc* command.

## Make Admin-Action API call(s) to a running local-rp

  ```bash
  export CLUSTER=<cluster-name>
  export AZURE_SUBSCRIPTION_ID=<subscription-id>
  export RESOURCEGROUP=<resource-group-name>
    [OR]
  . ./env
  ```

* Perform AdminUpdate on a dev cluster
  ```bash
  curl -X PATCH -k "https://localhost:8443/subscriptions/$AZURE_SUBSCRIPTION_ID/resourceGroups/$RESOURCEGROUP/providers/Microsoft.RedHatOpenShift/openShiftClusters/$CLUSTER?api-version=admin" --header "Content-Type: application/json" -d "{}"
  ```

* Get Cluster details of a dev cluster
  ```bash
  curl -X GET -k "https://localhost:8443/subscriptions/$AZURE_SUBSCRIPTION_ID/resourceGroups/$RESOURCEGROUP/providers/Microsoft.RedHatOpenShift/openShiftClusters/$CLUSTER?api-version=admin" --header "Content-Type: application/json" -d "{}"
  ```

* Get SerialConsole logs of a VM of dev cluster
  ```bash
  VMNAME="aro-cluster-qplnw-master-0"
  curl -X GET -k "https://localhost:8443/admin/subscriptions/$AZURE_SUBSCRIPTION_ID/resourceGroups/$RESOURCEGROUP/providers/Microsoft.RedHatOpenShift/openShiftClusters/$CLUSTER/serialconsole?vmName=$VMNAME" --header "Content-Type: application/json" -d "{}"
  ```

* Redeploy a VM in a dev cluster
  ```bash
  VMNAME="aro-cluster-qplnw-master-0"
  curl -X POST -k "https://localhost:8443/admin/subscriptions/$AZURE_SUBSCRIPTION_ID/resourceGroups/$RESOURCEGROUP/providers/Microsoft.RedHatOpenShift/openShiftClusters/$CLUSTER/redeployvm?vmName=$VMNAME" --header "Content-Type: application/json" -d "{}"
  ```

* Stop a VM in a dev cluster
  ```bash
  VMNAME="aro-cluster-qplnw-master-0"
  curl -X POST -k "https://localhost:8443/admin/subscriptions/$AZURE_SUBSCRIPTION_ID/resourceGroups/$RESOURCEGROUP/providers/Microsoft.RedHatOpenShift/openShiftClusters/$CLUSTER/stopvm?vmName=$VMNAME" --header "Content-Type: application/json" -d "{}"
  ```

* Stop and deallocate a VM in a dev cluster
  ```bash
  VMNAME="aro-cluster-qplnw-master-0"
  curl -X POST -k "https://localhost:8443/admin/subscriptions/$AZURE_SUBSCRIPTION_ID/resourceGroups/$RESOURCEGROUP/providers/Microsoft.RedHatOpenShift/openShiftClusters/$CLUSTER/stopvm?vmName=$VMNAME" --header "Content-Type: application/json" -d "{}"
  ```

* Start a VM in a dev cluster
  ```bash
  VMNAME="aro-cluster-qplnw-master-0"
  curl -X POST -k "https://localhost:8443/admin/subscriptions/$AZURE_SUBSCRIPTION_ID/resourceGroups/$RESOURCEGROUP/providers/Microsoft.RedHatOpenShift/openShiftClusters/$CLUSTER/startvm?vmName=$VMNAME" --header "Content-Type: application/json" -d "{}"
  ```

* List VM Resize Options for a master node of dev cluster
  ```bash
  curl -X GET -k "https://localhost:8443/admin/subscriptions/$AZURE_SUBSCRIPTION_ID/resourceGroups/$RESOURCEGROUP/providers/Microsoft.RedHatOpenShift/openShiftClusters/$CLUSTER/skus" --header "Content-Type: application/json" -d "{}"
  ```

* Resize master node of a dev cluster
  ```bash
  VMNAME="aro-cluster-qplnw-master-0"
  VMSIZE="Standard_D16s_v3"
  curl -X POST -k "https://localhost:8443/admin/subscriptions/$AZURE_SUBSCRIPTION_ID/resourceGroups/$RESOURCEGROUP/providers/Microsoft.RedHatOpenShift/openShiftClusters/$CLUSTER/resize?vmName=$VMNAME&vmSize=$VMSIZE" --header "Content-Type: application/json" -d "{}"
  ```

* List Clusters of a local-rp
  ```bash
  curl -X GET -k "https://localhost:8443/admin/providers/microsoft.redhatopenshift/openshiftclusters"
  ```

* List cluster Azure Resources of a dev cluster
  ```bash
  curl -X GET -k "https://localhost:8443/admin/subscriptions/$AZURE_SUBSCRIPTION_ID/resourceGroups/$RESOURCEGROUP/providers/Microsoft.RedHatOpenShift/openShiftClusters/$CLUSTER/resources"
  ```

* Perform Cluster Upgrade on a dev cluster
  ```bash
  curl -X POST -k "https://localhost:8443/admin/subscriptions/$AZURE_SUBSCRIPTION_ID/resourceGroups/$RESOURCEGROUP/providers/Microsoft.RedHatOpenShift/openShiftClusters/$CLUSTER/upgrade"
  ```

* Get container logs from an OpenShift pod in a cluster
  ```bash
  NAMESPACE=<namespace-name>
  POD=<pod-name>
  CONTAINER=<container-name>
  curl -X GET -k "https://localhost:8443/admin/subscriptions/$AZURE_SUBSCRIPTION_ID/resourceGroups/$RESOURCEGROUP/providers/Microsoft.RedHatOpenShift/openShiftClusters/$CLUSTER/kubernetespodlogs?podname=$POD&namespace=$NAMESPACE&container=$CONTAINER"
  ```

## OpenShift Version

* We have a cosmos container which contains supported installable OCP versions, more information on the definition in `pkg/api/openshiftversion.go`.

* Admin - List OpenShift installation versions
  ```bash
  curl -X GET -k "https://localhost:8443/admin/versions"
  ```

* Admin - Put a new OpenShift installation version
  ```bash
  curl -X PUT -k "https://localhost:8443/admin/versions" --header "Content-Type: application/json" -d '{ "properties": { "version": "4.10.0", "enabled": true, "openShiftPullspec": "test.com/a:b", "installerPullspec": "test.com/a:b" }}'
  ```

* List the enabled OpenShift installation versions within a region
  ```bash
  curl -X GET -k "https://localhost:8443/subscriptions/$AZURE_SUBSCRIPTION_ID/providers/Microsoft.RedHatOpenShift/locations/$LOCATION/openshiftversions?api-version=2022-09-04"
  ```

## OpenShift Cluster Manager (OCM) Configuration API Actions

* Create a new OCM configuration
  * You can find example payloads in the projects `./hack/ocm` folder.

  ```bash
  curl -X PUT -k "https://localhost:8443/subscriptions/$AZURE_SUBSCRIPTION_ID/resourceGroups/$RESOURCEGROUP/providers/Microsoft.RedHatOpenShift/openShiftClusters/$CLUSTER/syncsets/mySyncSet?api-version=2022-09-04" --header "Content-Type: application/json" -d @./hack/ocm/syncset.b64


## Debugging OpenShift Cluster

* SSH to the bootstrap node:
> __NOTE:__ If you have a password-based `sudo` command, you must first authenticate before running `sudo` in the background
  ```bash
  sudo openvpn secrets/vpn-$LOCATION.ovpn &
  CLUSTER=cluster hack/ssh-agent.sh bootstrap
  ```

* Get an admin kubeconfig:

  ```bash
  CLUSTER=cluster make admin.kubeconfig
  export KUBECONFIG=admin.kubeconfig
  ```

* "SSH" to a cluster node:

  * Get the admin kubeconfig and `export KUBECONFIG` as detailed above.
  * Run the ssh-agent.sh script. This takes the argument is the name of the NIC attached to the VM you are trying to ssh to.
   * Given the following nodes these commands would be used to connect to the respective node

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

* Connect to the VPN:

To access the cluster for oc / kubectl or SSH'ing into the cluster you need to connect to the VPN first.
> __NOTE:__ If you have a password-based `sudo` command, you must first authenticate before running `sudo` in the background
  ```bash
  sudo openvpn secrets/vpn-aks-$LOCATION.ovpn &
  ```

* Access the cluster via API (oc / kubectl):

  ```bash
  make aks.kubeconfig
  export KUBECONFIG=aks.kubeconfig

  $ oc get nodes
  NAME                                 STATUS   ROLES   AGE   VERSION
  aks-systempool-99744725-vmss000000   Ready    agent   9h    v1.23.5
  aks-systempool-99744725-vmss000001   Ready    agent   9h    v1.23.5
  aks-systempool-99744725-vmss000002   Ready    agent   9h    v1.23.5
  ```

* "SSH" into a cluster node:

  * Run the ssh-aks.sh script, specifying the cluster name and the node number of the VM you are trying to ssh to.
  ```
  hack/ssk-aks.sh aro-aks-cluster 0 # The first VM node in 'aro-aks-cluster'
  hack/ssk-aks.sh aro-aks-cluster 1 # The second VM node in 'aro-aks-cluster'
  hack/ssk-aks.sh aro-aks-cluster 2 # The third VM node in 'aro-aks-cluster'
  ```

* Access via Azure Portal

Due to the fact that the AKS cluster is private, you need to be connected to the VPN in order to view certain AKS cluster properties, because the UI interrogates k8s via the VPN.

### Metrics

To run fake metrics socket:
```bash
go run ./hack/monitor
```
