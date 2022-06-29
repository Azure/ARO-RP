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
     --template-file deploy/databases-development.json \
     --parameters \
       "databaseAccountName=$DATABASE_ACCOUNT_NAME" \
       "databaseName=$DATABASE_NAME" \
     >/dev/null
   ```

## Preparation to Create Cluster:

1. Update the Address Space of "rp-vnet" to allow for creation of a new VPN. You should be able to do this at: `https://ms.portal.azure.com/#@microsoft.onmicrosoft.com/resource/subscriptions/<subscription-id>/resourceGroups/<aro-shared-rp-rg>/providers/Microsoft.Network/virtualNetworks/rp-vnet/addressSpace`. See the table below for what we did.

| Address Space | Address Range         | Address Count |
| ------------- | --------------------- | ------------- |
| 10.0.0.0/24   | 10.0.0.0 - 10.0.0.255 | 256 |
| 10.1.0.0/24   | 10.1.0.0 - 10.1.0.255 | 256 |

2. Create a new "Virtual Network Gateway (Gateway type: VPN)" in the Azure Portal manually. This needs to be configured to the "Virtual Network" named "rp-vnet" which will already existing in the shared RP's resource group. This new VPN will allow the local ARO-RP to connect to the to the existing "rp-vnet" to create a cluster. 

3. Configure the new `rp-vnet` VPN with the same public certificate used for the existing dev-vpn. This is done at: `https://ms.portal.azure.com/#@microsoft.onmicrosoft.com/resource/subscriptions/<subscription-id>/resourceGroups/<aro-shared-rp-rg>/providers/Microsoft.Network/virtualNetworkGateways/rp-vnet/pointtositeconfiguration`. 
You can simply copy the info from the dev-vpn at: `https://ms.portal.azure.com/#@microsoft.onmicrosoft.com/resource/subscriptions/<subscription-id>/resourceGroups/<aro-shared-rp-rg>/providers/Microsoft.Network/virtualNetworkGateways/dev-vpn/pointtositeconfiguration`.

4. Connect to the "rp-vnet" VPN created above. You can use openvpn or the azure vpn client, both have worked fine in our testing. 
5. Go to Point-to-Site Configuration for "rp-vnet" (`https://ms.portal.azure.com/#@microsoft.onmicrosoft.com/resource/subscriptions/<subscription-id>/resourceGroups/<aro-shared-rp-rg>/providers/Microsoft.Network/virtualNetworkGateways/rp-vnet/pointtositeconfiguration`) and download the VPN Client Certificate to your local environment. You can extract the zip file anywhere you would like, but we put it under the "secrets" folder because that is where the ARO-RP secrets reside.
6. If using openvpn:
    1. Copy the last two certificates ("P2S client certificate" and "P2S client certificate private key") from ./secrets/vpn-eastus.ovpn file to ./secrets/vpn-rp-eastus.ovpn. You can overwrite the placeholders for those certificates at the bottom in the ./secrets/vpn-rp-eastus.ovpn file.

    2. Execute openvpn secrets/vpn-rp-eastus.ovpn. You may need sudo depending on your environment.
  > __NOTE:__ the azure vpn client for windows appears to require extra efforts; this is only for MacOS atm
7. If using Azure VPN Client
    1. Click the 'import' button in the vpn list, you will be prompted with an "open file dialog".
    2. Select the file: ./secrets/rp-vnet/AzureVPN/azurevpnconfig.xml. The data will be filled into the import screen with the exception of "Client Certificate Public Key Data" and "Private Key".
    3. Copy the "P2S client certificate" into the "Client Certificate Public Key Data" field and "P2S client certificate private key" into the "Private Key" field.
    4. Click "Save" and you should see your newly created VPN connection in the VPN list on the left.
    5. Click the new VPN connection and click "Connect".
8. Use nmap to execute the following command: nmap -p 443 -sT 10.x.x.x -Pn. You can get this IP at: `https://ms.portal.azure.com/#blade/Microsoft_Azure_Compute/VirtualMachineInstancesMenuBlade/Networking/instanceId/subscriptions/<subscription-id>/resourceGroups/<aro-shared-rp-rg>/providers/Microsoft.Compute/virtualMachineScaleSets/dev-proxy-vmss/virtualMachines/0`. Look for "NIC Private IP", ours during setup became 10.0.0.4. This is the internal ip of the Proxy VM.
9. Confirm the nmap output looks like this: (if it does not then your VPN is not connected correctly; kill anything using port 443 and connect again)
```bash
Starting Nmap 7.92 ( https://nmap.org ) at 2022-03-29 18:29 EDT
Nmap scan report for 10.0.0.4
Host is up (0.015s latency).

PORT STATE SERVICE
443/tcp open https

Nmap done: 1 IP address (1 host up) scanned in 0.25 seconds
```
10. Update the PROXY_HOSTNAME environment variable in ./secrets/env to point the IP you located above of for the Proxy VM.
11. Now that your VPN is connected correctly and you've updated PROXY_HOSTNAME you need to source your env file for that update.
```bash
. ./secrets/env
```
12. Execute the local ARO-RP
```bash
make runlocal-rp
```

