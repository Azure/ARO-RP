# Testing

## Unit tests

  ```bash
  make test
  ```

## E2e tests

E2e tests can be run in CI with the `/azp run e2e` command in your GitHub PR.

E2e tests can also be run locally as follows:
- Deploy or use an existing cosmos database
- Run the RP
- Validate the RP is running properly by hitting the `/healthz` route
- Register a subscription where to run the e2e
- Deploy a cluster: RG / Vnet / Cluster
- Export the KUBECONFIG file
- Run the `make e2e` target
- Delete the cluster and dependencies such as DB, RB & Vnet.

These steps can be acheived using commands below.  Look at the [e2e helper
file](../hack/e2e/run-rp-and-e2e.sh) to understand each of the bash functions
below.


```bash
# source your environment file
. ./secrets/env

# source the e2e helper file
. ./hack/e2e/run-rp-and-e2e.sh

# Deploy a new DB
deploy_e2e_db

# build the rp binary
make aro

# run the RP as background process
run_rp

# validate if the RP is ready to receive requests
validate_rp_running

# Register the sub you are using to run e2e
register_sub

# Deploy cluster prereqs. RG & Vnet
deploy_e2e_deps

# Run e2e
run_e2e

# Stop the local RP
kill_rp

# Delete the DB
clean_e2e_db

# Delete cluster prereqs. RG & Vnet
clean_e2e
```
