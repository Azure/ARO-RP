Commands issues during working sessions between RedHat and CloudFit to get the ARO-RP running locally on MacOS

PARENT_DOMAIN_NAME=osadev.cloud
PARENT_DOMAIN_RESOURCEGROUP=dns
SECRET_SA_ACCOUNT_NAME=rharosecretscf2
ADMIN_OBJECT_ID="$(az ad group show -g 'Azure Red Hat OpenShift MSFT Engineering' --query objectId -o tsv)"
echo $ADMIN_OBJECT_ID --> 2fdb57d4-3fd3-415d-b604-1d0e37a188fe
PULL_SECRET='{"auths":{"arosvc.azurecr.io":{"auth":"OTM5MDQ5YjQtNTllMS00YzlhLWJlYzgtMjAyZTAxZjc2MWFlOjZCLkpFOmZPT2hvLTI3P244TlYybDZqQS9UdjBMd1hm"},"arointsvc.azurecr.io":{"auth":"MmY2Y2VhNzktZjUyYi00YmNjLTk3MDQtMmNiZGM0YjYyMTM5OlM1fi1acF9icTFUYjFTNFpvOHNxS0dBMFpYV35pZjJVNTI="}}}'
AZURE_TENANT_ID=$(az account show --query tenantId -o tsv)
AZURE_SUBSCRIPTION_ID=$(az account show --query id -o tsv)

go run ./hack/genkey -client arm
mv arm.* secrets
AZURE_ARM_CLIENT_ID="$(az ad app create \
 --display-name aro-v4-arm-shared-cf \
 --query appId \
 -o tsv)"
az ad app credential reset \
 --id "$AZURE_ARM_CLIENT_ID" \
 --cert "$(base64 -b0 <secrets/arm.crt)" >/dev/null  
az ad sp create --id "$AZURE_ARM_CLIENT_ID" >/dev/null
AZURE_ARM_CLIENT_ID - aro-v4-arm-shared-cf - 9e1e2680-7835-46bb-912b-416b3f4c09c1 (this is app registration and also a service principal / enterprise application)
AZURE_ARM_CLIENT_ID=9e1e2680-7835-46bb-912b-416b3f4c09c1

go run ./hack/genkey -client firstparty
mv firstparty.* secrets
AZURE_FP_CLIENT_ID="$(az ad app create \
 --display-name aro-v4-fp-shared-cf \
 --query appId \
 -o tsv)"
az ad app credential reset \
 --id "$AZURE_FP_CLIENT_ID" \
 --cert "$(base64 -b0 <secrets/firstparty.crt)" >/dev/null
az ad sp create --id "$AZURE_FP_CLIENT_ID" >/dev/null
AZURE_FP_CLIENT_ID - aro-v4-fp-shared-cf - 659309db-b31e-4fe2-ab27-cab3f649fad9 (this is app registration and also a service principal / enterprise application)
AZURE_FP_CLIENT_ID=659309db-b31e-4fe2-ab27-cab3f649fad9

// F2F20FDB-B9EB-44F5-9027-89A61CF62183
AZURE_RP_CLIENT_SECRET="$(uuidgen)"
AZURE_RP_CLIENT_ID="$(az ad app create \
 --display-name aro-v4-rp-shared-cf \
 --end-date '2299-12-31T11:59:59+00:00' \
 --key-type password \
 --password "$AZURE_RP_CLIENT_SECRET" \
 --query appId \
 -o tsv)"
az ad sp create --id "$AZURE_RP_CLIENT_ID" >/dev/null
AZURE_RP_CLIENT_ID - aro-v4-rp-shared-cf - c75e8a68-34e6-413e-9687-398a4995198e (this is app registration and also a service principal / enterprise application)
AZURE_RP_CLIENT_SECRET=F2F20FDB-B9EB-44F5-9027-89A61CF62183
AZURE_RP_CLIENT_ID=c75e8a68-34e6-413e-9687-398a4995198e

// AB7FB79C-46CD-4C3C-AEB6-2D8FBBC6313D
AZURE_GATEWAY_CLIENT_SECRET="$(uuidgen)"
AZURE_GATEWAY_CLIENT_ID="$(az ad app create \
 --display-name aro-v4-gateway-shared-cf \
 --end-date '2299-12-31T11:59:59+00:00' \
 --key-type password \
 --password "$AZURE_GATEWAY_CLIENT_SECRET" \
 --query appId \
 -o tsv)"
