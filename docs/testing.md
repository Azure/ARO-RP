# Testing

## Unit tests

To run RP unit tests:

```bash
make test-go
```

To Run Go tests with coverage:

```bash
# first navigate to your directory with the code you'd like to see coverage on
t="/tmp/go-cover.$$.tmp" 
go test -coverprofile=$t $@ && go tool cover -html=$t && unlink $t
```

To run python client and `az aro` CLI tests:

```bash
make test-python
```

To run Go linting tasks (requires [golanglint-ci](https://golangci-lint.run/usage/install/) to be installed):

```bash
make lint-go
```

For faster feedback, you may want to set up [golanglint-ci's editor integration](https://golangci-lint.run/usage/integrations/).

## E2e tests

E2e tests can be run in CI with the `/azp run e2e` command in your GitHub PR.

E2e tests can also be run locally as follows:
- Deploy or use an existing cosmos database
- Run the RP
- Validate the RP is running properly by hitting the `/healthz` route
- Register a subscription where to run the e2e
- Run the `make test-e2e` target
- Delete the cosmos database, if applicable

You can also modify the flags passed to the e2e.test run by setting the E2E_FLAGS environment variable before running `make test-e2e`.

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

# Run e2e
make test-e2e

# Stop the local RP
kill_rp

# Delete the DB
clean_e2e_db
```
