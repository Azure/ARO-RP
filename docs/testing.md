# Testing

## Unit tests

To run RP unit tests:

```bash
make test-go
```

In case of MacOS, the go-diff module creates [issue](https://github.com/golangci/golangci-lint/issues/3087) making the test fail. Until a new release of the module with the [fix](https://github.com/sourcegraph/go-diff/pull/65) is available, an easy workaround to mitigate the issue is to install diffutils using `brew install diffutils`

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
- Make sure that you meet the requirements from [Prepare the database and run the rp](./deploy-development-rp.md) (do not create the database yet)
- Source the [helper script](../hack/e2e/run-rp-and-e2e.sh) to set the proper ENV variables.
- Run the rp
- Validate the RP is running properly by hitting the `/healthz` route
- Register a subscription where to run the e2e
- Create an openshift cluster
- Run the `make test-e2e` target
- Delete the openshift cluster, if applicable
- Delete the cosmos database, if applicable

You can also modify the flags passed to the e2e.test run by setting the E2E_FLAGS environment variable before running `make test-e2e`.

These steps can be acheived using commands below.  Look at the [e2e helper
file](../hack/e2e/run-rp-and-e2e.sh) to understand each of the bash functions
below.

## Smoke tests

We have some other tests under `test/e2e` that are not part of the CI.
These tests are labelled as `smoke` and are supposed to be run to check the basic functionality of the OCP cluster.
They can be used as a gap analysis for new OCP versions or the installer updates.

You can run the smoke tests by running the following command:

```bash
E2E_LABEL=smoke make test-e2e
```

If you want to run both e2e and smoke tests:

```bash
E2E_LABEL= make test-e2e
```

### Run a specific test

End to end tests are run using ginkgo. You can run subsets of tests or ignore some tests by following the [ginkgo documentation](https://onsi.github.io/ginkgo/#filtering-specs)

```bash
# source your environment file
. ./secrets/env

# set the CLUSTER env if you are testing locally
export CLUSTER=<cluster-name>

# source the e2e helper file
. ./hack/e2e/run-rp-and-e2e.sh

# Deploy a new DB if it does not exist yet
deploy_e2e_db

# build the rp binary
make aro

# run the RP as background process
run_rp

# validate if the RP is ready to receive requests
validate_rp_running

# create an openshift cluster if it does not exist yet
go run ./hack/cluster create

# Register the sub you are using to run e2e
register_sub

# Run e2e
make test-e2e

# delete the openshift cluster if applicable
go run ./hack/cluster delete

# Stop the local RP
kill_rp

# Delete the DB
clean_e2e_db
```

If you already created a dev cluster, you can run the e2e tests just by running the following command:

```bash
CLUSTER=<cluster-name> RESOURCEGROUP=<resource-group> make test-e2e
```

For smoke tests:

```bash
CLUSTER=<cluster-name> RESOURCEGROUP=<resource-group> E2E_LABEL=smoke make test-e2e
```
