# Azure Red Hat OpenShift Operator

## MVP

* monitor and repair pull secret (acr part)
* periodically check for internet connectivity and mark as supported/unsupported
* monitor and repair mdsd as needed

## Future responsibilities

### Decentralizing service monitoring

* periodically check for internet connectivity and mark as supported/unsupported

### Automatic service remediation

* monitor and repair pull secret (acr part)
* monitor and repair mdsd as needed

### End user warnings

### Decentralizing ARO customization management

* take over install customizations

## dev help

### run controller locally
```sh
oc delete -n openshift-azure-operator deployment/aro-operator
make generate
oc apply -f operator/deploy/staticresources/*.yaml
# [optional] to test pullsecrets
  export PULL_SECRET_PATH=operator/pull-secrets
  mkdir $PULL_SECRET_PATH
  #get the acr "username:pass" >> $PULL_SECRET_PATH/arosvc.azurecr.io
go run ./cmd/aro operator
```

### test a new controller image

```sh
export ARO_IMAGE=quay.io/asalkeld/aos-init:latest #(change to yours)
make publish-image-aro

go run ./cmd/aro rp &

#Then run an update
curl -X PATCH -k "https://localhost:8443/subscriptions/$AZURE_SUBSCRIPTION_ID/resourceGroups/$RESOURCEGROUP/providers/Microsoft.RedHatOpenShift/openShiftClusters/$CLUSTER?api-version=admin" --header "Content-Type: application/json" -d "{}"

#check on the deployment
oc -n openshift-azure-operator get all
oc -n openshift-azure-operator get clusters.aro.openshift.io/cluster -o yaml
oc -n openshift-azure-operator logs deployment.apps/aro-operator
```
