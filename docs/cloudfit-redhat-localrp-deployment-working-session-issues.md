### TODO: Make PR for update to ./deploy/rbac-development.json that allows for a custom fpRoleDefinitionId (https://github.com/Azure/ARO-RP/pull/2055)
### TODO: Make PR for similar issue on deleting a cluster
### TODO: Make PR for this issue separate from this branch (below). Just suggest RP-MODE check?
***hack in create pkg/util/cluster/cluster.go (this is so that our env vars from secrets/env are used)
  appID := os.Getenv("AZURE_CLIENT_ID")
  appSecret := os.Getenv("AZURE_CLIENT_SECRET")
  if !(appID != "" && appSecret != "") {
    if appID == "" && appSecret == "" {
      c.log.Infof("creating AAD application")
      appID, appSecret, err = c.createApplication(ctx, "aro-"+clusterName)
      if err != nil {
        return err
      }
    } else {
      return fmt.Errorf("fp service principal id is not found")
    }
  }
  spID := os.Getenv("AZURE_SERVICE_PRINCIPAL_ID")
  if spID == "" {
    spID, err = c.createServicePrincipal(ctx, appID)
    if err != nil {
      return err
    }
  }

  // CDP-DOC: Document this change in the updates to RH.
  /*
  appID, appSecret, err := c.createApplication(ctx, "aro-"+clusterName)
  if err != nil {
    return err
  }

  spID, err := c.createServicePrincipal(ctx, appID)
  if err != nil {
    return err
  }
  */


Error: Happens on the deployments into the new aro cluster's GV
{
    "status": "Failed",
    "error": {
        "code": "ResourceDeploymentFailure",
        "message": "The response for resource had empty or invalid content."
    }
}

Issue:
{"statusCode":"BadRequest","serviceRequestId":null,"statusMessage":"{\"error\":{\"code\":\"SubnetsHaveNoServiceEndpointsConfigured\",\"message\":\"Subnets rp-subnet of virtual network /subscriptions/26c7e39e-2dfa-4854-90f0-6bc88f7a0fb8/resourceGroups/v5-eastus/providers/Microsoft.Network/virtualNetworks/rp-vnet do not have ServiceEndpoints for Microsoft.Storage resources configured. Add Microsoft.Storage to subnet's ServiceEndpoints collection before try

### TODO: Make PR for this issue separate from this branch
Karan created the service endpoint for our testing manually and this worked.
Corey updated rp-development-predeploy.json with service endpoints for rp-subnet under rp-vnet. I used the same service endpointsw from rp-production-predeploy.json. This template has not yet been tested. Need to recreate the shared RP to do so. Or, just re-run that shell script.

Issue: Karan located an nsg priority issue
### TODO: Make PR for this issue separate from this branch
Issue: We ran into an NSG Priority problem:
Updated nsg priority to 120 here https://github.com/CloudFitSoftware/ARO-RP/blob/cfs/rh-cf-rp-dev-env-working-sessions/pkg/cluster/nsg.go#L34
