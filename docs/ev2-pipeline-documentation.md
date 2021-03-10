# Ev2 Pipeline Documentation

This is a living document that will contain definitions of Ev2 structures as well as a list of parameters for each structure in the Ev2 release pipeline for the ARO-RP. This should serve as a single point of entry for anyone wanting to further develop the Ev2 pipelines.

## Concerns

1. Rollout Strategy
   1. Sectors (Stages in ADO)
      1. **Being defined by CF and RH**
2. Manifest Generation
   1. Tooling - PowerShell? Etc..
   2. Base Templates
      1. Rollout Spec
      2. Service Model
      3. Scope Tags / Scope Bindings

## Glossary

1. Rollout Spec: The rollout is defined at the "root" of the Ev2 deployment. This is simply a folder that holds json files containing deployment details. At it's core the Rollout points to a ServiceModel (see the "ServiceModel" section below) that determines the resources that will be deployed.
   1. Reference: <https://ev2docs.azure.net/getting-started/authoring/rollout-spec/rolloutspec.html>
   2. Rollout Spec Common Parameters
      1. ContentVersion: This is the version of the Rollout Spec. It can, but does not have to, correspond to the version of the ServiceModel/Build being deployed.
      2. RolloutMetadata.ServiceModelPath: Path to the ServiceModel.json file. This is a filename that could be specified different if multiple service models were being maintained. In our case for the RP one ServiceModel should be sufficient, but this is worth noting for future consideration.
      3. RolloutMetadata.Name: Name of the service being deployed. Typically it is something like "{ServiceName} {BuildNumber}" but it can be anything descriptive.
      4. RolloutMetadata.RolloutType: RolloutType is based on the build being deployed. Values can be Major, Minor, or Hotfix.
      5. OrchestratedSteps: A series of steps that specifies the order of items to deploy in the ServiceModel.
         1. For ARO-RP this will point to the RP application in the ServiceResourceGroups.ServiceResources array.
2. ServiceModel: The ServiceModel specifies the resources that will be deployed across any specified regions.
   1. For the ARO-RP pipeline we will need to deploy to regions based on sectors defined by the team. The implementation of the Ev2 configuration will be generated using a PowerShell script. It will give us the ability to have configuration files generated for each individual region as well as master files that can be utilized to deploy to all regions at once. This will be further defined as the implementation is copmleted.
   2. ServiceModel Common Parameters:
      1. Environment: Test, Staging, Prod, etc. This can be customized as needed.
      2. ServiceResourceGroupDefinitions: A "serviceResourceGroup" points to a specific Azure Resource Group. The definition here allows for applying a shared set of templates, parameters, and other compositional pieces (such as extensions) to a set of Azure Resource Groups.
      3. ServiceResourceGroup: Specifies the exact Azure Resource Group name to deploy the ARM templates specified in the "serviceResourceGroupDefinition" being referenced. This is the level at which Azure SubscriptionId and Region/Location are specified. Multiple serviceResourceGroups could be a powerful tool to deploy to different clouds.
      4. ServiceResources: Specifies the applications to publish. For ARO-RP this will be the RP application itself.
      5. scopeTags/scopeBindings: Allows for sharing values (such as location or a url) across configuration files. This allows for getting rid of hard-coded values that require multiple search/replace to update.
   3. ServiceModel Parameters for ARO-RP (these will be further defined going forward):
      1. ServiceMetadata.Environment: Specified by the pipeline when triggering a release. RP-INT or RP-PROD
      2. ServiceResourceGroup.AzureResourceGroupName: Specified by the pipeline when triggering a release. Could this also be a convention?
      3. ServiceResourceGroup.InstanceOf: Points to the ServiceResourceGroupDefinition of which to derive from.
      4. ServiceResourceGroup.ScopeTags: This will be specified at the RG level based on the parameters below.
      5. For the ARO-RP deployment parameters must match up with what is in the ARM templates in order for Ev2 to function properly:
         1. Parameters for ARO-RP and Deployment:
            1. Service Tree Id (GUID)
            2. AZURE_ENVIRONMENT: This must be set prior to installing the ARO-RP otherwise it will default to "AzureCloud" (public).
            3. AcrResourceId: The Ev2 deployment will pull this from Az KV via the Ev2 http extension.
            4. ServicePrincipal: The ARO-RP template takes these as a chunk of json ("azureDevOpsJSONSPN") with the following properties: AZURE_CLIENT_ID, AZURE_CLIENT_SECRET, AZURE_TENANT_ID. The Ev2 deployment will specify this json to the ARO-RP, but the properties will be stored individually and bound in from Az KV via the Ev2 http extension.
            5. StorageAccount: The Ev2 deployment will pull this from Az KV via the Ev2 http extension.
            6. SubscriptionId: This is specified in the Ev2 deployment's ServiceResourceGroup. For purposes for the ARO-RP pipeline this will supplied to the template via this parameters file.
            7. rpMode: This can be "development", "int", or in the case of production be left blank.
      6. ServiceResourceGroup.SubscriptionId: Best practice shows this hard-coded at this level. For a second entry this would require another ServiceResourceGroup item in the ServiceResourceGroups array.
      7. ServiceResourceGroup.Location: Best practice shows this hard-coded at this level. For a second entry this would require another ServiceResourceGroup item in the ServiceResourceGroups array.

## Items to address during implementation

1. Where params will be stored (Azure KeyVault, etc).
2. Monitor Jim Winter's current PR -> <https://github.com/Azure/ARO-RP/pull/1368>
