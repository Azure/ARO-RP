# Azure Red Hat OpenShift Operator

## Responsibilities

### Decentralizing service monitoring

This has the advantage of moving a proportion of monitoring effort to the edge,
giving headroom (and the corresponding potential disadvantage of increased
management complexity).  Doing this helps avoid bloat and complexity risks in
central monitoring as well as enabling additional and more complex monitoring
use cases.  Note that not all monitoring can be decentralised.

In all cases below the status.Conditions will be set.

* periodically check for outbound internet connectivity from both the master and worker nodes.
* periodically validate the cluster Service Principal permissions.
* [TODO] Enumerate daemonset statuses, pod statuses, etc.  We currently log diagnostic information associated with these checks in service logs; moving the checks to the edge will make these cluster logs, which is preferable.

### Automatic service remediation

There will be use cases where we may want to remediate end user decisions automatically.
Carrying out remediation locally is advantageous because it is likely to be simpler,
more reliable, and with a shorter time to remediate.

* monitor and repair pull secret (acr part)

### End user warnings

* [TODO] see https://docs.openshift.com/container-platform/4.4/web_console/customizing-the-web-console.html#creating-custom-notification-banners_customizing-web-console

### Decentralizing ARO customization management

A cluster agent provides a centralized location to handle this use case.  Many
post-install configurations should probably move here.

* monitor and repair mdsd as needed
* set the alertmanager webhook

## Developer documentation

### How to Run the operator locally (out of cluster)

Make sure KUBECONFIG is set:
```sh
make admin.kubeconfig
export KUBECONFIG=$(pwd)/admin.kubeconfig
oc delete -n openshift-azure-operator deployment/aro-operator
make generate
go run ./cmd/aro operator
```

### How to run a custom operator image

In one terminal
```sh
export ARO_IMAGE=quay.io/asalkeld/aos-init:latest #(change to yours)
go run ./cmd/aro rp
```

In a second terminal
```sh
export ARO_IMAGE=quay.io/asalkeld/aos-init:latest #(change to yours)
make publish-image-aro

#Then run an update
curl -X PATCH -k "https://localhost:8443/subscriptions/$AZURE_SUBSCRIPTION_ID/resourceGroups/$RESOURCEGROUP/providers/Microsoft.RedHatOpenShift/openShiftClusters/$CLUSTER?api-version=admin" --header "Content-Type: application/json" -d "{}"

#check on the deployment
oc -n openshift-azure-operator get all
oc -n openshift-azure-operator get clusters.aro.openshift.io/cluster -o yaml
oc -n openshift-azure-operator logs deployment.apps/aro-operator
```