az ad sp create --id "$AZURE_GATEWAY_CLIENT_ID" >/dev/null
AZURE_GATEWAY_CLIENT_ID - aro-v4-gateway-shared-cf - 3045d26d-caa4-4aed-aef5-c29854825676 (this is app registration and also a service principal / enterprise application)
AZURE_GATEWAY_CLIENT_SECRET=AB7FB79C-46CD-4C3C-AEB6-2D8FBBC6313D
AZURE_GATEWAY_CLIENT_ID=3045d26d-caa4-4aed-aef5-c29854825676

// 898BEC6A-9147-4C9B-8B8A-CA5992C42328
AZURE_CLIENT_SECRET="$(uuidgen)"
AZURE_CLIENT_ID="$(az ad app create \
 --display-name aro-v4-tooling-shared-cf \
 --end-date '2299-12-31T11:59:59+00:00' \
 --key-type password \
 --password "$AZURE_CLIENT_SECRET" \
 --query appId \
 -o tsv)"
az ad sp create --id "$AZURE_CLIENT_ID" >/dev/null
AZURE_CLIENT_ID - aro-v4-tooling-shared-cf - 81bc2ad6-3025-4b9f-8d9b-6dbe7b49d6e4 (this is app registration and also a service principal / enterprise application)
AZURE_CLIENT_SECRET=898BEC6A-9147-4C9B-8B8A-CA5992C42328
AZURE_CLIENT_ID=81bc2ad6-3025-4b9f-8d9b-6dbe7b49d6e4

// Added ability to customize the fp role def id to avoid interferring with other subs
LOCATION=eastus
az deployment sub create \
 -l $LOCATION \
 --template-file deploy/rbac-development.json \
 --parameters \
   "armServicePrincipalId=$(az ad sp list --filter "appId eq '$AZURE_ARM_CLIENT_ID'" --query '[].objectId' -o tsv)" \
   "fpServicePrincipalId=$(az ad sp list --filter "appId eq '$AZURE_FP_CLIENT_ID'" --query '[].objectId' -o tsv)" \
   "fpRoleDefinitionId"="$(uuidgen)" \
   "devServicePrincipalId=$(az ad sp list --filter "appId eq '$AZURE_CLIENT_ID'" --query '[].objectId' -o tsv)" \
 >/dev/null

ERROR (prior to allowing custom guid for fp role def id):

WARNING: The underlying Active Directory Graph API will be replaced by Microsoft Graph API in a future version of Azure CLI. Please carefully review all breaking changes introduced during this migration: https://docs.microsoft.com/cli/azure/microsoft-graph-migration
WARNING: The underlying Active Directory Graph API will be replaced by Microsoft Graph API in a future version of Azure CLI. Please carefully review all breaking changes introduced during this migration: https://docs.microsoft.com/cli/azure/microsoft-graph-migration
WARNING: The underlying Active Directory Graph API will be replaced by Microsoft Graph API in a future version of Azure CLI. Please carefully review all breaking changes introduced during this migration: https://docs.microsoft.com/cli/azure/microsoft-graph-migration
ERROR: {"status":"Failed","error":{"code":"DeploymentFailed","message":"At least one resource deployment operation failed. Please list deployment operations for details. Please see https://aka.ms/DeployOperations for usage details.","details":[{"code":"BadRequest","message":"{\r\n  \"error\": {\r\n    \"code\": \"RoleAssignmentUpdateNotPermitted\",\r\n    \"message\": \"Tenant ID, application ID, principal ID, and scope are not allowed to be updated.\"\r\n  }\r\n}"},{"code":"Forbidden","message":"{\r\n  \"error\": {\r\n    \"code\": \"LinkedAuthorizationFailed\",\r\n    \"message\": \"The client 'v-cperkins@microsoft.com' with object id 'fa22c3cf-f51f-443c-abeb-830c405d24c7' has permission to perform action 'Microsoft.Authorization/roleDefinitions/write' on scope '/subscriptions/26c7e39e-2dfa-4854-90f0-6bc88f7a0fb8'; however, it does not have permission to perform action 'Microsoft.Authorization/roleDefinitions/write' on the linked scope(s) '/subscriptions/46626fc5-476d-41ad-8c76-2ec49c6994eb' or the linked scope(s) are invalid.\"\r\n  }\r\n}"}]}}

