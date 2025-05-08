# Upstream testing on ARO

OpenShift v4 uses a release payload image to configure the cluster.  Sometimes
we need to test a custom payloads before we submit a patch to upstream or
to confirm that a patch from upstream fixes something for ARO.
This section describes how to do this.

## Creating a custom payload

In this case we will be using an existing release payload as a base for testing.

1. Build the required operator component and host it in an external public
   repository.

1. Pull the release payload you want to use as a base to test your changes.
   `config.json` must contain an appropriate pull secret, e.g. one from
   `cloud.openshift.com`. An example:

   ```bash
   podman pull quay.io/openshift-release-dev/ocp-release-nightly@sha256:ab5022516a948e40190e4ce5729737780b96c96d2cf4d3fc665105b32d751d20 \
   --authfile=~/.docker/config.json
   ```

1. Extract required files:

   ```bash
   podman run  -it --entrypoint "/bin/bash" -v /tmp/cvo:/tmp/cvo:z --rm \
     quay.io/openshift-release-dev/ocp-release-nightly@sha256:ab5022516a948e40190e4ce5729737780b96c96d2cf4d3fc665105b32d751d20
   cat release-manifests/0000_50_cloud-credential-operator_05_deployment.yaml \
     >/tmp/cvo/0000_50_cloud-credential-operator_05_deployment.yaml
   cat release-manifests/image-references > /tmp/cvo/image-references
   ```

1. Apply changes you need and build an image using this example `Dockerfile`:

   ```bash
   FROM quay.io/openshift-release-dev/ocp-release-nightly@sha256:ab5022516a948e40190e4ce5729737780b96c96d2cf4d3fc665105b32d751d20
   ADD 0000_50_cloud-credential-operator_05_deployment.yaml /release-manifests/0000_50_cloud-credential-operator_05_deployment.yaml
   ADD image-references /release-manifests/image-references
   ```

1. Publish images to your personal repository:

   ```bash
   podman build -t quay.io/$USER/release-payload ./
   podman push quay.io/$USER/release-payload
   ```

1. Update pull spec in the code to use the new payload (see the `pkg/util/version` package).

1. Create a new cluster using your local RP.

## Modifying existing cluster

In some cases it is easier and faster to replace a component
on already existing cluster to test a patch.

In the example below we are going to apply a custom `kube-apiserver` image.

1. Build/copy the required component image and host it in an external public repository.
1. `oc login` into a cluster.
1. Get ssh to the node (either directly, if cluster is in the dev vnet, or via pod using `./hack/ssh-k8s.sh`).
1. Make cluster version operator stop overriding the deployment of `kube-apiserver-operator` so we can later scale it down:

    * Prepare a patch:

        ```shell
        cat > unmanaged-patch.yaml << EOF
        - op: add
          path: /spec/overrides
          value:
          - kind: Deployment
            group: ""
            name: kube-apiserver-operator
            namespace: openshift-kube-apiserver-operator
            unmanaged: true
        EOF
        ```

    * Apply the patch:

        ```shell
        oc patch clusterversion version --type json -p "$(cat unmanaged-patch.yaml)"
        ```

1. Scale down the `kube-apiserver-operator` to allow us to modify `kube-apiserver` later:

   ```shell
   oc -n openshift-kube-apiserver-operator scale deployment kube-apiserver-operator --replicas=0
   ```

1. Now we need to modify `kube-apiserver` pod. It is a [static pod](https://kubernetes.io/docs/tasks/configure-pod-container/static-pod/), so we need to have access to master nodes:

   * (Optional) Via SSH on each master: make a copy of `kube-apiserver` manifest `/etc/kubernetes/manifests/kube-apiserver-pod.yaml` and put it outside `/etc/kubernetes/`. This can be useful in case you decide to restore the state of the API server later.

   * Via SSH on each master: replace `kube-apiserver` image with the one you are testing. Manifest contains several images, and we need to modify only one with `kube-apiserver-NUMBER` (`kube-apiserver-7`, for example).

1. Verify that `kube-apiserver-NUMBER` containers from the list have a new image:

    ```
    oc -n openshift-kube-apiserver get pods -l app=openshift-kube-apiserver -o json | jq ".items[].spec.containers[] | {name: .name, image: .image}"
    ```
