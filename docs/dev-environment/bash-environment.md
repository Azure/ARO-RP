# Bash Environment Helpers
- [Bash Environment Helpers](#bash-environment-helpers)
  - [ARO-RP Bash Environment Helpers](#aro-rp-bash-environment-helpers)
  - [Helper Aliases and Functions](#helper-aliases-and-functions)
    - [Aliases](#aliases)
    - [Functions](#functions)
  - [Bash rc config Drop-Ins](#bash-rc-config-drop-ins)
    - [bashrc.d Setup](#bashrcd-setup)
    - [bashrc Drop-Ins Testing](#bashrc-drop-ins-testing)

## ARO-RP Bash Environment Helpers

1. Configure `kubectl`/`oc` to use your cluster's admin kubeconfig
    ```sh
    export ARO_RP="$HOME/go/src/github.com/Azure/ARO-RP"
    export KUBECONFIG="$ARO_RP/admin.kubeconfig"
    ```

## Helper Aliases and Functions

### Aliases

These are useful aliases you may wish to configure.

```bash
alias ll='ls -lhtr --color=auto'
```

### Functions

1. `rp-admin-operator-update()`
    ```sh
    echo '# rp-admin-operator-update()
    rp-admin-operator-update() {
        err_unset="Have you sourced your env file?"
        curl -X PATCH \
            -k \"https://localhost:8443/subscriptions/${AZURE_SUBSCRIPTION_ID:?$err_unset}/resourceGroups/${RESOURCEGROUP:?$err_unset}/providers/Microsoft.RedHatOpenShift/openShiftClusters/${CLUSTER:?$err_unset}?api-version=admin\" \
            --header \"Content-Type: application/json\" \
            -d "{\"properties\": {\"maintenanceTask\": \"OperatorUpdate\"}}"
    }' > ~/.bashrc.d/rp-admin-operator-update
    ```
2. `rp-admin-update()`
    ```sh
    echo '# rp-admin-update()
    # args:
    #   *) Used for curl patch contents
    rp-admin-update() {
        err_unset="Have you sourced your env file?"

        curl -X PATCH \
            -k "https://localhost:8443/subscriptions/${AZURE_SUBSCRIPTION_ID:?$err_unset}/resourceGroups/${RESOURCEGROUP:?$err_unset}/providers/Microsoft.RedHatOpenShift/openShiftClusters/${CLUSTER:?$err_unset}?api-version=admin" \
            --header "Content-Type: application/json" \
            -d "${*}"
    }' > ~/.bashrc.d/rp-admin-update
    ```
3. `rp-recovery-etcd()`
    ```sh
    echo '# rp-recovery-etcd()
    rp-recovery-etcd() {
        err_unset="Have you sourced your env file?"

        curl -X POST \
            -k "https://localhost:8443/admin/subscriptions/${AZURE_SUBSCRIPTION_ID:?$err_unset}/resourceGroups/${RESOURCEGROUP:?$err_unset}/providers/Microsoft.RedHatOpenShift/openShiftClusters/${CLUSTER:?$err_unset}/etcdrecovery?api-version=admin&namespace=openshift-etcd&name=cluster&kind=Etcd" \
            --header "Content-Type: application/json" \
            -d "{}"
    }' > ~/.bashrc.d/rp-recovery-etcd
    ```
* run `yq` as a container
    ```sh
        echo 'yq() {
        # log_level="${PODMAN_LOG_LEVEL:-warn}"
        # 
        # Useful for debugging podman runtime errors
        log_level="${PODMAN_LOG_LEVEL:-warn}"

        image="quay.io/konflux-ci/yq:latest"

        # YQ_* environment variables are useful for passing variables to be parsed by yq
        podman run \
            --userns=keep-id \
            --env '\''YQ_*'\'' \
            --log-level="$log_level" \
            --security-opt label=disable \
            -i \
            --rm \
            -v=${PWD}:/workdir:rw,rshared \
	    --workdir /workdir \
            "$image" \
                --exit-status=1 \
                "$@"
    }

    # yq must be exported for scripts (or any subshell) to access it
    export -f yq
    ' > ~/.bashrc.d/yq
    ```

## Bash rc config Drop-Ins

### bashrc.d Setup

`bash` can be configured to source drop-in configurations.  
This allows you to easily add new configurations without modifying your primary `~/.bashrc`

1. Backup `~/.bashrc`
    ```sh
    cp ~/.bashrc ~/.bashrc~
    ```
2. Append `~/.bashrc.d` drop-in configurations to `~/.bashrc`
    ```sh
    # User specific aliases and functions
    if [ -d ~/.bashrc.d ]; then
	    for rc in ~/.bashrc.d/*; do
		    if [ -f "$rc" ]; then
			    . "$rc"
		    fi
	    done
        unset rc
    fi
    ```
3. Create `~/.bashrc.d`
    ```sh
    mkdir ~/.bashrc.d
    ```

### bashrc Drop-Ins Testing

Test it out!

1. Set your prefered shell editor
    ```sh
    echo 'export EDITOR="$(which vim)"' > ~/.bashrc.d/editor
    ```
2. Re-Source `~/.bashrc`
    ```sh
    . ~/.bashrc
    ```
3. Verify `EDITOR`
    ```sh
    user:~/go/src/github.com/Azure/ARO-RP$ echo $EDITOR
    /usr/bin/vim
    ```
