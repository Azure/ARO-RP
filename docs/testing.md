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

### Run tests to private clusters

If you want to run e2e tests to private clusters, you need VPN access to the cluster.

#### hack script

If you are using hack script to create the cluster, you already have the VPN access.

```bash
sudo openvpn secrets/vpn-eastus.ovpn  # for eastus
# sudo openvpn secrets/vpn-aks-westeurope.ovpn  # for westeurope
# sudo openvpn secrets/vpn-aks-australiaeast.ovpn  # for australiaeast

CLUSTER=<cluster-name> RESOURCEGROUP=<resource-group> make test-e2e
```

#### az cli

If you are using az cli, and your virtual network doesn't have VPN gateway, you need to create VPN gateway and connect to it.
This is an example script to create VPN gateway and its client.

```bash
# Set the variables
RESOURCE_GROUP=
VNET=

GATEWAY_SUBNET=10.0.4.0/23
ADDRESS_PREFIX=192.168.0.0/16
VPNGW=vpn-gateway
VPNGW_PUBLIC_IP=vpn-gateway-ip
VPN_ROOT=/tmp/vpn-root
VPN_CLIENT=/tmp/vpn-client
VPN_CLIENT_CONF=/tmp/myvpn.ovpn

# Create VPN gateway
az network vnet subnet create \
--vnet-name=$VNET \
-n GatewaySubnet \
-g $RESOURCE_GROUP \
--address-prefix $GATEWAY_SUBNET

az network public-ip create \
-n $VPNGW_PUBLIC_IP \
-g $RESOURCE_GROUP

az network vnet-gateway create \
-n $VPNGW \
--public-ip-address $VPNGW_PUBLIC_IP \
-g $RESOURCE_GROUP \
--vnet $VNET \
--gateway-type Vpn \
--sku VpnGw2 \
--vpn-gateway-generation Generation2 \
--address-prefixes $ADDRESS_PREFIX \
--client-protocol OpenVPN

go run ./hack/genkey -ca $VPN_ROOT
go run ./hack/genkey -client -keyFile $VPN_ROOT.key -certFile $VPN_ROOT.crt $VPN_CLIENT

az network vnet-gateway root-cert create \
-g $RESOURCE_GROUP \
-n dev-vpn \
--gateway-name $VPNGW \
--public-cert-data $VPN_ROOT.crt

# Generate VPN client configuration
curl -so vpnclientconfiguration.zip "$(az network vnet-gateway vpn-client generate \
    -g "$RESOURCE_GROUP" \
    -n "$VPNGW" \
    -o tsv)"
export CLIENTCERTIFICATE="$(openssl x509 -inform der -in $VPN_CLIENT.crt)"
export PRIVATEKEY="$(openssl pkey -inform der -in $VPN_CLIENT.key)"
unzip -qc vpnclientconfiguration.zip 'OpenVPN\\vpnconfig.ovpn' \
    | envsubst \
    | grep -v '^log ' > "$VPN_CLIENT_CONF"
rm vpnclientconfiguration.zip
```

After creating the VPN gateway, you can connect to it using the following command:

```bash
sudo openvpn --config $VPN_CLIENT.ovpn
CLUSTER=<cluster-name> RESOURCEGROUP=<resource-group> make test-e2e
```

### Run tests to upgraded clusters

To run e2e (smoke) tests to upgraded clusters, run the following command:

```bash
oc adm upgrade channel <channel>  # e.g., oc adm upgrade channel stable-4.14
oc adm upgrade --to=<version>  # e.g., oc adm upgrade --to=4.14.16
``` 