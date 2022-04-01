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
   export PROXY_HOSTNAME="vm0.$PROXY_DOMAIN_NAME_LABEL.\$LOCATION.cloudapp.azure.com" (this changes to IP when connected to rp-vpn)
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

{"statusCode":"BadRequest","serviceRequestId":null,"statusMessage":"{\"error\":{\"code\":\"SubnetsHaveNoServiceEndpointsConfigured\",\"message\":\"Subnets rp-subnet of virtual network /subscriptions/26c7e39e-2dfa-4854-90f0-6bc88f7a0fb8/resourceGroups/v5-eastus/providers/Microsoft.Network/virtualNetworks/rp-vnet do not have ServiceEndpoints for Microsoft.Storage resources configured. Add Microsoft.Storage to subnet's ServiceEndpoints collection before try

Karan created the service endpoint for our testing manually and this worked.
Corey updated rp-development-predeploy.json with service endpoints for rp-subnet under rp-vnet. I used the same service endpointsw from rp-production-predeploy.json. This template has not yet been tested. Need to recreate the shared RP to do so. Or, just re-run that shell script.

Issue: We ran into an NSG Priority problem:
Updated nsg priority to 120 here https://github.com/CloudFitSoftware/ARO-RP/blob/cfs/rh-cf-rp-dev-env-working-sessions/pkg/cluster/nsg.go#L34

# Preparation to Create Cluster:

1. Update the Address Space of "rp-vnet" to allow for creation of a new VPN. You should be able to do this at: https://ms.portal.azure.com/#@microsoft.onmicrosoft.com/resource/subscriptions/<subscription-id>/resourceGroups/<aro-shared-rp-rg>/providers/Microsoft.Network/virtualNetworks/rp-vnet/addressSpace. See the table below for what we did.

| Address Space | Address Range         | Address Count |
| ------------- | --------------------- | ------------- |
| 10.0.0.0/24   | 10.0.0.0 - 10.0.0.255 | 256 |
| 10.1.0.0/24   | 10.1.0.0 - 10.1.0.255 | 256 |

1. Create a new "Virtual Network Gateway (Gateway type: VPN)" in the Azure Portal manually. This needs to be configured to the "Virtual Network" named "rp-vnet" which will already existing in the shared RP's resource group. This new VPN will allow the local ARO-RP to connect to the to the existing "rp-vnet" to create a cluster. 

1. Configure the new "rp-vnet" VPN with the same public certificate used for the existing dev-vpn. This is done at: https://ms.portal.azure.com/#@microsoft.onmicrosoft.com/resource/subscriptions/<subscription-id>/resourceGroups/<aro-shared-rp-rg>/providers/Microsoft.Network/virtualNetworkGateways/rp-vnet/pointtositeconfiguration. You can simply copy the info from: https://ms.portal.azure.com/#@microsoft.onmicrosoft.com/resource/subscriptions/<subscription-id>/resourceGroups/<aro-shared-rp-rg>/providers/Microsoft.Network/virtualNetworkGateways/dev-vpn/pointtositeconfiguration.

1. Connect to the "rp-vnet" VPN created above. You can use openvpn or the azure vpn client, both have worked fine in our testing. 
  1. Go to Point-to-Site Configuration for "rp-vnet" (https://ms.portal.azure.com/#@microsoft.onmicrosoft.com/resource/subscriptions/<subscription-id>/resourceGroups/<aro-shared-rp-rg>/providers/Microsoft.Network/virtualNetworkGateways/rp-vnet/pointtositeconfiguration) and download the VPN Client Certificate to your local environment. You can extract the zip file anywhere you would like, but we put it under the "secrets" folder because that is where the ARO-RP secrets reside.
  1. For openvpn:
    1. Copy ./secrets/rp-vnet/OpenVPN/vpnconfig.json to ./secrets/vpn-rp-eastus.ovpn
    1. Copy the last two certificates ("P2S client certificate" and "P2S client certificate private key") from ./secrets/vpn-eastus.ovpn file to ./secrets/vpn-rp-eastus.ovpn. You can overwrite the placeholders for those certificates at the bottom in the ./secrets/vpn-rp-eastus.ovpn file.
    1. Execute openvpn secrets/vpn-rp-eastus.ovpn. You may need sudo depending on your environment.
  1. For azure vpn client:
    1. Click the 'import' button in the vpn list, you will be prompted with an "open file dialog".
    1. Select the file: ./secrets/rp-vnet/AzureVPN/azurevpnconfig.xml. The data will be filled into the import screen with the exception of "Client Certificate Public Key Data" and "Private Key".
    1. Copy the "P2S client certificate" into the "Client Certificate Public Key Data" field and "P2S client certificate private key" into the "Private Key" field.
    1. Click "Save" and you should see your newly created VPN connection in the VPN list on the left.
    1. Click the new VPN connection and click "Connect".
  1. Use nmap to execute the following command: nmap -p 443 -sT 10.x.x.x -Pn. You can get this IP at: https://ms.portal.azure.com/#blade/Microsoft_Azure_Compute/VirtualMachineInstancesMenuBlade/Networking/instanceId/subscriptions/<subscription-id>/resourceGroups/<aro-shared-rp-rg>/providers/Microsoft.Compute/virtualMachineScaleSets/dev-proxy-vmss/virtualMachines/0. Look for "NIC Private IP", ours during setup became 10.0.0.4. This is the internal ip of the Proxy VM.
    1. Confirm the nmap output looks like this: (if it does not then your VPN is not connected correctly; kill anything using port 443 and connect again)
    ```bash
    Starting Nmap 7.92 ( https://nmap.org ) at 2022-03-29 18:29 EDT
    Nmap scan report for 10.0.0.4
    Host is up (0.015s latency).

    PORT STATE SERVICE
    443/tcp open https

    Nmap done: 1 IP address (1 host up) scanned in 0.25 seconds
    ```
    1. Update the PROXY_HOSTNAME environment variable in ./secrets/env to point the IP you located above of for the Proxy VM.
  1. Now that your VPN is connected correctly and you've updated PROXY_HOSTNAME you need to source your env file for that update.
  ```bash
  . ./secrets/env
  ```
  1. Execute the local ARO-RP
  ```bash
  make runlocal-rp
  ```

# Steps to Create Cluster:

  1. Open another terminal (make sure you source your ./secrets/env file in this terminal as well)
  1. Execute this command to create a cluster
  ```bash
  CLUSTER=<aro-cluster-name> go run ./hack/cluster create
  ```

  This will take a while but eventually if the cluster is created you should see the following in your terminal indicating the cluster creation was successful:
  ```bash
  INFO[2022-04-01T10:02:41-05:00]pkg/util/cluster/cluster.go:318 cluster.(*Cluster).Create() creating cluster complete
  ```

# Steps to connect to the Cluster and confirm it is up via kubectl or oc:

1. At your terminal execute to create the admin.kubeconfig locally. This will allow you to connect to the cluster via kubectl or oc
   ```bash
   CLUSTER=<aro-cluster-name> make admin.kubeconfig
   ```

1. Disconnect from "rp-vnet" vpn and connect to "dev-vnet" vpn. The steps are identical to connecting to "rp-vnet" in the #preparation-to-create-cluster section. You will just need to download the dev-vpn client certificate locally, create the VPN connection using your VPN client of choice, and use nmap to get the IP of the internal load balancer from the <aro-cluster-name-rg>. You can find this address at: https://ms.portal.azure.com/#@microsoft.onmicrosoft.com/resource/subscriptions/<subscriptionid>/resourceGroups/<aro-cluster-rp-rg>/providers/Microsoft.Network/loadBalancers/<aro-cluster-name>-<random string for your lb>-internal/frontendIpPool (internal-lb-ip-v4). In my case the IP was 10.62.174.
```bash
nmap -p 6443 -sT 10.62.174.4 -Pn
Starting Nmap 7.92 ( https://nmap.org ) at 2022-04-01 10:36 CDT
Nmap scan report for 10.62.174.4
Host is up (0.070s latency).

PORT STATE SERVICE
6443/tcp open sun-sr-https

Nmap done: 1 IP address (1 host up) scanned in 0.14 seconds
```
1. Update admin.kubeconfig cluster.server parameter to use this IP as well. It should look like this:
```bash
server: https://<ip>:6443
```
1. Updated your kubeconfig env var to point to the admin.kubeconfig
```bash
export KUBECONFIG=$(pwd)/admin.kubeconfig
```
1. Execute a kubectl (or oc) command to see if you can list any K8s objects
```bash
kubectl get nodes --insecure-skip-tls-verify
```
2. You should see something like this. If so, your cluster is up!
```bash
NAME                                        STATUS   ROLES    AGE    VERSION
cdp-cfs-eleven-bljdk-master-0               Ready    master   3h7m   v1.22.3+4dd1b5a
cdp-cfs-eleven-bljdk-master-1               Ready    master   3h6m   v1.22.3+4dd1b5a
cdp-cfs-eleven-bljdk-master-2               Ready    master   3h6m   v1.22.3+4dd1b5a
cdp-cfs-eleven-bljdk-worker-eastus1-2r9b4   Ready    worker   177m   v1.22.3+4dd1b5a
cdp-cfs-eleven-bljdk-worker-eastus2-jgrj9   Ready    worker   177m   v1.22.3+4dd1b5a
cdp-cfs-eleven-bljdk-worker-eastus3-fd646   Ready    worker   177m   v1.22.3+4dd1b5a
```

*** There are some pods not coming up but we are going to investigate those soon.