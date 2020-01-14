# Using `az aro`

This repo includes the development `az aro` extension.  If you have a
whitelisted subscription, it can be used against the pre-GA Azure Red Hat
OpenShift v4 service, or (by setting `RP_MODE=development`) it can be used
against a development RP running at https://localhost:8443/.


## Installing the extension

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
   git clone https://github.com/Azure/ARO-RP
   cd ARO-RP
   ```

   Note: you will be able to update the `az aro` extension in the future by
   simply running `git pull`.

1. Build the development `az aro` extension:

   `make az`

1. Add the ARO extension path to your `az` configuration:

   ```
   cat >>~/.azure/config <<EOF
   [extension]
   dev_sources = $PWD/python
   EOF
   ```

1. Verify the ARO extension is registered:

   ```
   az -v
   ...
   Extensions:
   aro                                0.1.0 (dev) /path/to/rp/python/az/aro
   ...
   Development extension sources:
       /path/to/rp/python
   ...
   ```


## Registering the resource provider

If using the pre-GA Azure Red Hat OpenShift v4 service with a whitelisted
subscription, ensure that the `Microsoft.RedHatOpenShift` resource provider is
registered:

```
az feature register --subscription <Your Subscription ID> --name preview --namespace Microsoft.RedHatOpenShift
```

You can verify the status of the registration by doing :
```
az feature show  --namespace Microsoft.RedHatOpenShift --name preview
```

## Prerequisites to create an Azure Red Hat OpenShift v4 cluster

You will need the following in order to create an Azure Red Hat OpenShift v4
cluster:

1. A vnet containing two empty subnets, each with no network security group
   attached.  Your cluster will be deployed into these subnets.

   ```
   LOCATION=eastus
   RESOURCEGROUP="v4-$LOCATION"
   CLUSTER=cluster

   az group create -g "$RESOURCEGROUP" -l $LOCATION
   az network vnet create \
     -g "$RESOURCEGROUP" \
     -n dev-vnet \
     --address-prefixes 10.0.0.0/9 \
     >/dev/null
   for subnet in "$CLUSTER-master" "$CLUSTER-worker"; do
     az network vnet subnet create \
       -g "$RESOURCEGROUP" \
       --vnet-name dev-vnet \
       -n "$subnet" \
       --address-prefixes 10.$((RANDOM & 127)).$((RANDOM & 255)).0/24 \
       --service-endpoints Microsoft.ContainerRegistry \
       >/dev/null
   done
   az network vnet subnet update \
     -g "$RESOURCEGROUP" \
     --vnet-name dev-vnet \
     -n "$CLUSTER-master" \
     --disable-private-link-service-network-policies true \
     >/dev/null
   ```

1. A cluster AAD application (client ID and secret) and service principal, or
   sufficient AAD permissions for `az aro create` to create these for you
   automatically.

1. The RP service principal and cluster service principal must each have the
   Contributor role on the cluster vnet.  If you have the "User Access
   Administrator" role on the vnet, `az aro create` will set up the role
   assignments for you automatically.


## Using the extension

1. Create a cluster:

   ```
   az aro create \
     -g "$RESOURCEGROUP" \
     -n "$CLUSTER" \
     --vnet dev-vnet \
     --master-subnet "$CLUSTER-master" \
     --worker-subnet "$CLUSTER-worker"
   ```

   Note: cluster creation takes about 35 minutes.

1. Access the cluster console:

   You can find the cluster console URL (of the form
   `https://console-openshift-console.apps.<random>.<location>.aroapp.io/`) in
   the Azure Red Hat OpenShift v4 cluster resource:

   ```
   az aro list -o table
   ```

   You can log into the cluster using the `kubeadmin` user.  The password for
   the `kubeadmin` user can be found as follows:

   ```
   az aro list-credentials -g "$RESOURCEGROUP" -n "$CLUSTER"
   ```

   Note: the cluster console certificate is not yet signed by a CA: expect a
   security warning in your browser.

1. Scale the number of cluster VMs:

   ```
   COUNT=4

   az aro update -g "$RESOURCEGROUP" -n "$CLUSTER" --worker-count "$COUNT"
   ```

1. Delete a cluster:

   ```
   az aro delete -g "$RESOURCEGROUP" -n "$CLUSTER"

   # (optionally)
   for subnet in "$CLUSTER-master" "$CLUSTER-worker"; do
     az network vnet subnet delete -g "$RESOURCEGROUP" --vnet-name dev-vnet -n "$subnet"
   done
   ```
