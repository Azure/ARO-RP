Documentation is a work in progress.

# aro deploy

## Run via ADO

Environment is picked up from instance metadata endpoint, no env vars are required to be set.

## Run via Ev2

Ev2 will run `aro deploy` in ACI which cannot access instance metadata endpoint (169.254.169.254).  We have all the info needed already so we make it available via environment variables.  In addition, a variable indicating we're running in Ev2 is required to ensure current non-Ev2 behavior is not impacted.

Environment variables you **must** set:
* AZURE_EV2: any value other than empty string
* AZURE_ENVIRONMENT: the cloud environment name from go-autorest environment names
* AZURE_SUBSCRIPTION_ID: the target subscription ID
* AZURE_TENANT_ID: the target tenant ID
* LOCATION: the target location
* RESOURCEGROUP: the target resource group name

Environment variables you may **optionally** set:
* HOSTNAME_OVERRIDE: in case default behavior doesn't give a value we want for hostname, pass in an override here.  This can happen when running `aro deploy` inside an Azure Conainer Instance (ACI) where the hostname is not meaningful and we wish to use the host's hostname.