https://ms.portal.azure.com/#blade/HubsExtension/DeploymentDetailsBlade/overview/id/%2Fsubscriptions%2F26c7e39e-2dfa-4854-90f0-6bc88f7a0fb8%2Fproviders%2FMicrosoft.Resources%2Fdeployments%2Frbac-development

go run ./hack/genkey -client portal-client
mv portal-client.* secrets

AZURE_PORTAL_CLIENT_ID="$(az ad app create \
 --display-name aro-v4-portal-shared-cf \
 --reply-urls "https://localhost:8444/callback" \
 --query appId \
 -o tsv)"
az ad app credential reset \
 --id "$AZURE_PORTAL_CLIENT_ID" \
 --cert "$(base64 -b0 <secrets/portal-client.crt)" >/dev/null

az rest --method PATCH \
  --uri https://graph.microsoft.com/v1.0/applications/36a3e030-3ae6-483d-8cd8-710dd23b87d8/ \
  --body '{"api":{"requestedAccessTokenVersion": 2}}'
AZURE_PORTAL_CLIENT_ID=b11e4b0e-bafa-420b-ae75-108b6ea45198

AZURE_DBTOKEN_CLIENT_ID="$(az ad app create --display-name dbtoken-cf \
  --oauth2-allow-implicit-flow false \
  --query appId \
  -o tsv)"

OBJ_ID="$(az ad app show --id $AZURE_DBTOKEN_CLIENT_ID --query objectId)"

// NOTE: the graph API requires this to be done from a managed machine
az rest --method PATCH \
  --uri https://graph.microsoft.com/v1.0/applications/$OBJ_ID/ \
  --body '{"api":{"requestedAccessTokenVersion": 2}}'
AZURE_DBTOKEN_CLIENT_ID=c3cfda35-62ea-4850-927f-100b33678ec8
OBJ_ID=36a3e030-3ae6-483d-8cd8-710dd23b87d8

go run ./hack/genkey -ca vpn-ca
mv vpn-ca.* secrets

go run ./hack/genkey -client -keyFile secrets/vpn-ca.key -certFile secrets/vpn-ca.crt vpn-client
mv vpn-client.* secrets

go run ./hack/genkey proxy
mv proxy.* secrets

go run ./hack/genkey -client proxy-client
mv proxy-client.* secrets

ssh-keygen -f secrets/proxy_id_rsa -N ''
Your identification has been saved in secrets/proxy_id_rsa
Your public key has been saved in secrets/proxy_id_rsa.pub
The key fingerprint is:
SHA256:1q+ZTzkM0GOyfhMzvCUllBWva5Z1cWvf56NI7sIubkc corey@MacBook-Pro-2222.lan
The key's randomart image is:
+---[RSA 3072]----+
|          .oo.   |
|         o.  .   |
|        o = . ...|
|         B + .  +|
|        S O o .o.|
|       o E @ =..o|
|        + +.@   +|
|       o =oO.. o.|
|      o.+.*=... o|
+----[SHA256]-----+


go run ./hack/genkey localhost
mv localhost.* secrets

go run ./hack/genkey -ca dev-ca
mv dev-ca.* secrets

go run ./hack/genkey -client -keyFile secrets/dev-ca.key -certFile secrets/dev-ca.crt dev-client
mv dev-client.* secrets

## pickup at Environment File

RESOURCEGROUP_PREFIX=v4
PROXY_DOMAIN_NAME_LABEL=aroproxy

