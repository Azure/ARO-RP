# Hive

## Version

Replace the HIVE_IMAGE define with the latest version of the Hive image in the hack/hive-generate-config.sh file and update the HIVE_IMAGE_COMMIT_HASH define to the commit sha the image was built from. This ensures we use the correct config files for the version we are using.

## Generating config

In order to generate config for a dev environment you need to ensure you have the correct `LOCATION` set in your env file. Once this is done you can simply run the config generation script.

```bash
# source your environment file
. ./env
# run the config generation
./hack/hive-generate-config.sh
```

This will download the latest source, reset to the hash specified in HIVE_IMAGE_COMMIT_HASH and build the config using kustomise.

## Installing

Ensure you have the latest AKS kubefig:
```bash
# get the AKS kubeconfig
make aks.config
```

Set KUBECONFIG to the aks.kubeconfig file, for example:
```bash
export KUBECONFIG="$(pwd)/aks.kubeconfig"
```

Installing then simply requires the running of the install script.

```bash
./hack/hive-dev-install.sh
```
