# ARO Builder Image

## Overview
The ARO builder image is an image hosted in our Azure Container Registry. It is used as a base image in which we build and ship our container images such as the RP, the ARO operator, and the ARO proxy.  It contains the golang toolset and other dependencies necessary to build the source code.

## Builder Image Tags
The ARO builder image is tagged with the golang version that it ships with.  A tag of 1.14 would correspond to the golang 1.14 version.
>__NOTE:__ There may not exist a container of the latest golang version as it may not be built yet.  See [Updating the Builder Image](#updating-or-building-the-builder-image) for more details.

## Using the Builder Image
In order to use the builder image, you must first authenticate with the Azure Container Registry.  You can use `docker` or `podman` to login to arointsvc.azurecr.io with the pull secret present in secrets.

From there you can build any of the images which require the builder.

## Updating or Building the Builder Image
The builder image is based off of a RHEL7 image.  In RHEL7 the gpgme-devel package is not available to install unless you entitle the system or are running an entitled build.  If you are on Fedora or RHEL, you can use the instructions in [Entitle your Fedora Box](#entitle-your-fedora-box) to entitle your system and utilize subscription based RHEL packages.

If you are not on a Fedora system or don't want to entitle your system, you can utilize the CI VM Scale Set pool which is entitled to build the builder image.  Currently the CI pipeline builds the builder image in the E2E pipeline.  It then pushes it to the appropriate Azure Registry.
> __TODO:__ The image should only build on manual trigger, not on every single CI run.  It should be moved into its own pipeline which is manually triggered by a team member in the future.

### Entitle your Fedora Box
General instructions to entitle your Fedora system can be found [here](https://patrick.uiterwijk.org/blog/2016/10/6/rhel-containers-on-non-rhel-hosts).  After completing these instructions, you should be able to build the builder image locally.
