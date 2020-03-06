
# RP Upgrade/Deploy tooling High-Level design


**Goal**: To have automated, replicatable RP deployment and upgrade tooling to enable fast rollout/upgrade/lifecycle of RP infrastructure.

**Requirements**:

-   Phase based deployment (for Vault configuration)
	-   PreDeploy - ManagedIdentity, Vaults (for certificate enablement)
	-   Deploy - All other resources
	-   Upgrade - gradually upgrade old RP
-   Idempotent
-   Configuration for deployment
-   Automated secret rotation (not now)

**Proposal**:
Command line tool:
`./aro deploy {region}`
Generic configuration file config.yaml:
```yaml
rps:
 - location: eastus
    subscriptionId: <subscriptionID>
    resourceGroupName: <ResourceGroupName>
 - location: westus
    subscriptionId: <subscriptionID>
    resourceGroupName: <ResourceGroupName>
    configuration:
       <same structure as global configuration>
configuration:
  databaseAccountName:
  domainName:
  extraCosmosDBIPs:
  extraKeyvaultAccessPolicies:
  fpServicePrincipalId:
  mdmFrontendUrl :
  mdsdConfigVersion:
  mdsdEnvironment:
  pullSecret:
  rpImage:
  rpImageAuth:
  rpMode:
  rpServicePrincipalId:
  sshPublicKey:
  vmssName:
```

**Flow**:

-   Each region can have substructure of same configuration struct as root. Individual regional configuration structs act as overrides for global configurable
-   For the start configuration sub-structure will be YAML representation of ARM parameters[1] files, used at the moment.

Install:
	-   PreDeploy:
	-   Create ResourceGroup
	-   Create ManagedIdentity
	-   Create Vault
Deploy:
	-   Setup encryption keys
	-   Submit a request for Certificates to be generated and wait for certificates
	 -   Create new certificates and wait
	-   Create A and NS global records for RP
	-   Initiate ACR new replica (optional for now)
	-   Deploy RP infrastructure
	-   Wait for RP to come online
	-   Deploy external azure integration components (optional for now)
Upgrade
	-   If required run PreDeploy (configured via env hook)
	-   Request new certificates for rotation (optional for now, requires RP core code changes)
	-   Deploy the new version of RP and wait for readiness
	-   Gradually terminate old RP VMSS and wait for them to report “stopped”
	-   Retire old RP VMSS instance.

1.  https://github.com/Azure/ARO-RP/blob/master/deploy/rp-production-parameters.json
