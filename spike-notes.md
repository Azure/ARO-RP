# 3rd Party Multi-Tenant Application
* The goal of this is to create a third party multi-tenant application in tenant A, have another tenant register to the application in tenant B, and then restrict API calls from the application such that it can provision and has access into tenant B, but tenant B is unable to perform any of these actions themselves - it must be performed by the application owned by tenant A.

## Procedure

1. Create A new multi-tenant Application

    * docs: https://docs.microsoft.com/en-us/cli/azure/ad/app?view=azure-cli-latest

    * example: 
    ```
    az ad app create --display-name "robryan-testing" --identifier-uri "http://localhost:8443" --sign-in-audience AzureADMultipleOrgs --app-roles @manifest.json
    ```

    * get our appID
    ```
    az ad app list --display-name robryan-testing | jq'.[].id' | tr -d '"'
    ```
    * this will be our allowed clientID


# Restricted API Calls

* Used devops spn as clientID (e2e service princpal) for testing restriction

## Procedure

1. Modified openshiftclusters_get requests to check clientID
1. If clientID != e2e service principal clientID, return 403
1. Deploy to INT

## Results

```
➜  ARO-RP git:(ocm-3psp-spike) ✗ az aro show -g aro-test-eastus --name mytestcluster -o table --query name
Result
-------------
mytestcluster
➜  ARO-RP git:(ocm-3psp-spike) ✗ az account show --query user                                             
{
  "name": "abc...123",
  "type": "servicePrincipal"
}
3:58
[cloud-user@robryan-vm ~]$ az aro show -g aro-test-eastus --name mytestcluster -o table --query name
(Forbidden) The client is not authorized to access the resource.
Code: Forbidden
Message: The client is not authorized to access the resource.
[cloud-user@robryan-vm ~]$ az account show --query user
{
  "name": "b-rossbryan@microsoft.com",
  "type": "user"
}
```