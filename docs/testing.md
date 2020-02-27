# Testing

## Unit tests

  ```
  make test
  ```

## E2e tests

Running E2e can be run via CI by running `/azp run e2e` command in your PR
or locally.

Running it locally can be acheived with the following steps :
- Deploying or use an existing cosmos database
- Run the RP
- Validate if the RP is running properly by hitting the `healthz` route
- Register a subscription where to run the E2e
- Deploy a cluster : RG / Vnet / Cluster
- Export the KUBECONFIG file
- Run the `make e2e` target
- Delete the cluster and dependencies such as DB, RB & Vnet.

These steps can be acheived using the following commands :

```bash
# source your environment file
. ./secrets/env

# Deploy a new DB
deploy_e2e_db

# build the rp binary
make aro

# source the E2e helper file
source ./hack/e2e/run-rp-and-e2e.sh

# run the RP as background process
run_rp

# validate if the RP is ready to receive requests
validate_rp_running

# Register the sub you are using to run E2e
register_sub

# Deploy cluster prereqs. RG & Vnet
deploy_e2e_deps

# Run E2E
run_e2e

# Stop the local RP
kill_rp

# Delete the DB
clean_e2e_db

# Delete cluster prereqs. RG & Vnet
clean_e2e
```

> We encouraging you to look at the [E2e helper file](../hack/e2e/run-rp-and-e2e.sh) to understand each of those functions.
