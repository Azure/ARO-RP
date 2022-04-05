param clusterParentDomainName string
param databaseAccountName string
param fpServicePrincipalId string
param rpServicePrincipalId string
param location string = resourceGroup().location

resource dnsZone 'Microsoft.Network/dnsZones@2018-05-01' = {
  location: 'global'
  name: '${location}.${clusterParentDomainName}'
}

resource rpPevnetPeering 'Microsoft.Network/virtualNetworks/virtualNetworkPeerings@2021-05-01' = {
  name: 'rp-vnet/rp-pe-vnet-001-peering'
  properties: {
    allowVirtualNetworkAccess: true
    allowForwardedTraffic: true
    allowGatewayTransit: false
    useRemoteGateways: false
    remoteVirtualNetwork: {
      id: resourceId('Microsoft.Network/virtualNetworks', 'rp-pe-vnet-001')
    }
  }
}

resource peRpVnetPeering 'Microsoft.Network/virtualNetworks/virtualNetworkPeerings@2021-05-01' = {
  name: 'rp-pe-vnet-001/rp-vnet-peering'
  properties: {
    allowVirtualNetworkAccess: true
    allowForwardedTraffic: true
    allowGatewayTransit: false
    useRemoteGateways: false
    remoteVirtualNetwork: {
      id: resourceId('Microsoft.Network/virtualNetworks', 'rp-vnet')
    }
  }
}

resource documentDbAccount 'Microsoft.DocumentDB/databaseAccounts@2021-11-15-preview' = {
  name: databaseAccountName
  kind: 'GlobalDocumentDB'
  location: location
  properties: {
    consistencyPolicy: {
      defaultConsistencyLevel: 'Strong'
    }
    locations: [
      {
        locationName: location
      }
    ]
    databaseAccountOfferType: 'Standard'
    backupPolicy: {
      type: 'Periodic'
      periodicModeProperties: {
        backupIntervalInMinutes: 240
        backupRetentionIntervalInHours: 720
      }
    }
  }
  tags: {
    defaultExperience: 'Core (SQL)'
  }
}

resource nwContributorRoleDefinion 'Microsoft.Authorization/roleDefinitions@2018-01-01-preview' existing = {
  name: '4d97b98b-1d4f-4787-a291-c67834d212e7'
  scope: subscription()
}

resource docDbAccContributorRoleDefinition 'Microsoft.Authorization/roleDefinitions@2018-01-01-preview' existing = {
  scope: subscription()
  name: '5bd9cd88-fe45-4216-938b-f97437e15450'
}

resource docDbRoleAssignment 'Microsoft.Authorization/roleAssignments@2015-07-01' = {
  name: guid(resourceGroup().name, 'DocumentDB Account Contributor')
  properties: {
    // scope: resourceId('Microsoft.DocumentDB/databaseAccounts',databaseAccountName)
    roleDefinitionId: docDbAccContributorRoleDefinition.id
    principalId: rpServicePrincipalId
  }
  dependsOn: [
    documentDbAccount
  ]
}

resource dnsZoneContributorRoleDefinition 'Microsoft.Authorization/roleDefinitions@2018-01-01-preview' existing = {
  scope: subscription()
  name: 'befefa01-2a29-4197-83a8-272ff33ce314'
}

resource readerRoleDefinion 'Microsoft.Authorization/roleDefinitions@2018-01-01-preview' existing = {
  name: 'acdd72a7-3385-48ef-bd42-f606fba81ae7'
  scope: subscription()
}

resource fpDnsContributorRoleAssignment 'Microsoft.Authorization/roleAssignments@2015-07-01' = {
  name: guid(resourceGroup().name, 'FP DNS Zone Contributor')
  properties: {
    principalId: fpServicePrincipalId
    roleDefinitionId: dnsZoneContributorRoleDefinition.id  
  }
}

resource rpReaderRoleAssignment 'Microsoft.Authorization/roleAssignments@2015-07-01' = {
  name: guid(resourceGroup().name, 'RP / Reader')
  properties: {
    principalId: rpServicePrincipalId
    roleDefinitionId: readerRoleDefinion.id
  }
}

resource fpNwContributorRoleAssignment 'Microsoft.Authorization/roleAssignments@2015-07-01' = {
  name: guid(resourceGroup().name, 'FP / Network Contributor')
  properties: {
    principalId: fpServicePrincipalId
    roleDefinitionId: nwContributorRoleDefinion.id
  }
}