## Steps to Create Cluster:

  1. Open another terminal (make sure you source your ./secrets/env file in this terminal as well)
  1. Execute this command to create a cluster
  ```bash
  CLUSTER=<aro-cluster-name> go run ./hack/cluster create
  ```

  This will take a while but eventually if the cluster is created you should see the following in your terminal indicating the cluster creation was successful:
  ```bash
  INFO[2022-04-01T10:02:41-05:00]pkg/util/cluster/cluster.go:318 cluster.(*Cluster).Create() creating cluster complete
  ```

## Steps to connect to the Cluster and confirm it is up via kubectl or oc:

1. At your terminal execute to create the admin.kubeconfig locally. This will allow you to connect to the cluster via kubectl or oc
   ```bash
   CLUSTER=<aro-cluster-name> make admin.kubeconfig
   ```

2. Disconnect from "rp-vnet" vpn and connect to "dev-vnet" vpn. The steps are identical to connecting to "rp-vnet" in the #preparation-to-create-cluster section. You will just need to download the dev-vpn client certificate locally, create the VPN connection using your VPN client of choice, and use nmap to get the IP of the internal load balancer from the <aro-cluster-name-rg>. You can find this address at: `https://ms.portal.azure.com/#@microsoft.onmicrosoft.com/resource/subscriptions/<subscriptionid>/resourceGroups/<aro-cluster-rp-rg>/providers/Microsoft.Network/loadBalancers/<aro-cluster-name>-<random string for your lb>-internal/frontendIpPool` (internal-lb-ip-v4). In my case the IP was 10.62.174.
```bash
nmap -p 6443 -sT 10.62.174.4 -Pn
Starting Nmap 7.92 ( https://nmap.org ) at 2022-04-01 10:36 CDT
Nmap scan report for 10.62.174.4
Host is up (0.070s latency).

PORT STATE SERVICE
6443/tcp open sun-sr-https

Nmap done: 1 IP address (1 host up) scanned in 0.14 seconds
```
3. Update admin.kubeconfig cluster.server parameter to use this IP as well. It should look like this:
```bash
server: https://<ip>:6443
```
4. Updated your kubeconfig env var to point to the admin.kubeconfig
```bash
export KUBECONFIG=$(pwd)/admin.kubeconfig
```
5. Execute a kubectl (or oc) command to see if you can list any K8s objects
```bash
kubectl get nodes --insecure-skip-tls-verify
```
6. You should see something like this. If so, your cluster is up!
```bash
NAME                                        STATUS   ROLES    AGE    VERSION
cdp-cfs-eleven-bljdk-master-0               Ready    master   3h7m   v1.22.3+4dd1b5a
cdp-cfs-eleven-bljdk-master-1               Ready    master   3h6m   v1.22.3+4dd1b5a
cdp-cfs-eleven-bljdk-master-2               Ready    master   3h6m   v1.22.3+4dd1b5a
cdp-cfs-eleven-bljdk-worker-eastus1-2r9b4   Ready    worker   177m   v1.22.3+4dd1b5a
cdp-cfs-eleven-bljdk-worker-eastus2-jgrj9   Ready    worker   177m   v1.22.3+4dd1b5a
cdp-cfs-eleven-bljdk-worker-eastus3-fd646   Ready    worker   177m   v1.22.3+4dd1b5a
```

## Available RP endpoints not exposed via `az aro`
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

* Get Cluster details of a dev cluster
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

* Get container logs from an OpenShift pod in a cluster
  ```bash
  NAMESPACE=<namespace-name>
  POD=<pod-name>
  CONTAINER=<container-name>
  curl -X GET -k "https://localhost:8443/admin/subscriptions/$AZURE_SUBSCRIPTION_ID/resourceGroups/$RESOURCEGROUP/providers/Microsoft.RedHatOpenShift/openShiftClusters/$CLUSTER/kubernetespodlogs?podname=$POD&namespace=$NAMESPACE&container=$CONTAINER"
  ```

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
