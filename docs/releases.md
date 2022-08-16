# Releases

The ARO is released with the help of annotated `git tag -a`. The tag has the
following form

```
vYYYYMMDD.nn
```

where `nn` is the release number of the day and annotation contains handpicked
release summary, which can use markdown notation. Example git command would look like:

```
git tag -a v20220223.01
```

## Prerequisite

Before the release can be created, the ARO configuration in the [ADO repository](https://msazure.visualstudio.com/AzureRedHatOpenShift/_git/RP-Config)
have to be tagged with the same tag as the ARO release. Make sure,
you are tagging the right commit with the changes you want to release.


## Release pipeline

Currently the release is done manually via the [EV2 pipelines](https://msazure.visualstudio.com/AzureRedHatOpenShift/_wiki/wikis/ARO.wiki/233405/Performing-Release).


### Release page

The generate release notes pipeline does not have any parameter, instead the pipeline is started on the `tag` as illustrated on the image.

![Start pipelines with tag](img/pipelines.png "Aro Monitor Architecture")

Once the release notes pipeline is finished, a new item is added to the [GitHub release page](https://github.com/Azure/ARO-RP/releases) with
the following format:

```
# ${{TAG}}

${{tag annotation}}

## Changes:

- hash commit message
- hash commit message

```

The title of the release is the used `git tag`. The description is extracted
from the tag annotation.


### Image distribution

Moreover, when the release is built, the tagged ARO image is pushed to the
following registries:

- `arosvc`: production
- `arointsvc`: mirrored for integration testing