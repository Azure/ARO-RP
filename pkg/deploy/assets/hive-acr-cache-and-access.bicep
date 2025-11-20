// Combined Bicep template for Hive ACR artifact cache and AKS access
// Deploys credential set, cache rules, and AKS role assignment together

@description('Name of the Azure Container Registry')
param acrName string

@description('Name of the AKS cluster to grant pull access')
param aksClusterName string = 'aro-aks-cluster-001'

@description('Username or client ID for Quay.io authentication')
@secure()
param hiveRegistryUsername string

@description('Password or client secret for Quay.io authentication')
@secure()
param hiveRegistryPassword string

@description('Source repository for Hive images')
param sourceRepository string = 'quay.io/redhat-services-prod/crt-redhat-acm-tenant/hive-operator/hive'

@description('Target repository name in ACR')
param targetRepository string = 'redhat-services-prod/crt-redhat-acm-tenant/hive-operator/hive'

var credentialSetName = 'hive-pull-credentials'
var cacheRuleName = 'hive-cache-rule'
var aksClusterId = resourceId('Microsoft.ContainerService/managedClusters', aksClusterName)
var acrResourceId = resourceId('Microsoft.ContainerRegistry/registries', acrName)
var acrPullRoleDefinitionId = subscriptionResourceId('Microsoft.Authorization/roleDefinitions', '7f951dda-4ed3-4680-a7ca-43fe172d538d')

resource acr 'Microsoft.ContainerRegistry/registries@2023-01-01-preview' existing = {
  name: acrName
}

resource aksCluster 'Microsoft.ContainerService/managedClusters@2023-01-01' existing = {
  name: aksClusterName
}

resource credentialSet 'Microsoft.ContainerRegistry/registries/credentialSets@2023-01-01-preview' = {
  parent: acr
  name: credentialSetName
  properties: {
    authCredentials: [
      {
        name: 'Credential1'
        usernameSecretIdentifier: hiveRegistryUsername
        passwordSecretIdentifier: hiveRegistryPassword
      }
    ]
    loginServer: 'quay.io'
  }
}

resource cacheRule 'Microsoft.ContainerRegistry/registries/cacheRules@2023-01-01-preview' = {
  parent: acr
  name: cacheRuleName
  properties: {
    sourceRepository: sourceRepository
    targetRepository: targetRepository
    credentialSetResourceId: credentialSet.id
  }
}

resource acrPullRoleAssignment 'Microsoft.Authorization/roleAssignments@2022-04-01' = {
  name: guid(aksClusterId, acrResourceId, acrPullRoleDefinitionId)
  scope: acr
  properties: {
    roleDefinitionId: acrPullRoleDefinitionId
    principalId: aksCluster.properties.identityProfile.kubeletidentity.objectId
    principalType: 'ServicePrincipal'
    description: 'Allows AKS cluster to pull Hive images from ACR'
  }
}

output credentialSetId string = credentialSet.id
output cacheRuleId string = cacheRule.id
output roleAssignmentId string = acrPullRoleAssignment.id

