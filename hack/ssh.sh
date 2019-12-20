#!/bin/bash -e

if [[ ! -e admin.kubeconfig ]]; then
    echo "run hack/get-admin-kubeconfig.sh first" >&2
    exit 1
fi

export KUBECONFIG=admin.kubeconfig

NODENAME=aro-master-0
[[ "$#" -gt 0 ]] && NODENAME=$1

cleanup() {
    [[ -n "$POD" ]] && oc delete pod -n default "$POD" >/dev/null
}

trap cleanup EXIT

POD=$(oc create -o json -f - <<EOF | jq -r .metadata.name
kind: Pod
apiVersion: v1
metadata:
  generateName: debug
  namespace: default
spec:
  containers:
  - command:
    - /sbin/chroot
    - /host
    - /bin/bash
    - -c
    - "cd && exec bash --login"
    image: ubi8/ubi-minimal
    name: debug
    stdin: true
    tty: true
    volumeMounts:
    - mountPath: /host
      name: host
  hostIPC: true
  hostNetwork: true
  hostPID: true
  nodeName: $NODENAME
  restartPolicy: Never
  terminationGracePeriodSeconds: 0
  volumes:
  - hostPath:
      path: /
    name: host
EOF
)

oc wait --for=condition=Ready "pod/$POD" >/dev/null
oc attach -it -n default -c debug "pod/$POD"
