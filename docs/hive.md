# Hive

## Version

The commit sha is used to specify the image tag and also used during config generation to checkout the correct version of the config files. The config files are subsequently used by the `hack/hive/hive-dev-install.sh` script during installation or during config updates.

1. You can either
   1. Provide the hive image commit has as an argument to `hack/hive/hive-generate-config.sh`. This is useful for testing new hive images before hive releases.
      1. Example: `./hack/hive/hive-generate-config.sh d7ead609f4`
   2. Accept the default version by providing no arguments, which should be the latest.
      1. Example: `./hack/hive/hive-generate-config.sh`

## Generating config

In order to generate config for a dev environment you need to ensure you have the correct `LOCATION` is set in your env file. Once this is done you can simply run the config generation script.

```bash
# source your environment file
. ./env
# run the config generation
./hack/hive/hive-generate-config.sh
```

This will download the latest source, reset to the hash specified in HIVE_IMAGE_COMMIT_HASH, and build the config using kustomise.

## Installing

1. Connect to the appropriate aks vpn
   1. vpn-aks-westeurope.ovpn
   2. vpn-aks-eastus.ovpn
   3. vpn-aks-australiaeast.ovpn
2. Ensure you have the latest AKS kubeconfig  
    ```bash
    # get the AKS kubeconfig
    . ./env
    make aks.kubeconfig
    ```
3. Set KUBECONFIG to the aks.kubeconfig file, for example:
    ```bash
    export KUBECONFIG="$PWD/aks.kubeconfig"
    ```
4. Installing then simply requires the running of the install script.
    ```bash
    ./hack/hive/hive-dev-install.sh
    ```
   > __NOTE:__  When Hive is already installed and SKIP_DEPLOYMENTS is set to "true" then Hive installation can be skipped without user's approval.
