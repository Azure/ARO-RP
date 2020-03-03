# Deploy/Upgrade RP

The `deploy` directory contains artifacts for manual environment deployment.
Production deployment and upgrade can be orchestrated using deployment tooling:
`go run ./cmd/aro deploy`

The deploy utility is decoupled from the `env` package and is configured using
the following environment variables:

* Required:

  * AZURE_SUBSCRIPTION_ID: RP subscription ID

  * LOCATION: RP location

  * RESOURCEGROUP: RP resource group name

  * RP_PARAMETERS_FILE: location of environment parameters file (same variable
  used for PreDeploy and Deploy)

* Optional:

  * RP_VERSION: RP VM scaleset git commit version

  * RP_PREDEPLOY_ONLY: exit after pre-deploy step


Notes:

* If the deployment tool is run on an existing resource group, it will update
  the resources according to the deployment artifacts.

* The new RP VMSS will be created with postfix `-short_gitcommit`.

* Parameters file example `deploy/rp-production-parameters.json`.

* The utility will not re-deploy rp-production-predeploy.json if the deployment
  already exists. If you want to re-deploy rp-production-predeploy.json, delete
  existing deployment object.

## Deployment logical order:

* Deploy managed identity `rp-production-managed-identity.json`. This will produce
  `rpServicePrincipalId` required by next deployments.

* Deploy pre-deploy resources `rp-production-predeploy.json` with `rp-production-predeploy-parameters.json`

* Deploy main deployment resources `rp-production.json` with `rp-production-parameters.json`


## Utility example

```bash
# run pre-deploy phase only
export RP_PREDEPLOY_ONLY=true
export RP_PARAMETERS_FILE=parameters-predeploy.json
go run ./cmd/aro deploy

# deploy RP under name "test
unset RP_PREDEPLOY_ONLY
export RP_VERSION="test"
export RP_PARAMETERS_FILE=parameters.json
go run ./cmd/aro deploy

# deploy second VMSS instance with name test2 and retire test
export RP_VERSION="test2"
export RP_PARAMETERS_FILE=parameters.json
go run ./cmd/aro deploy
```
