param workloadMsiName string

resource workloadMsi 'Microsoft.ManagedIdentity/userAssignedIdentities@2024-11-30' existing = {
  name: workloadMsiName //using the workloadMsiName param passed from mimo.msi.bicepparam file
}

output workloadMsiObjectId string = worklodMsi.properties.principalId
