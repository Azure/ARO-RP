# Hive

## Version

Update the HIVE_IMAGE_COMMIT_HASH in `hack/hive-generate-config.sh` with the latest commit sha of the Hive image you are deploying. The commit sha is used to specify the image tag and also used during config generation to checkout the correct version of the config files. The config files are subsequently used by the `hack/hive-dev-install.sh` script during iunstallation or during config updates.

## Generating config

In order to generate config for a dev environment you need to ensure you have the correct `LOCATION` is set in your env file. Once this is done you can simply run the config generation script.

```bash
# source your environment file
. ./env
# run the config generation
./hack/hive-generate-config.sh
```

This will download the latest source, reset to the hash specified in HIVE_IMAGE_COMMIT_HASH, and build the config using kustomise.

## Installing

Ensure you have the latest AKS kubeconfig:
```bash
# get the AKS kubeconfig
make aks.kubeconfig
```

Set KUBECONFIG to the aks.kubeconfig file, for example:
```bash
export KUBECONFIG="$PWD/aks.kubeconfig"
```

Installing then simply requires the running of the install script.

```bash
./hack/hive-dev-install.sh
```
