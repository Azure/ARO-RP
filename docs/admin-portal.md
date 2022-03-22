# Admin Portal

## Introduction

The admin portal is a SRE-facing front end used for performing various actions and retrieving information for clusters in a given region.

The admin portal runs inside the RP and calls against the RP for cluster information and serves that information using a MSFT inspired front end.

The front end is developed using react and typescript. The back end api is written in golang and makes direct calls to CosmoDB for cluster information.

The portal front end lives in the top level directory of the ARO-RP repo within the `portal` directory. The portal back end exists within the `pkg/portal`

The front end code is compiled into go code using the bindata golang module. This front end code is then served through the RP.


## Developing

You will require Node.js and `npm`. These instructions were tested with the versions from the Fedora 34 repos.

1. Make your desired changes in `portal/v2/src/` and commit them.

1. Run `make build-portal` from the main directory. This will install the dependencies and kick off the Webpack build, placing the results in `portal/v2/build/`.

1. Run `make generate`. This will regenerate the golang file containing the portal content to be served.

1. Commit the results of `build-portal` and `generate`.

## Running Admin Portal in development

### Running Portal Served from development RP

1. Complete Steps mentioned above to build and compile portal.

1. Make sure development environment variables are set and also set `export NO_NPM=1`

1. Run `make run-portal`

1. Go to localhost:8444 to view admin portal running

### Running Portal Served from the front end development server

1. Complete Steps mentioned above to build and compile portal.

1. Make sure development environment variables are set

1. Run `make run-portal`

1. In a seperate tab change directory to `portal/v2/` and run `npm run start` to run front end development server

1. Go to localhost:3000 to view admin portal running

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
