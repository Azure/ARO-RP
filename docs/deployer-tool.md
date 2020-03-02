# Deploy/Upgrade RP

`deploy` directory contains artifacts for manual environment deployment.
Production deployment and upgrade can be orchestrated using deployment tooling:
`go run ./cmd/aro deploy`

Utility is decoupled from `env` package and requires these variables to run:
```
AZURE_TENANT_ID
AZURE_CLIENT_SECRET
AZURE_CLIENT_ID
AZURE_SUBSCRIPTION_ID - Required
AZURE_RP_PARAMETERS_FILE - Required, location of environment parameters file
AZURE_RP_RESOURCEGROUP_NAME - Required, RP resource group name
AZURE_RP_VERSION - overrides gitCommit version for the RP VMSS
```

* If ran on existing ResourceGroup it will update corresponding resources, as per generators output.
* New VMSS will be created with postfix `-short_gitcommit`
* Parameters file example - `deploy/rp-production-parameters.json`
* Utility will not re-deploy re-deploy phase if deployment already exist. If you
want to re-deploy - delete existing deployment object.