// don't accidently let the closing EOF be indented, classic mistake I always make!
cat >env <<EOF                                                            
   export AZURE_TENANT_ID='$AZURE_TENANT_ID'
   export AZURE_SUBSCRIPTION_ID='$AZURE_SUBSCRIPTION_ID'
   export AZURE_ARM_CLIENT_ID='$AZURE_ARM_CLIENT_ID'
   export AZURE_FP_CLIENT_ID='$AZURE_FP_CLIENT_ID'
   export AZURE_FP_SERVICE_PRINCIPAL_ID='$(az ad sp list --filter "appId eq '$AZURE_FP_CLIENT_ID'" --query '[].objectId' -o tsv)'
   export AZURE_DBTOKEN_CLIENT_ID='$AZURE_DBTOKEN_CLIENT_ID'
   export AZURE_PORTAL_CLIENT_ID='$AZURE_PORTAL_CLIENT_ID'
   export AZURE_PORTAL_ACCESS_GROUP_IDS='$ADMIN_OBJECT_ID'
   export AZURE_PORTAL_ELEVATED_GROUP_IDS='$ADMIN_OBJECT_ID'
   export AZURE_CLIENT_ID='$AZURE_CLIENT_ID'
   export AZURE_SERVICE_PRINCIPAL_ID='$(az ad sp list --filter "appId eq '$AZURE_CLIENT_ID'" --query '[].objectId' -o tsv)'
   export AZURE_CLIENT_SECRET='$AZURE_CLIENT_SECRET'
   export AZURE_RP_CLIENT_ID='$AZURE_RP_CLIENT_ID'
   export AZURE_RP_CLIENT_SECRET='$AZURE_RP_CLIENT_SECRET'
   export AZURE_GATEWAY_CLIENT_ID='$AZURE_GATEWAY_CLIENT_ID'
   export AZURE_GATEWAY_SERVICE_PRINCIPAL_ID='$(az ad sp list --filter "appId eq '$AZURE_GATEWAY_CLIENT_ID'" --query '[].objectId' -o tsv)'
   export AZURE_GATEWAY_CLIENT_SECRET='$AZURE_GATEWAY_CLIENT_SECRET'
   export RESOURCEGROUP="$RESOURCEGROUP_PREFIX-\$LOCATION"
   export PROXY_HOSTNAME="vm0.$PROXY_DOMAIN_NAME_LABEL.\$LOCATION.cloudapp.azure.com"
   export DATABASE_NAME="\$USER"
   export RP_MODE='development'
   export PULL_SECRET='$PULL_SECRET'
   export SECRET_SA_ACCOUNT_NAME='$SECRET_SA_ACCOUNT_NAME'
   export DATABASE_ACCOUNT_NAME="\$RESOURCEGROUP"
   export KEYVAULT_PREFIX="\$RESOURCEGROUP"
   export ADMIN_OBJECT_ID='$ADMIN_OBJECT_ID'
   export PARENT_DOMAIN_NAME='$PARENT_DOMAIN_NAME'
   PARENT_DOMAIN_RESOURCEGROUP='$PARENT_DOMAIN_RESOURCEGROUP'
   export DOMAIN_NAME="\$LOCATION.\$PARENT_DOMAIN_NAME"
   export AZURE_ENVIRONMENT='AzurePublicCloud'
EOF

** had trouble with make secrets-update and make secrets not wanting to read env vars, had to manually hack to move on

***hack in create pkg/util/cluster/cluster.go (this is so that our env vars from secrets/env are used)
  // CDP-PR: Make a PR for this that uses RP_MODE?
  // CDP-DOC: ZachJ modified so we could utilize AAD info created earlier in shared setup
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

Solution:
Karan used these queries and found the following error that was being hiddened by the error above:

ShoeboxEntries
| where resourceId contains "/subscriptions/26c7e39e-2dfa-4854-90f0-6bc88f7a0fb8/resourceGroups/aro-aro-cdp-cf"
| where TIMESTAMP > ago(1d)
| where resultType == "Failure"
| where correlationId == "01928c62-35cf-4d58-be0d-c509ae1a26b8"
| order by TIMESTAMP desc 


HttpIncomingRequests
| where correlationId == "f50ed338-7d9f-45ae-8e2e-c11e428485d5"
| where TIMESTAMP > ago(1d)

