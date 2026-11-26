# Local operator + OTel image testing on an OpenShift cluster

This guide shows how to:

1. Build local test images with Docker
2. Push them to the connected cluster's internal registry
3. Point ARO operator + Geneva OTel exporters at those images

## Prerequisites

- You are logged into the target cluster: `oc whoami`
- Docker is available locally
- You are at repo root (`ARO-RP`)
- For Go builds, use the repo tags:

```bash
export GOFLAGS='-tags=containers_image_openpgp,exclude_graphdriver_btrfs,exclude_graphdriver_devicemapper'
```

## 1. Login to the OpenShift internal registry

Enable default route (one-time, if needed):

```bash
oc patch configs.imageregistry.operator.openshift.io/cluster --type merge -p '{"spec":{"defaultRoute":true}}'
```

Login:

```bash
REGISTRY_HOST="$(oc get route default-route -n openshift-image-registry -o jsonpath='{.spec.host}')"
docker login -u "$(oc whoami)" -p "$(oc whoami -t)" "${REGISTRY_HOST}"
```

> **Not using `oc login`?** If you're on a certificate-based admin kubeconfig
> (e.g. `userAdminKubeconfig` from `hack/db`), `oc whoami -t` has no token — use a
> service-account token for the registry login instead:
> `docker login -u builder -p "$(oc create token builder -n openshift-azure-operator)" "${REGISTRY_HOST}"` (podman: add `--tls-verify=false`).

## 2. Build and push OTel image (Dockerfile.telemetryexporter)

Use `Dockerfile.telemetryexporter` (includes `systemd`/`journalctl` support needed by `journald` receiver):

```bash
VERSION="$(git rev-parse --short HEAD)"
OTEL_IMG="${REGISTRY_HOST}/openshift-azure-operator/aro-otel-local:${VERSION}"

docker build --no-cache \
  -f Dockerfile.telemetryexporter \
  --build-arg VERSION="${VERSION}" \
  --build-arg REGISTRY=registry.access.redhat.com \
  --build-arg BUILDER_REGISTRY=quay.io \
  -t "${OTEL_IMG}" .

docker push "${OTEL_IMG}"
```

## 3. Build and push operator image (local binary + UBI minimal)

Build binary:

```bash
CGO_ENABLED=0 go build -o _out/aro-operator ./cmd/aro
```

Build/push image:

```bash
OP_IMG="${REGISTRY_HOST}/openshift-azure-operator/aro-operator-local:${VERSION}"

cat > /tmp/Dockerfile.aro-operator-local <<'EOF'
FROM registry.access.redhat.com/ubi9/ubi-minimal:latest
COPY --chmod=0755 aro-operator /usr/local/bin/aro
USER 1000
ENV PATH=/usr/local/bin
ENTRYPOINT ["/usr/local/bin/aro", "operator"]
EOF

docker build -f /tmp/Dockerfile.aro-operator-local -t "${OP_IMG}" _out
docker push "${OP_IMG}"
```

**Note**: Upon retry, use `--no-cache` with the `docker build` command, to ensure that all latest dependencies are pulled.

The UBI base supplies the CA trust bundle required by operator controllers that
connect to Azure services. A scratch image does not include these certificates.

## 4. Set minimal-logs as baseline feature flags

```bash
oc -n default patch cluster.aro.openshift.io cluster --type merge -p \
  '{"spec":{"operatorflags":{"aro.genevalogging.otel.profile":"minimal-logs","aro.genevalogging.otel.master.profile":"minimal-logs","aro.genevalogging.otel.worker.profile":"minimal-logs"}}}'
```

## 5. Roll operator to the local image

```bash
oc -n openshift-azure-operator set image deployment/aro-operator-worker aro-operator="${OP_IMG}"
oc -n openshift-azure-operator set image deployment/aro-operator-master aro-operator="${OP_IMG}"

oc -n openshift-azure-operator rollout status deployment/aro-operator-worker --timeout=8m
oc -n openshift-azure-operator rollout status deployment/aro-operator-master --timeout=8m
```

## 6. Point Geneva OTel to the local exporter image

```bash
oc -n default patch cluster.aro.openshift.io cluster --type merge -p \
  "{\"spec\":{\"operatorflags\":{\"aro.genevalogging.otel.pullSpec\":\"${OTEL_IMG}\"}}}"
```

Restart exporters after config/image changes:

```bash
oc -n openshift-azure-logging rollout restart daemonset/otel-exporter-master
oc -n openshift-azure-logging rollout restart daemonset/otel-exporter-worker

oc -n openshift-azure-logging rollout status daemonset/otel-exporter-master --timeout=8m
oc -n openshift-azure-logging rollout status daemonset/otel-exporter-worker --timeout=8m
```

## 7. Quick verification

```bash
oc -n openshift-azure-operator get pods -l app=aro-operator-worker -o wide
oc -n openshift-azure-operator get pods -l app=aro-operator-master -o wide
oc -n openshift-azure-logging get pods -l app=otel-exporter-worker -o wide
oc -n openshift-azure-logging get pods -l app=otel-exporter-master -o wide

oc -n openshift-azure-logging get configmap otel-config -o jsonpath='{.data.worker-config\.yaml}' | head -n 60
```

## 8. Run the OTEL gateway e2e test

`test/e2e/otel_gateway.go` checks that the operator-rendered exporter config
delivers containers, journald and audit logs to a synthetic in-cluster
collector. It runs on local clusters only and is skipped unless
`TELEMETRY_COLLECTOR_IMAGE` is set.

Set `TELEMETRY_COLLECTOR_IMAGE` to the collector image you are about to
release: `arointsvc.azurecr.io/telemetrycollector:<ARO-RP release tag>`, built
by the `image-telemetry-collector` pipeline in sdp-pipelines. Dev clusters can
pull from `arointsvc` without extra credentials. The exporter copies use
whatever exporter image the operator runs; to test a new exporter image, set
`aro.genevalogging.otel.pullSpec` first (step 6).

Run it like any local e2e test ([E2e tests](./testing.md#e2e-tests)): local RP
running, `./env` sourced with `CLUSTER`/`RESOURCEGROUP` pointing at your
cluster, and `CI` unset (with `CI` set, the suite creates and deletes
clusters). The cluster needs `spec.gatewayPrivateEndpointIP` set on its Cluster
CR, otherwise the operator doesn't create the exporters.

```bash
export TELEMETRY_COLLECTOR_IMAGE=arointsvc.azurecr.io/telemetrycollector:<ARO-RP release tag>
make e2e.test
./e2e.test --ginkgo.focus "OTEL gateway integration"
```

### What the test covers, and what it doesn't

- It tests the operator deployed on the cluster, not your local checkout. To
  cover operator changes, roll out your operator image first (steps 3 and 5).
- Fields the test overrides on its exporter copies are not covered: gateway
  TLS, journald `start_at`, the gateway endpoint and hostAliases, file storage,
  and names, labels, annotations and the container name.
- The collector runs a static config
  (`test/e2e/static_resources/otel-collector.yaml`), not the production gateway
  config, so gateway authentication, processors, routing and the Kusto/mdsd
  exporters are not covered, and neither is exporter-to-gateway TLS.

## Notes

- `aro-operator-master` runs controllers like `GenevaLogging`; if it is scaled to `0`, profile/config changes may not reconcile.
- If a new image fails with `not executable`, ensure your Dockerfile uses either:
  - `COPY --chmod=0755 ...`, or
  - `RUN chmod 0755 ...` (only if the base image has a shell/coreutils).
