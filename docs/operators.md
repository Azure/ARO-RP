# Azure Red Hat OpenShift Operator

## Responsibilities

### Decentralizing service monitoring

This has the advantage of moving a proportion of monitoring effort to the edge,
giving headroom (and the corresponding potential disadvantage of increased
management complexity).  Doing this helps avoid bloat and complexity risks in
central monitoring as well as enabling additional and more complex monitoring
use cases.  Note that not all monitoring can be decentralised.

In all cases below the status.Conditions will be set.

* periodically check for outbound internet connectivity from both the master and
  worker nodes.
* periodically validate the cluster Service Principal permissions.
* [TODO] Enumerate daemonset statuses, pod statuses, etc.  We currently log
  diagnostic information associated with these checks in service logs; moving
  the checks to the edge will make these cluster logs, which is preferable.

### Automatic service remediation

There will be use cases where we may want to remediate end user decisions
automatically. Carrying out remediation locally is advantageous because it is
likely to be simpler, more reliable, and with a shorter time to remediate.

Remediations in place:
* periodically reset NSGs in the master and worker subnets to the defaults (controlled by the reconcileNSGs feature flag)

### End user warnings

* [TODO] see https://docs.openshift.com/container-platform/4.4/web_console/customizing-the-web-console.html#creating-custom-notification-banners_customizing-web-console

### Decentralizing ARO customization management

A cluster agent provides a centralized location to handle this use case.  Many
post-install configurations should probably move here.

* monitor and repair mdsd as needed
* set the alertmanager webhook

### Controllers and Deployment

The full list of operator controllers with descriptions can be
found in the README at the root of the repository.

The static pod resources can be found at `pkg/operator/deploy/staticresources`. The
deploy operation kicks off two deployments in the `openshift-azure-operator` namespace, one for
master and one for worker. The `aro-operator-master` deployment runs all controllers,
while the `aro-operator-worker` deployment runs only the internet checker in the worker subnet.

## Developer documentation

### How to Run a pre built operator image

Add the following to your "env" before running the rp
```sh
export ARO_IMAGE=arointsvc.azurecr.io/aro:latest
```

### How to Run the operator locally (out of cluster)

Make sure KUBECONFIG is set:
```sh
make admin.kubeconfig
export KUBECONFIG=$(pwd)/admin.kubeconfig
```

Alternatively you can set the KUBECONFIG by logging into your test cluster using `oc`:

```sh
oc login ${cluster-api-url} -u kubeadmin -p ${kubeadmin-password}
```

If you are using a private cluster, you need to connect to the respective VPN of your region. For example for eastus:
```sh
sudo openvpn --config secrets/vpn-eastus.ovpn
```
Scale the operator:
```sh
oc scale -n openshift-azure-operator deployment/aro-operator-master --replicas=0
```

Build the operator binary and run it locally (as if it was running a master node)
```sh
make generate
go run ./cmd/aro operator master
```

### How to create & publish ARO Operator image to ACR/Quay

1. Login to AZ
  ```bash
  az login
  ```

2. Install Docker according to the steps outlined in [Prepare Your Dev Environment](../../docs/prepare-your-dev-environment.md)

3. Publish Image to ACR

   * Pre-requisite:
     ```
     ACR created in Azure Portal with Name ${AZURE_PREFIX}aro
     2GB+ of Free RAM
     ```

    * Setup environment variables
      ```bash
      export DST_ACR_NAME=${AZURE_PREFIX}aro
      export DST_AUTH=$(echo -n '00000000-0000-0000-0000-000000000000:'$(az acr login -n ${DST_ACR_NAME} --expose-token | jq -r .accessToken) | base64 -w0)
      ```

    * Login to the Azure Container Registry
      ```bash
      docker login -u 00000000-0000-0000-0000-000000000000 -p "$(echo $DST_AUTH | base64 -d | cut -d':' -f2)" "${DST_ACR_NAME}.azurecr.io"
      ```

4. Publish Image to Quay

    * Pre-requisite:
      ```
      Quay account with repository created
      2GB+ of Free RAM
      ```

    * Setup mirroring environment variables
      ```bash
      export DST_QUAY=<quay-user-name>/<repository-name>
      export ARO_IMAGE=quay.io/${DST_QUAY}
      ```

    * Login to the Quay Registry
      ```bash
      docker login quay.io/${DST_QUAY}
      ```

5. Build and Push ARO Operator Image
  ```bash
  make publish-image-aro-multistage
  ```

### How to run a custom operator image

Add the following to your "env" before running the rp
```sh
export ARO_IMAGE=quay.io/asalkeld/aos-init:latest #(change to yours)
```

```sh
make publish-image-aro-multistage

#Then run an update
curl -X PATCH -k "https://localhost:8443/subscriptions/$AZURE_SUBSCRIPTION_ID/resourceGroups/$RESOURCEGROUP/providers/Microsoft.RedHatOpenShift/openShiftClusters/$CLUSTER?api-version=admin" --header "Content-Type: application/json" -d "{}"

#check on the deployment
oc -n openshift-azure-operator get all
oc -n openshift-azure-operator get clusters.aro.openshift.io/cluster -o yaml
oc -n openshift-azure-operator logs deployment.apps/aro-operator-master
oc -n openshift-config get secrets/pull-secret -o template='{{index .data ".dockerconfigjson"}}' | base64 -d
```

### How to run operator e2e tests

```sh
go test ./test/e2e -tags e2e -test.v --ginkgo.v --ginkgo.focus="ARO Operator"
```
