// Credential Set for Artifact Cache
// Stores credentials needed to pull from the new Hive repository

@description('Name of the Azure Container Registry')
param acrName string

@description('Name for the credential set')
param credentialSetName string = 'hive-pull-credentials'

@description('Username or client ID for authentication')
@secure()
param username string

@description('Password or client secret for authentication')
@secure()
param password string

resource acr 'Microsoft.ContainerRegistry/registries@2023-01-01-preview' existing = {
  name: acrName
}

resource credentialSet 'Microsoft.ContainerRegistry/registries/credentialSets@2023-01-01-preview' = {
  parent: acr
  name: credentialSetName
  properties: {
    authCredentials: [
      {
        name: 'Credential1'
        usernameSecretIdentifier: username
        passwordSecretIdentifier: password
      }
    ]
    loginServer: 'quay.io'
  }
}

output credentialSetResourceId string = credentialSet.id
output credentialSetName string = credentialSet.name

