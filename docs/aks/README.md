# AKS RP Documentation
This documentation is designed to give you an understanding of how the AKS deployments work, including things that you might consider simple but are taken for-granted.

Topics:

1. Pipelines
1. Images & Image Registry
1. Helm Charts
1. Helm Repositories

## Pipelines
The pipelines in-use for aks deployments are located in the global pipelines repository in the private repo "ARO-Pipelines".

## Images
Images are built in pipelines and deployed to the relevant Azure Container Registry (ACR). These images are usually associated with a `Dockerfile.<component>` located in the root of this repository. Building these containers is performed in a variety of ways, the easiest example of which is the `Dockerfile.proxy` which is built using `make image-proxy` and deployed using `make deploy-image-proxy`. Components like the monolith `rp` are more complicated as there are numerous enviornment variables that are required and many major sub-commands. These are usually covered by the documentation in the `/docs` folder which speak about deploying a development rp.

## Helm charts
Helm charts are located under the `/helm` directory and are built using pipelines themselves. The pipelines run a `make charts` and deployed using the `make deploy-charts`.

## Helm repositories
At current there's only one "repository" which is an OCI repository located in the default ACR for the enviornment for which the charts are being deployed to. E.g. if the current environment in the pipeline is set to have the `myacr` acr then the repository would be located at `oci://myacr.acr.io/helm`. Charts are deployed to a subdirectory of the repository, e.g. the `mimo` chart will be deployed to `oci://myacr.acr.io/helm/mimo`