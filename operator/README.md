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

```sh
WATCH_NAMESPACE="" PULL_SECRET_PATH=operator/pull-secrets go run ./operator/cmd/manager
```

if you change the api in operator/pkg/apis make sure to do the following:

```sh
cd operator
operator-sdk generate crds
operator-sdk generate k8s

```

also see: https://sdk.operatorframework.io/docs/golang/quickstart/
