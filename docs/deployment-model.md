# Deployment model

For better or worse, the ARO-RP codebase has four different deployment models.


## 1. Production deployment (PROD)

Running in production.  PROD deployments at a given commit are intended to be
identical (bar configuration) across all regions, regardless if the region is a
designated canary region (westcentralus / eastus2euap) or not.

Subscription [feature flags](feature-flags.md) are used to prevent end users
from accessing the ARO service in canary regions, or in regions or for
api-versions which are in the process of being built out.  The subscription used
for regular E2E service health checking has the relevant feature flags set.

The RP configures deny assignments on cluster resource groups only when running
in PROD.  This is because Azure policy only permits deny assignments to be set
by first party RPs when running in PROD.  The deny assignment functionality is
gated by the DisableDenyAssignments RP feature flag, which must be set in all
non-PROD deployments.


## 2. Pre-production deployment (INT)

INT deployment is intended to be as identical as possible to PROD, although
inevitably there are always some differences.

A subscription [feature flag](feature-flags.md) is used to selectively redirect
requests to the INT RP.

Here is a non-exhaustive list of differences between INT and PROD:

* INT is deployed entirely separately from PROD in the MSIT tenant, which does
  not have production access overheads.

* The INT ACR is entirely separate from PROD.

* INT uses different subdomains for hosting the RP service and clusters.

* INT does not use the production first party AAD application.  Instead it uses
  a multitenant AAD application which must be manually patched and granted
  permissions in any subscription where the RP will deploy clusters.

* There is standing access (i.e. no JIT) to the INT environment, INT elevated
  geneva actions and INT SRE portal.

* INT uses the Test instances of Geneva for RP and cluster logging and
  monitoring.  Geneva actions use separate credentials to authenticate to the
  INT RP.

* Monitoring of the INT environment does not match PROD monitoring.

* As previously mentioned, deny assignments are not enabled in INT.


## 3. Development deployment

A developer is able to deploy the entire ARO service stack in Azure in a way
that is intended to be as representative as possible of PROD/INT, and many ARO
service components can also be meaningfully run and debugged without being run
on Azure infrastructure at all.  This latter "local development mode" is also
currently used by our pull request E2E testing.

Some magic is needed to make all of this work, and this translates into a larger
delta from PROD/INT in some cases:

* Development deployment is entirely separate from INT and PROD and may in
  principal use any AAD tenant.

* Development uses different subdomains again for hosting the RP service and
  clusters.

* No inbound ARM layer

  In PROD/INT, service REST API requests are made to PROD ARM, and this proxies
  the requests to the RP service.  Thus PROD/INT RPs are configured to authorize
  only incoming service REST API requests from ARM.

  In development, ARM does not front the RP service, thus different authorizers
  are used.  In development mode, the authorizer used for ARM is also used for
  Geneva actions, so a developer can test Geneva actions manually.

  The ARO Go and Python client libraries in this repo carry patches such that
  they when the environment variable `RP_MODE=development` is set, they dial the
  RP on localhost with no authentication instead of dialling ARM.

  In addition, any HTTP headers injected by ARM via its proxying are unavailable
  in development mode.  For instance, the RP frontend fakes up the Referer
  header in this case, in order for client polling code to work correctly in
  development mode.

* No first party application

  In PROD, ARM is configured to automagically grant the RP first party
  application Owner on any resource group it creates in a customer subscription.

  In INT, the INT multitenant application which fakes the first party
  application is granted Owner on every subscription which is INT enabled.  This
  simple but has the disadvantage that the RP has more permissions in INT than
  it does in PROD.

  In development, pkg/env/armhelper.go fakes up ARM's automagic behaviour using
  a completely separate helper AAD application.  This makes setting up the
  development more onerous, but has the advantage that the RP's permissions in
  development match those in PROD.

* No cluster signed certificates

  Integration with Digicert is disabled in development mode.  This is controlled
  by the DisableSignedCertificates RP feature flag.

* No readiness delay

  In PROD/INT, the RP waits 2 minutes before indicating health to its load
  balancer, helping us to detect if the RP crash loops.  Similarly, it waits for
  frontend and backend tasks to complete before exiting.  To make the feature
  development/test cycle faster, these behaviours are disabled in development
  mode via the DisableReadinessDelay feature flag.

* There is standing access to development infrastructure using shared
  development credentials.

* Test instances of Geneva, matching INT, are used in development mode for
  cluster logging and monitoring (and RP logging and monitoring as appropriate).

* Development environments are not monitored.

* As previously mentioned, deny assignments are not enabled in development.

See [Prepare a shared RP development
environment](prepare-a-shared-rp-development-environment.md) for the process to
set up a development environment.  The same development AAD applications and
credentials are used regardless whether the RP runs on Azure or locally.


## 3a. Development on Azure

In the case that a developer deploys the entire ARO service stack in Azure, in
addition to the differences listed in section 3, note the following:

* Currently a separate ACR is created which must be populated with the latest
  OpenShift release.  TODO: this is inconvenient and adds expense.

* Service VMSS capacity is set to 1 instead of 3 (i.e. not highly available) to
  save time and money.

* Because the RP is internet-facing, TLS subject name and issuer authentication
  is required for all API accesses.

* hack/tunnel is used to forward RP API requests from a listener on localhost,
  wrapping these with the aforementioned TLS client authentication.


## 3b. Local development mode / CI

Many ARO service components can be meaningfully run and debugged locally on a
developer's laptop.  Notable exceptions include the deployment tooling including
the custom script extension which is used to initialize the RP VMSS.

"Local development mode" is also currently used by our pull request E2E testing.
This has the advantage of saving the time, money and flakiness that would be
implied by setting up an entire service stack on every PR.  However it is also
disadvantageous in the sense that coverage is less and the testing is less
representative.

When running in local development mode, in addition to the differences listed in
section 3, note the following:

* Local development mode is enabled, regardless of component, by setting the
  environment variable `RP_MODE=development`.  This enables code guarded by
  `env.IsLocalDevelopmentMode()` and also automatically sets many of the RP
  feature flags listed in section 3.

* All services listen on localhost only and authentication is largely disabled.

  The ARO Go and Python client libraries in this repo carry patches such that
  they when the environment variable `RP_MODE=development` is set, they dial the
  RP on localhost with no authentication instead of dialling ARM.

* Generation of ACR tokens per cluster is disabled; the INT ACR is used to pull
  OpenShift container images.

* Production VM instance metadata and MSI authorizers obviously don't work.
  These are fixed up using environment variables.  See
  pkg/util/instancemetadata.

* The INT/PROD mechanism of dialing a cluster API server whose private endpoint
  is on the RP vnet also obviously doesn't work.  Local development RPs share a
  proxy VM which is deployed on the RP vnet which can proxy these connections.
  See pkg/proxy.

* As a cost saving exercise, all local development RPs share a single Cosmos DB
  account (but containing a unique database per developer) per region.
