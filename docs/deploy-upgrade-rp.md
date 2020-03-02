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

  * RP_PARAMETERS_FILE: location of environment parameters file (if
    RP_PREDEPLOY_ONLY is not set)

* Optional:

  * RP_VERSION: RP VM scaleset git commit version

  * RP_PREDEPLOY_ONLY: exit after pre-deploy step


Notes:

* If the deployment tool is run on an existing resource group, it will update
  the resources according to the deployment artifacts.

* The new RP VMSS will be created with postfix `-short_gitcommit`.

* Parameters file exampleL `deploy/rp-production-parameters.json`.

* The utility will not re-deploy rp-production-nsg.json if the deployment
  already exists. If you want to re-deploy rp-production-nsg.json, delete
  existing deployment object.
