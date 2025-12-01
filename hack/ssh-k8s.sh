#!/bin/bash -e

set -o pipefail

while [[ $1 == -* ]]; do
  if [[ $1 == -- ]]; then
    shift
    break
  fi
  shift
done

if [[ "$#" -gt 0 ]]; then
  NODENAME=$1
  shift
fi

COMMAND='bash --login'
TTY=true
TTYOPT=-t
if [[ "$#" -gt 0 ]]; then
  COMMAND="bash -c $(printf ' %q' "$@" | sed -e "s/'/''/g")"
  TTY=false
  TTYOPT=
fi

cleanup() {
    [[ -n "$POD" ]] && oc delete pod -n default "$POD" >/dev/null
}

trap cleanup EXIT

POD=$(oc create -o json -f - <<EOF | jq -r .metadata.name
kind: Pod
apiVersion: v1
metadata:
  generateName: debug
  labels:
    openshift.io/run-level: "0"
  namespace: default
spec:
  containers:
  - command:
    - /sbin/chroot
    - /host
    - /bin/bash
    - -c
    - 'cd && exec $COMMAND'
    image: ubi9/ubi-minimal
    name: debug
    securityContext:
      capabilities:
        add:
        - CHOWN
        - DAC_OVERRIDE
        - DAC_READ_SEARCH
        - FOWNER
        - FSETID
        - KILL
        - SETGID
        - SETUID
        - SETPCAP
        - LINUX_IMMUTABLE
        - NET_BIND_SERVICE
        - NET_BROADCAST
        - NET_ADMIN
        - NET_RAW
        - IPC_LOCK
        - IPC_OWNER
        - SYS_MODULE
        - SYS_RAWIO
        - SYS_CHROOT
        - SYS_PTRACE
        - SYS_PACCT
        - SYS_ADMIN
        - SYS_BOOT
        - SYS_NICE
        - SYS_RESOURCE
        - SYS_TIME
        - SYS_TTY_CONFIG
        - MKNOD
        - LEASE
        - AUDIT_WRITE
        - AUDIT_CONTROL
        - SETFCAP
        - MAC_OVERRIDE
        - MAC_ADMIN
        - SYSLOG
        - WAKE_ALARM
        - BLOCK_SUSPEND
        - AUDIT_READ
    stdin: true
    tty: $TTY
    volumeMounts:
    - mountPath: /host
      name: host
  hostIPC: true
  hostNetwork: true
  hostPID: true
  nodeName: "$NODENAME"
  restartPolicy: Never
  terminationGracePeriodSeconds: 0
  volumes:
  - hostPath:
      path: /
    name: host
EOF
)

oc wait --timeout=300s --for=condition=Ready -n default "pod/$POD" >/dev/null
oc attach -i $TTYOPT -n default -c debug "pod/$POD"
