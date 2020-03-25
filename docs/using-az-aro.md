# Using `az aro`

This repo includes the development `az aro` extension.  If you have a
whitelisted subscription, it can be used against the pre-GA Azure Red Hat
OpenShift v4 service, or (by setting `RP_MODE=development`) it can be used
against a development RP running at https://localhost:8443/.


## Installing the extension

1. Install a supported version of [Python](https://www.python.org/downloads), if
   you don't have one installed already.  The `az` client supports Python 2.7
   and Python 3.5+.  A recent Python 3.x version is recommended.  You will also
   need setuptools installed, if you don't have it installed already.

1. Install the
   [`az`](https://docs.microsoft.com/en-us/cli/azure/install-azure-cli) client,
   if you haven't already.  You will need `az` version 2.0.72 or greater, as
   this version includes the `az network vnet subnet update
   --disable-private-link-service-network-policies` flag.

1. Log in to Azure:

   ```bash
   az login
   ```

1. Git clone this repository to your local machine:

   ```bash
   git clone https://github.com/Azure/ARO-RP
   cd ARO-RP
   ```

   Note: you will be able to update the `az aro` extension in the future by
   simply running `git pull`.

1. Build the development `az aro` extension:

   `make az`

   Note: you may see a message like the following; if so you can safely ignore
   it:

   ```
   byte-compiling build/bdist.linux-x86_64/egg/azext_aro/vendored_sdks/azure/mgmt/redhatopenshift/v2019_12_31_preview/models/_models_py3.py to _models_py3.pyc
     File "build/bdist.linux-x86_64/egg/azext_aro/vendored_sdks/azure/mgmt/redhatopenshift/v2019_12_31_preview/models/_models_py3.py", line 45
       def __init__(self, *, visibility=None, url: str=None, ip: str=None, **kwargs) -> None:
                        ^
    SyntaxError: invalid syntax
    ```

1. Add the ARO extension path to your `az` configuration:

   ```bash
   cat >>~/.azure/config <<EOF
   [extension]
   dev_sources = $PWD/python
   EOF
   ```

1. Verify the ARO extension is registered:

   ```bash
   az -v
   ...
   Extensions:
   aro                                0.3.0 (dev) /path/to/rp/python/az/aro
   ...
   Development extension sources:
       /path/to/rp/python
   ...
   ```


## Registering the resource provider

If using the pre-GA Azure Red Hat OpenShift v4 service, ensure that the
`Microsoft.RedHatOpenShift` resource provider is registered:

```bash
az provider register -n Microsoft.RedHatOpenShift --wait
```


## Prerequisites to create an Azure Red Hat OpenShift v4 cluster

You will need the following in order to create an Azure Red Hat OpenShift v4
cluster:

1. A vnet containing two empty subnets, each with no network security group
   attached.  Your cluster will be deployed into these subnets.

   ```bash
   LOCATION=eastus
   RESOURCEGROUP="v4-$LOCATION"
   export CLUSTER=cluster
   USER_PULL_SECRET=<https://cloud.redhat.com/ pull secret content>

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

   ```bash
   az aro create \
     -g "$RESOURCEGROUP" \
     -n "$CLUSTER" \
     --vnet dev-vnet \
     --master-subnet "$CLUSTER-master" \
     --worker-subnet "$CLUSTER-worker"
     --pull-secret "$USER_PULL_SECRET"
   ```

   Note: cluster creation takes about 35 minutes.

1. Access the cluster console:

   You can find the cluster console URL (of the form
   `https://console-openshift-console.apps.<random>.<location>.aroapp.io/`) in
   the Azure Red Hat OpenShift v4 cluster resource:

   ```bash
   az aro list -o table
   ```

   You can log into the cluster using the `kubeadmin` user.  The password for
   the `kubeadmin` user can be found as follows:

   ```bash
   az aro list-credentials -g "$RESOURCEGROUP" -n "$CLUSTER"
   ```

1. Delete a cluster:

   ```bash
   az aro delete -g "$RESOURCEGROUP" -n "$CLUSTER"

   # (optionally)
   for subnet in "$CLUSTER-master" "$CLUSTER-worker"; do
     az network vnet subnet delete -g "$RESOURCEGROUP" --vnet-name dev-vnet -n "$subnet"
   done
   ```
