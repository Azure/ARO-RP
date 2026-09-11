// Main Bicep file for shared RP development environment orchestration
// This automates the deployment steps previously done manually via deploy-shared-env.sh

targetScope = 'resourceGroup'

// ============================================================================
// Parameters
// ============================================================================

@description('Location for all resources')
param location string = resourceGroup().location

@description('Admin object ID for Key Vault access')
param adminObjectId string

@description('First Party service principal ID')
param fpServicePrincipalId string

@description('RP service principal ID')
param rpServicePrincipalId string

@description('Key Vault prefix for naming')
param keyvaultPrefix string

@description('Cluster parent domain name')
param clusterParentDomainName string

@description('Database account name for CosmosDB')
param databaseAccountName string

@description('DNS domain name for the location')
param domainName string

@description('Proxy certificate (base64 encoded)')
@secure()
param proxyCert string

@description('Proxy client certificate (base64 encoded)')
@secure()
param proxyClientCert string

@description('Proxy domain name label')
param proxyDomainNameLabel string

@description('Proxy container image')
param proxyImage string = 'arointsvc.azurecr.io/proxy:latest'

@description('Proxy image authentication (base64 encoded)')
@secure()
param proxyImageAuth string

@description('Proxy key (base64 encoded)')
@secure()
param proxyKey string

@description('SSH public key for proxy')
param sshPublicKey string

@description('VPN CA certificate (base64 encoded)')
@secure()
param vpnCACertificate string

@description('OIDC storage account name')
param oidcStorageAccountName string

@description('Global DevOps service principal ID (optional)')
param globalDevopsServicePrincipalId string = ''

@description('Deploy AKS development resources')
param deployAks bool = true

@description('Deploy MIWI infrastructure')
param deployMiwi bool = true

@description('Use Basic SKU for Public IP (workaround for VPN gateway issues)')
param useBasicPublicIp bool = false

// ============================================================================
// Module 1: RP Development Predeploy
// ============================================================================

module rpDevPredeploy '../../../pkg/deploy/assets/rp-development-predeploy.json' = {
  name: 'rp-development-predeploy'
  params: {
    adminObjectId: adminObjectId
    fpServicePrincipalId: fpServicePrincipalId
    keyvaultPrefix: keyvaultPrefix
    rpServicePrincipalId: rpServicePrincipalId
  }
}

// ============================================================================
// Module 2: RP Development (Main Infrastructure)
// ============================================================================

module rpDev '../../../pkg/deploy/assets/rp-development.json' = {
  name: 'rp-development'
  params: {
    clusterParentDomainName: clusterParentDomainName
    databaseAccountName: databaseAccountName
    fpServicePrincipalId: fpServicePrincipalId
    rpServicePrincipalId: rpServicePrincipalId
    globalDevopsServicePrincipalId: globalDevopsServicePrincipalId
  }
  dependsOn: [
    rpDevPredeploy
  ]
}

// ============================================================================
// Module 3: RP Managed Identity
// ============================================================================

module rpManagedIdentity '../../../pkg/deploy/assets/rp-production-managed-identity.json' = {
  name: 'rp-managed-identity'
  params: {}
}

// ============================================================================
// Module 4: Environment Development (Proxy and VPN)
// ============================================================================

module envDev '../../../pkg/deploy/assets/env-development.json' = {
  name: 'env-development'
  params: {
    proxyCert: proxyCert
    proxyClientCert: proxyClientCert
    proxyDomainNameLabel: proxyDomainNameLabel
    proxyImage: proxyImage
    proxyImageAuth: proxyImageAuth
    proxyKey: proxyKey
    sshPublicKey: sshPublicKey
    vpnCACertificate: vpnCACertificate
    // Conditional parameters for Public IP SKU workaround
    publicIPAddressSkuName: useBasicPublicIp ? 'Basic' : 'Standard'
    publicIPAddressAllocationMethod: useBasicPublicIp ? 'Dynamic' : 'Static'
  }
  dependsOn: [
    rpDev
  ]
}

// ============================================================================
// Module 5: AKS Development (Optional)
// ============================================================================

module aksDev '../../../pkg/deploy/assets/aks-development.json' = if (deployAks) {
  name: 'aks-development'
  params: {
    dnsZone: domainName
    keyvaultPrefix: keyvaultPrefix
    sshRSAPublicKey: sshPublicKey
  }
  dependsOn: [
    rpDev
  ]
}

// ============================================================================
// Module 6: MIWI Infrastructure (Optional)
// ============================================================================

module miwiInfra '../../../pkg/deploy/assets/rp-development-miwi.json' = if (deployMiwi) {
  name: 'rp-development-miwi'
  params: {
    rpServicePrincipalId: rpServicePrincipalId
    oidcStorageAccountName: oidcStorageAccountName
  }
  dependsOn: [
    rpDev
  ]
}

// ============================================================================
// Post-Deployment: Enable Static Website on OIDC Storage Account
// ============================================================================

// Note: Bicep cannot directly enable static website feature via deploymentScripts.
// This must be done via the wrapper script using:
// az storage blob service-properties update --static-website true --account-name ${oidcStorageAccountName} --auth-mode login
// The OIDC storage account is created by the rp-development-miwi.json template.

// ============================================================================
// Outputs
// ============================================================================

@description('Resource group name')
output resourceGroupName string = resourceGroup().name

@description('Location')
output location string = location

@description('Key Vault service name')
output keyvaultSvcName string = '${keyvaultPrefix}-svc'

@description('Key Vault portal name')
output keyvaultPorName string = '${keyvaultPrefix}-por'

@description('Key Vault gateway name (if exists)')
output keyvaultGwyName string = '${keyvaultPrefix}-gwy'

@description('Database account name')
output databaseAccountName string = databaseAccountName

@description('Domain name')
output domainName string = domainName

@description('OIDC storage account name (if deployed)')
output oidcStorageAccountName string = deployMiwi ? oidcStorageAccountName : ''

@description('VPN gateway name for client configuration')
output vpnGatewayName string = 'dev-vpn'

@description('Deployment completed successfully')
output deploymentStatus string = 'Shared environment infrastructure deployed successfully'
