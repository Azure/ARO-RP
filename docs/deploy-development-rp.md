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

1. Set SECRET_SA_ACCOUNT_NAME to the name of the storage account containing your
   shared development environment secrets and save them in `secrets`:

   ```bash
   SECRET_SA_ACCOUNT_NAME=rharosecrets make secrets
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
     --template-file deploy/databases-development.json \
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
   CLUSTER=cluster go run ./hack/cluster create
   ```

   Later the cluster can be deleted as follows:

   ```bash
   CLUSTER=cluster go run ./hack/cluster delete
   ```

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

* Get Cluster detials of a dev cluster
  ```bash
  curl -X GET -k "https://localhost:8443/subscriptions/$AZURE_SUBSCRIPTION_ID/resourceGroups/$RESOURCEGROUP/providers/Microsoft.RedHatOpenShift/openShiftClusters/$CLUSTER?api-version=admin" --header "Content-Type: application/json" -d "{}"
  ```

* Get SerialConsole logs of a VM of dev cluster
  ```bash
  VMNAME="aro-cluster-qplnw-master-0"
  curl -X GET -k "https://localhost:8443/admin/subscriptions/$AZURE_SUBSCRIPTION_ID/resourceGroups/$RESOURCEGROUP/providers/Microsoft.RedHatOpenShift/openShiftClusters/$CLUSTER/serialconsole?vmName=$VMNAME" --header "Content-Type: application/json" -d "{}"
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

## Debugging

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

### Metrics

To run fake metrics socket:
```bash
go run ./hack/monitor
```
