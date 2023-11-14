# Feature
This project is part of the [migration tasks](https://dev.azure.com/msazure/AzureRedHatOpenShift/_wiki/wikis/AzureRedHatOpenShift.wiki/515448/Migration) to migrate the RP application from VMSS to the AKS Cluster and follow the [design](https://dev.azure.com/msazure/AzureRedHatOpenShift/_wiki/wikis/AzureRedHatOpenShift.wiki/567165/Mock-RP-Design).

# Pipelines (Build & Release)
- [RP Infrastructure - INT](https://dev.azure.com/msazure/AzureRedHatOpenShift/_build?definitionId=321081)

# Resources
- Deployment:
    - Set up the cluster and deploy the RP application.
- [Istio](https://dev.azure.com/msazure/AzureRedHatOpenShift/_wiki/wikis/AzureRedHatOpenShift.wiki/499024/Istio):
    - Follow [AKS Instruction](https://learn.microsoft.com/en-us/azure/aks/istio-about) to implement the Istio addon for the AKS cluster.
    - Use it to create an Istio service mesh and gateway for the RP application.
- [TLS Certificate](https://learn.microsoft.com/en-us/azure/aks/ingress-tls?tabs=azure-cli):
    - Use the [CSI driver](https://learn.microsoft.com/en-us/azure/aks/csi-secrets-store-driver) to synchronize the Key Vault certificate to AKS.
    - Create a dummy pod in the `aks-istio-ingress` namespace to synchronize the certificate using the CSI driver.
- [MISE](https://identitydivision.visualstudio.com/DevEx/_git/MISE?path=%2Fdocs%2FContainer.md&_a=preview):
    - Currently, follow the [sidecar pattern](https://dev.azure.com/msazure/AzureRedHatOpenShift/_wiki/wikis/AzureRedHatOpenShift.wiki/595474/MISE-istio-external-authorization-V.S.-side-car-pattern) to implement MISE.
    - To enable MISE authentication, edit the `values.yaml`'s `MISE_AUTH_ENABLED` attribute and `enableMISE` in `pkg>poc>miseAuthentication.go` to be either true or false.