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

## 3. Build and push operator image (local binary + minimal Dockerfile)

Build binary:

```bash
CGO_ENABLED=0 go build -o _out/aro-operator ./cmd/aro
```

Build/push image:

```bash
OP_IMG="${REGISTRY_HOST}/openshift-azure-operator/aro-operator-local:${VERSION}"

cat > /tmp/Dockerfile.aro-operator-local <<'EOF'
FROM scratch
COPY --chmod=0755 aro-operator /usr/local/bin/aro
USER 1000
ENV PATH=/usr/local/bin
ENTRYPOINT ["/usr/local/bin/aro", "operator"]
EOF

docker build -f /tmp/Dockerfile.aro-operator-local -t "${OP_IMG}" _out
docker push "${OP_IMG}"
```

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

## Notes

- `aro-operator-master` runs controllers like `GenevaLogging`; if it is scaled to `0`, profile/config changes may not reconcile.
- If a new image fails with `not executable`, ensure your Dockerfile uses either:
  - `COPY --chmod=0755 ...`, or
  - `RUN chmod 0755 ...` (only if the base image has a shell/coreutils).
