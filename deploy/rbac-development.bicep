param fpRoleDefinionName string = 'ARO v4 Development FirstParty Subscription'
param fpServicePrincipalId string
param armServicePrincipalId string
param toolingServicePrincipalId string

targetScope = 'subscription'

resource fpRoleDefinion 'Microsoft.Authorization/roleDefinitions@2015-07-01' = {
  name: guid(subscription().id,fpRoleDefinionName)
  properties: {
    assignableScopes: [
      subscription().id
    ]
    roleName: fpRoleDefinionName
    permissions: [
      {
        actions: [
          'Microsoft.Resources/subscriptions/resourceGroups/write'
        ] 
      }
    ]
  }
} 

resource fpRoleAssignment 'Microsoft.Authorization/roleAssignments@2015-07-01' = {
  name: guid(subscription().id, fpRoleDefinionName)
  properties: {
    principalId: fpServicePrincipalId
    roleDefinitionId: fpRoleDefinion.id
  }
}

resource armRoleAssignment 'Microsoft.Authorization/roleAssignments@2015-07-01' = {
  name: guid(subscription().id, 'ARM / User Access Administrator')
  properties: {
    principalId: armServicePrincipalId
    roleDefinitionId: userAccessAdminRoleDefinion.id
  }
}

resource toolingContributorRoleAssignment 'Microsoft.Authorization/roleAssignments@2015-07-01' = {
  name: guid(subscription().id, 'Tooling / Contributor')
  properties: {
    principalId: toolingServicePrincipalId
    roleDefinitionId: contributorRoleDefinion.id
  }
}

resource toolingUAARoleAssignment 'Microsoft.Authorization/roleAssignments@2015-07-01' = {
  name: guid(subscription().id, 'Tooling / User Access Administrator')
  properties: {
    principalId: toolingServicePrincipalId
    roleDefinitionId: userAccessAdminRoleDefinion.id
  }
}

resource userAccessAdminRoleDefinion 'Microsoft.Authorization/roleDefinitions@2018-01-01-preview' existing = {
  scope: subscription()
  name:  '18d7d88d-d35e-4fb5-a5c3-7773c20a72d9'
}

resource contributorRoleDefinion 'Microsoft.Authorization/roleDefinitions@2018-01-01-preview' existing = {
  scope: subscription()
  name: 'b24988ac-6180-42a0-ab88-20f7382dd24c'
}