{"statusCode":"BadRequest","serviceRequestId":null,"statusMessage":"{\"error\":{\"code\":\"SubnetsHaveNoServiceEndpointsConfigured\",\"message\":\"Subnets rp-subnet of virtual network /subscriptions/26c7e39e-2dfa-4854-90f0-6bc88f7a0fb8/resourceGroups/v5-eastus/providers/Microsoft.Network/virtualNetworks/rp-vnet do not have ServiceEndpoints for Microsoft.Storage resources configured. Add Microsoft.Storage to subnet's ServiceEndpoints collection before try

Karan created the service endpoint for our testing manually and this worked.
Corey updated rp-development-predeploy.json with service endpoints for rp-subnet under rp-vnet. I used the same service endpointsw from rp-production-predeploy.json. This template has not yet been tested. Need to recreate the shared RP to do so. Or, just re-run that shell script.

Next Error:
_id= component=backend correlation_id= request_id=59b67467-2a9c-4fc9-bd3b-f5d08fd3cdf6 resource_group=v5-eastus resource_id=/subscriptions/26c7e39e-2dfa-4854-90f0-6bc88f7a0fb8/resourcegroups/v5-eastus/providers/microsoft.redhatopenshift/openshiftclusters/aro-cdp-cf-5 resource_name=aro-cdp-cf-5 subscription_id=26c7e39e-2dfa-4854-90f0-6bc88f7a0fb8
ERRO[2022-03-25T11:18:07-05:00]pkg/util/steps/runner.go:34 steps.Run() step [AuthorizationRefreshingAction [Action github.com/Azure/ARO-RP/pkg/cluster.(*manager).validateResources-fm]] encountered error: 400: InvalidLinkedVNet: properties.masterProfile.subnetId: The provided subnet '/subscriptions/26c7e39e-2dfa-4854-90f0-6bc88f7a0fb8/resourceGroups/v5-eastus/providers/Microsoft.Network/virtualNetworks/dev-vnet/subnets/aro-cdp-cf-5-master' is invalid: must not have a network security group attached.  client_principal_name= client_request_id= component=backend correlation_id= request_id=59b67467-2a9c-4fc9-bd3b-f5d08fd3cdf6 resource_group=v5-eastus resource_id=/subscriptions/26c7e39e-2dfa-4854-90f0-6bc88f7a0fb8/resourcegroups/v5-eastus/providers/microsoft.redhatopenshift/openshiftclusters/aro-cdp-cf-5 resource_name=aro-cdp-cf-5 subscription_id=26c7e39e-2dfa-4854-90f0-6bc88f7a0fb8

FATA[2022-03-25T11:18:14-05:00]hack/cluster/cluster.go:66 main.main() Code="InvalidLinkedVNet" Message="The provided subnet '/subscriptions/26c7e39e-2dfa-4854-90f0-6bc88f7a0fb8/resourceGroups/v5-eastus/providers/Microsoft.Network/virtualNetworks/dev-vnet/subnets/aro-cdp-cf-5-master' is invalid: must not have a network security group attached." Target="properties.masterProfile.subnetId"
exit status 1

**** This could be corruption from the many clusters we have created. I am going to cleanup and recreate my user cosmos db and try again.
** Cleaned up all rgs, and other resources
** emptied database records
** creating cluster aro-cdp-cf-uno....

Changes I Made Manually:

I created a new VPN in rp-vnet from Azure Portal. To do this, I had to first update the Address Space of rp-vnet to allow creation of a new VPN. For now i used the same public certificate we use for dev-vpn 

Steps to Create Cluster:

1. Go to VPN Point-to-Site Configuration rp-vnet p2s and download the VPN Client Certificate

2. Store the OpenVPN client certificate as secrets/vpn-rp-eastus.ovpn

3. Update the P2S client certificate and P2S client certificate private key to be same as secrets/vpn-eastus.ovpn

4. run sudo openvpn secrets/vpn-rp-eastus.ovpn

5. Update nsg priority to 120 here https://github.com/CloudFitSoftware/ARO-RP/blob/cfs/rh-cf-rp-dev-env-working-sessions/pkg/cluster/nsg.go#L34

6. Update PROXY_HOSTNAME environment variable to point to internal ip of Proxy VM export PROXY_HOSTNAME="10.0.0.4"

7. source updated environment file . ./env

8. run make run-localrp

9. in another terminal run CLUSTER=<cluster_name> go run ./hack/cluster create



Steps to access Cluster:

1. Once / If cluster creates, celebrate

2. run CLUSTER=<cluster_name> make admin.kubeconfig

3. disconnect from rp-vnet vpn and connect to dev-vnet vpn sudo openvpn secrets/vpn-eastus.ovpn

4. change newly created admin.kubeconfig cluster.server to point to internal loadbalancer ip (get this internal load balancer from azure resource group for your cluster

Eg: change server: https://api.kmagdani-test-rh.v4-eastus.osadev.cloud:6443 to something like server: https://10.x.x.x:6443

5. export KUBECONFIG=$(pwd)/admin.kubeconfig

6. Run kubectl/oc get nodes --insecure-skip-tls-verify to verify you can get cluster objects

