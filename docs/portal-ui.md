# Portal UI

## Developing

You will require Node.js and `npm`. These instructions were tested with the versions from the Fedora 34 repos.

1. Make your desired changes in `portal/src/` and commit them.

1. Run `make build-portal` from the main directory. This will install the dependencies and kick off the Webpack build, placing the results in `portal/dist/`.

1. Run `make generate`. This will regenerate the golang file containing the portal content to be served.

1. Commit the results of `build-portal` and `generate`.

## Pointing Portal At Fake APIServer

1. Create a file containing the following as `fakekubeconfig`:

```
kind: Config
apiVersion: v1
clusters:
- cluster:
    insecure-skip-tls-verify: true
    server: https://localhost:6443
  name: test:6443
contexts:
- context:
    cluster: test:6443
    namespace: default
    user: test
  name: test
current-context: test
users:
- name: test
  user:
    token: sha256~aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa
```

1. `source secrets/env`

1. `go run hack/aead/aead.go --file fakekubeconfig`

1. Replace the CosmosDB Kubeconfig entries with what's returned (excepting the logs)

1. Replace `OpenShiftCluster.Properties.APIServerProfile.IP` value in CosmosDB with "127.0.0.1"

## Running the fake API Server

1. `go run hack/fakecluster/fakecluster.go`

## Adding Fake Data

1. Use `oc get --raw="/path/to/api` to get the raw output, and put it in `pkg/portal/cluster/testdata/<api.json>`

1. Add a new route to `hack/fakecluster/fakecluster.go`

1. Add new fetcher tests in `pkg/portal/cluster`, too!
