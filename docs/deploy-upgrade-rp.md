# Deploy/Upgrade RP

The `deploy` directory contains artifacts for manual environment deployment.
Production deployment and upgrade can be orchestrated using deployment tooling:
`go run ./cmd/aro deploy config.yaml location`

The deploy utility is decoupled from the `env` package and is configured with a
config file (see config.yaml.example).

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

* Deploy per subscription resources in every subscription `rp-production-subscription.json`.

* Configure service key vault.

* Deploy main deployment resources `rp-production.json` with
  `rp-production-parameters.json`.

* Configure DNS.

* Wait for new RP readiness.

* Terminate all old RP VMSSes.
