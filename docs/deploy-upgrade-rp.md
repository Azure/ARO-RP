# Deploy/Upgrade RP

The `deploy` directory contains artifacts for manual environment deployment.
Production deployment and upgrade can be orchestrated using deployment tooling:
`go run ./cmd/aro deploy config.yaml location`

The deploy utility is decoupled from the `env` package and is configured with a
config file (see config.yaml.example) and the following optional
environment variables:

* RP_VERSION: RP VM scaleset git commit version

* RP_PREDEPLOY_ONLY: exit after pre-deploy step

Notes:

* If the deployment tool is run on an existing resource group, it will update
  the resources according to the deployment artifacts.

* The new RP VMSS will be created with postfix `-short_gitcommit`.

## Deployment logical order:

* Deploy global subscription-level resources
  `rp-production-global-subscription.json`.

* Deploy managed identity `rp-production-managed-identity.json`. This will
  produce `rpServicePrincipalId` required by next deployments.

* Deploy global resources `rp-production-global.json`.

* Deploy pre-deploy resources `rp-production-predeploy.json` with
  `rp-production-predeploy-parameters.json`.

* Configure service key vault.

* Deploy main deployment resources `rp-production.json` with
  `rp-production-parameters.json`.

* Configure DNS.

* Wait for new RP readiness.

* Terminate all old RP VMSSes.

## Utility example

```bash
# run pre-deploy phase only
export RP_PARAMETERS_FILE=rp-production-predeploy-parameters.json
RP_PREDEPLOY_ONLY=true go run ./cmd/aro deploy

# deploy RP under name test
export RP_VERSION="test"
export RP_PARAMETERS_FILE=rp-production-parameters.json
go run ./cmd/aro deploy

# deploy second VMSS instance with name test2 and retire test
export RP_VERSION="test2"
export RP_PARAMETERS_FILE=rp-production-parameters.json
go run ./cmd/aro deploy
```
