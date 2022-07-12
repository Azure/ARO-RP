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


## Testing own hive build

1. Create a fork of `github.com/openshift/hive` on github
    1. Make your changes and push into your fork.
1. Build and push a custom Hive image:
    ```bash
    # From a hive repo checkout with your changes
    export IMG="quay.io/{username}/hive:latest"
    GOOS=linux GOARCH=amd64 make build image-hive-fedora-dev docker-push
    ```
1. Make sure that your image is public so AKS can pull it
1. Set environment variables like shown in the example below:
    ```bash
    export HIVE_REPO=https://github.com/m1kola/hive.git # Point to your fork
    export HIVE_IMAGE_COMMIT_HASH=c63c9b0               # Commit hash from your fork
    export HIVE_IMAGE=$IMG                              # Point to your image as previously set in $IMG
    ```
1. Run the following commands:
    ```bash
    source ./env

    make aks.kubeconfig
    export KUBECONFIG="$PWD/aks.kubeconfig"

    ./hack/hive-generate-config.sh
    ./hack/hive-dev-install.sh
    ```
1. Restart pods to make sure that hive is running the latest version:
    ```bash
    oc delete pods -n hive --all
    ```
