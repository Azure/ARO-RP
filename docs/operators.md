# About Azure Red Hat OpenShift Operator
## Overview
* The full list of operator controllers with descriptions can be found in the README at the root of the repository.
* The checks done by the operator can be found at: `ARO-RP/pkg/operator/controllers/checkers`.

* The static pod resources can be found at `pkg/operator/deploy/staticresources`. 
* The deploy operation kicks off two deployments in the `openshift-azure-operator` namespace: `aro-operator-master` and `aro-operator-worker`.
  * The `aro-operator-master` deployment runs all controllers,
  * The `aro-operator-worker` deployment runs only the internet checker in the worker subnet.

## Responsibilities
### Decentralizing service monitoring

This has the advantage of moving a proportion of monitoring effort to the edge,
giving headroom (and the corresponding potential disadvantage of increased
management complexity).  Doing this helps avoid bloat and complexity risks in
central monitoring as well as enabling additional and more complex monitoring
use cases.  Note that not all monitoring can be decentralised.

In all cases below the status.Conditions will be set.

### Automatic service remediation

There will be use cases where we may want to remediate end user decisions
automatically. Carrying out remediation locally is advantageous because it is
likely to be simpler, more reliable, and with a shorter time to remediate.

Remediations in place:
* periodically reset NSGs in the master and worker subnets to the defaults (controlled by the reconcileNSGs feature flag)
* recreate broken/missing pull secrets

### Decentralizing ARO customization management

A cluster agent provides a centralized location to handle this use case.  Many
post-install configurations should probably move here.

* monitor and repair mdsd as needed
* set the alertmanager webhook

### Remediation metrics

Metrics are emitted for each remediation with labels `success` and `error` to represent the outcome.
Currently, only `pullSecret` remediation metrics are being emitted.

# ARO Operator | Developer Documentation
## Building your own custom ARO Operator image

1. Build the image with `make image-aro-multistage`
1. Tag and push the image to your own repo
```
podman tag arointsvc.azurecr.io/aro:latest quay.io/<user>/aro:latest
podman push quay.io/<user>/aro:latest 
```

## Testing
There are 4 possible ways to test the operator.
* [In-cluster deployment](#in-cluster-deployment)
* [Using the RP API](#using-the-rp-api)
* [How to run the operator locally (out of cluster)](#how-to-run-the-operator-locally-out-of-cluster)
* [Mimicking AdminUpdate when updating the ARO Operator](#mimicking-adminupdate-when-updating-the-aro-operator)
* [How to create & publish ARO Operator image to ACR/Quay](#how-to-create--publish-aro-operator-image-to-acrquay)
* [How to run operator e2e tests](#how-to-run-operator-e2e-tests)

### In-cluster deployment
Using either a local dev cluster or a prod cluster:

1. Update the `aro-operator-master` deployment (This will run trigger a new rollout).
```
oc patch deployment aro-operator-master -n openshift-azure-operator --type='strategic' -p='{"spec":{"template":{"spec":{"containers":[{"name":"aro-operator","image":"quay.io/<user>/aro:latest"}]}}}}'
```

### How to run the operator locally (out of cluster)

1. Set the kubeconfig.
```sh
make admin.kubeconfig
export KUBECONFIG=$(pwd)/admin.kubeconfig
```
* (**For Private clusters**) Connect to the respective VPN of your region. For example for eastus:
```sh
sudo openvpn --config secrets/vpn-eastus.ovpn
```
2. Scale the operator:
```sh
oc scale -n openshift-azure-operator deployment/aro-operator-master --replicas=0
```
3. Build the operator binary and run it locally (as if it was running a master node)
```sh
make generate
go run ./cmd/aro operator master
```
### Using the RP API
#### Pre-requisites
* Have a local dev RP running
* Have a local dev cluster 

#### Steps
1. Stop the RP and update the `env` file, the variable value `$ARO_IMAGE` with the custom built image:
```sh
export ARO_IMAGE=quay.io/<user>/aro:latest
```
2. Start the RP

- We can mimick the AdminUpdate when updating the ARO Operator
This is the way we would test the same PUCM workflow we would use in Prod to update the operator.
```
curl -X PATCH -k "https://localhost:8443/subscriptions/$AZURE_SUBSCRIPTION_ID/resourceGroups/$RESOURCEGROUP/providers/Microsoft.RedHatOpenShift/openShiftClusters/$CLUSTER?api-version=admin" --header "Content-Type: application/json" -d "{}"
```

### How to create & publish ARO Operator image to ACR/Quay
1. Login to AZ `az login`

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

### How to run operator e2e tests

```sh
go test ./test/e2e -tags e2e -test.v --ginkgo.v --ginkgo.focus="ARO Operator"
```
