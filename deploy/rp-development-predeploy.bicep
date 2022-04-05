param adminObjectId string
param fpServicePrincipalId string
param keyvaultPrefix string
param rpServicePrincipalId string
param location string = resourceGroup().location


resource rpNsg 'Microsoft.Network/networkSecurityGroups@2021-05-01' = {
  name: 'rp-nsg'
  location: location
  properties: {
    securityRules: [
      {
        name: 'rp_in_arm'
        properties: {
          protocol: 'Tcp'
          sourcePortRange: '*'
          destinationPortRange: '443'
          sourceAddressPrefix: '*'
          destinationAddressPrefix: '*'
          access: 'Allow'
          priority: 120
          direction: 'Inbound'
        }
      }
      {
        name: 'ssh_in'
        properties: {
          protocol: 'Tcp'
          sourcePortRange: '*'
          destinationPortRange: '22'
          sourceAddressPrefix: '*'
          destinationAddressPrefix: '*'
          access: 'Allow'
          priority: 125
          direction: 'Inbound'
        }
      } 
    ]
  }
}

resource rpPeNsg 'Microsoft.Network/networkSecurityGroups@2021-05-01' = {
  name: 'rp-pe-nsg'
  location: location
}

resource rpVnet 'Microsoft.Network/virtualNetworks@2021-05-01' = {
  name: 'rp-vnet'
  location: location
  properties: {
    addressSpace: {
      addressPrefixes: [
        '10.0.0.0/24'
      ]
    }
    subnets: [
      {
        name: 'rp-subnet'
        properties: {
          addressPrefixes: [
            '10.0.0.0/24'
          ]
          networkSecurityGroup: {
            id: rpNsg.id
          }
        }
      }
    ]
  }
}

resource rpPeVnet 'Microsoft.Network/virtualNetworks@2021-05-01' = {
  name: 'rp-pe-vnet-001'
  location: location
  properties: {
    addressSpace: {
      addressPrefixes: [
        '10.0.4.0/22'
      ]
    }
    subnets: [
      {
        name: 'rp-pe-subnet'
        properties: {
          addressPrefix: '10.0.4.0/22'
          networkSecurityGroup: {
            id: rpPeNsg.id
            location: location
          }
          serviceEndpoints: [
            {
              service: 'Microsoft.Storage'
              locations: [
                '*'
              ]
            }
          ]
          privateEndpointNetworkPolicies: 'Disabled'
        }
      }
    ]
  }
}

resource clsKeyVault 'Microsoft.KeyVault/vaults@2021-11-01-preview' = {
  location: location
  name: '${keyvaultPrefix}-dev-cls'
  properties: {
    tenantId: tenant().tenantId
    sku: {
      family: 'A'
      name: 'standard'
    }
    accessPolicies: [
      {
        tenantId: tenant().tenantId
        objectId: fpServicePrincipalId
        permissions: {
          secrets: [
            'get'
          ]
          certificates: [
            'create'
            'delete'
            'get'
            'update'
          ]
        }
      }
      {
        objectId: adminObjectId
        tenantId: tenant().tenantId
        permissions: {
          certificates: [
            'get'
            'list'
          ]
        }
      }
    ]
    enableSoftDelete: true
  }
}

resource dbtKeyVault 'Microsoft.KeyVault/vaults@2021-11-01-preview' = {
  location: location
  name: '${keyvaultPrefix}-dev-dbt'
  properties: {
    tenantId: tenant().tenantId
    sku: {
      family: 'A'
      name: 'standard'
    }
    accessPolicies: [
      {
        tenantId: tenant().tenantId
        objectId: rpServicePrincipalId
        permissions: {
          secrets: [
            'get'
          ]
        }
      }
      {
        tenantId: tenant().tenantId
        objectId: adminObjectId
        permissions: {
          secrets: [
            'set'
            'list'
          ]
          certificates: [
            'delete'
            'get'
            'import'
            'list'
          ]
        }
      }
    ]
    enableSoftDelete: true
  }
}

resource porKeyVault 'Microsoft.KeyVault/vaults@2021-11-01-preview' = {
  location: location
  name: '${keyvaultPrefix}-dev-por'
  properties: {
    tenantId: tenant().tenantId
    sku: {
      family: 'A'
      name: 'standard'
    }
    accessPolicies: [
      {
        tenantId: tenant().tenantId
        objectId: rpServicePrincipalId
        permissions: {
          secrets: [
            'get'
          ]
        }
      }
      {
        tenantId: tenant().tenantId
        objectId: adminObjectId
        permissions: {
          secrets: [
            'get'
            'list'
          ]
          certificates: [
            'delete'
            'get'
            'import'
            'list'
          ]
        }
      }
    ]
    enableSoftDelete: true
  }
}

resource svcKeyVault 'Microsoft.KeyVault/vaults@2021-11-01-preview' = {
  name: '${keyvaultPrefix}-dev-svc'
  location: location
  properties: {
    tenantId: tenant().tenantId
    sku: {
      family: 'A'
      name: 'standard'
    }
    accessPolicies: [
      {
        tenantId: tenant().tenantId
        objectId: rpServicePrincipalId
        permissions: {
          secrets: [
            'get'
            'list'
          ]
        }
      }
      {
        tenantId: tenant().tenantId
        objectId: adminObjectId
        permissions: {
          secrets: [
            'get'
            'set'
            'list'
          ]
          certificates: [
            'delete'
            'get'
            'import'
            'list'
          ]
        }
      }
    ]
    enableSoftDelete: true
  }
}
