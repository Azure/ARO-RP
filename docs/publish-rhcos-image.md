# Publish RHCOS image

Each release we need to re-publish the RHCOS image into the Azure cloud partner portal.

1. Once new installer is vendored run `make vendor` to update local vendor directory.

1. Run `make generate` to update generated content

1. Run `go run ./hack/rhcos-sas/rhcos-sas.go` to copy RHCOS image. This might take a while.

1. Command above will output SAS URL. Publish it via [partner](https://partner.microsoft.com/) portal.
See the [ARO Wiki in Azure DevOps](https://msazure.visualstudio.com/AzureRedHatOpenShift/_wiki/wikis/ARO.wiki/186149/How-To-Publish-New-OCP-Image-for-ARO) for detailed steps.
