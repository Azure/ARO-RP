# Custom payload testing

OpenShift v4 uses a release payload image to configure the cluster.  Sometimes
we need to test custom payloads before we submit a patch to upstream.  This
section describes how to do this.

1. Build the required operator component and host it in an external public
   repository.

1. Pull the release payload you want to use as a base to test your changes.
   `config.json` must contain an appropriate pull secret, e.g. one from
   `cloud.redhat.com`. An example:

   ```
   podman pull quay.io/openshift-release-dev/ocp-release-nightly@sha256:ab5022516a948e40190e4ce5729737780b96c96d2cf4d3fc665105b32d751d20 \
   --authfile=~/.docker/config.json
   ```

1. Extract required files:

   ```
   podman run  -it --entrypoint "/bin/bash" -v /tmp/cvo:/tmp/cvo:z --rm \
     quay.io/openshift-release-dev/ocp-release-nightly@sha256:ab5022516a948e40190e4ce5729737780b96c96d2cf4d3fc665105b32d751d20
   cat release-manifests/0000_50_cloud-credential-operator_05_deployment.yaml \
     >/tmp/cvo/0000_50_cloud-credential-operator_05_deployment.yaml
   cat release-manifests/image-references > /tmp/cvo/image-references
   ```

1. Apply changes you need and build an image using this example `Dockerfile`:

   ```
   FROM quay.io/openshift-release-dev/ocp-release-nightly@sha256:ab5022516a948e40190e4ce5729737780b96c96d2cf4d3fc665105b32d751d20
   ADD 0000_50_cloud-credential-operator_05_deployment.yaml /release-manifests/0000_50_cloud-credential-operator_05_deployment.yaml
   ADD image-references /release-manifests/image-references
   ```

1. Publish images to your personal repository and update the code to use the new
   payload:

   ```
   podman build -t quay.io/$USER/release-payload ./
   podman push quay.io/$USER/release-payload
   ```
