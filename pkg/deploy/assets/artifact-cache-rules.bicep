// Artifact Cache Rules for Hive Images
// Based on https://msazure.visualstudio.com/AzureRedHatOpenShift/_git/sdp-pipelines?path=/classic/global/infra/Templates/artifact-cache.bicep

@description('Name of the Azure Container Registry')
param acrName string

@description('Source repository for Hive images')
param sourceRepository string = 'quay.io/redhat-services-prod/crt-redhat-acm-tenant/hive-operator/hive'

@description('Target repository name in ACR')
param targetRepository string = 'redhat-services-prod/crt-redhat-acm-tenant/hive-operator/hive'

@description('Credential set resource ID for pull authentication')
param credentialSetResourceId string

resource acr 'Microsoft.ContainerRegistry/registries@2023-01-01-preview' existing = {
  name: acrName
}

resource cacheRule 'Microsoft.ContainerRegistry/registries/cacheRules@2023-01-01-preview' = {
  parent: acr
  name: 'hive-cache-rule'
  properties: {
    sourceRepository: sourceRepository
    targetRepository: targetRepository
    credentialSetResourceId: credentialSetResourceId
  }
}

output cacheRuleName string = cacheRule.name
output cacheRuleId string = cacheRule.id

