<!--
Customer-facing article for Azure Red Hat OpenShift, parallel to
https://access.redhat.com/articles/7033551 (ROSA).

Publish on Microsoft Learn. The ARO support policy allows a customer
KubeletConfig only when ARO documentation explicitly allows it. This
article is that allowance, and only for the procedure below.
-->

# Changing the maximum number of process IDs per pod (podPidsLimit) for Azure Red Hat OpenShift

## Table of contents

- [Understanding process ID limits in Managed OpenShift](#understanding-process-id-limits-in-managed-openshift)
- [Risks of setting higher process ID limits for Azure Red Hat OpenShift pods](#risks-of-setting-higher-process-id-limits-for-azure-red-hat-openshift-pods)
- [Setting a higher podPidsLimit on an existing Azure Red Hat OpenShift cluster](#setting-a-higher-podpidslimit-on-an-existing-azure-red-hat-openshift-cluster)
- [Migrating to a higher podPidsLimit on an existing Azure Red Hat OpenShift cluster](#migrating-to-a-higher-podpidslimit-on-an-existing-azure-red-hat-openshift-cluster)

## Understanding process ID limits in Managed OpenShift

A process identifier (PID) is a unique identifier assigned by the Linux kernel to each process or thread currently running on a system. The number of processes that can run simultaneously on a system is limited to 4,194,304 by the Linux kernel, and may additionally be affected when PIDs are reserved by the system for stability purposes, or by limited access to other system resources such as memory, CPU, and disk space.

In Managed OpenShift, there are two supported limits for PID usage to consider when you schedule work on your cluster:

- The maximum number of PIDs per pod (default: 4,096 in OpenShift 4.11 and higher). This is controlled by the `podPidsLimit` parameter set on the node.
- The maximum number of PIDs per node. The node-wide PID capacity is constrained by kernel limits such as `kernel.pid_max` and `kernel.threads-max`, and by the resources available on the node. The default depends on the node. The kubelet `--system-reserved` setting reserves resources for system processes. It does not set these kernel limits. Azure Red Hat OpenShift manages that reservation. Leave it unchanged.

When a pod reaches its maximum number of PIDs, the kernel rejects new processes and threads in that pod, and the pod may stop functioning correctly. Reaching this per-pod limit does not itself evict the pod. Node-wide PID pressure is a separate condition and can evict pods. See [Eviction signals and thresholds](https://kubernetes.io/docs/concepts/scheduling-eviction/node-pressure-eviction/).

When a node exceeds the allowed maximum number of PIDs per node, the node can become unstable because new processes cannot have PIDs assigned. If existing processes cannot complete without creating additional processes, the entire node may become unusable and require reboot. This may result in data loss, depending on the processes and applications being run. Customer administrators and Site Reliability Engineering are notified when this threshold is reached, and you will notice a `Worker node is experiencing PIDPressure` warning in cluster logs.

## Risks of setting higher process ID limits for Azure Red Hat OpenShift pods

The `podPidsLimit` parameter for a pod controls the maximum number of processes and threads that can run simultaneously in that pod.

You can increase the value for `podPidsLimit` from the default of 4,096 to a maximum of 16,384. Changing this value may incur downtime for applications, because changing the `podPidsLimit` requires rebooting the affected node.

If you are running a large number of pods per node, and you have a high `podPidsLimit` value on your nodes, you risk exceeding the PID maximum for the node.

For a PID-only upper bound on pod density, divide the node's usable PID budget by `podPidsLimit` and round down. Determine that budget from the node's kernel limits, and leave headroom for system processes. The budget is not the same on every node. For example, if the usable PID budget is 3,650,000 and `podPidsLimit` is 16,384, at most 222 pods could reach that limit at the same time. This calculation does not by itself establish a safe pod count.

Memory, CPU, and available storage can also limit the maximum number of pods that can run simultaneously, even when `podPidsLimit` is set appropriately. For more information, see the [Azure Red Hat OpenShift service definition](https://learn.microsoft.com/azure/openshift/openshift-service-definitions).

## Setting a higher podPidsLimit on an existing Azure Red Hat OpenShift cluster

You can set a higher `podPidsLimit` on an existing Azure Red Hat OpenShift cluster by creating or editing a `KubeletConfig` that sets `podPidsLimit` for the worker nodes.

**Important:** Changing the `podPidsLimit` on an existing cluster will trigger non-control plane nodes in the cluster to reboot one at a time. Make this change outside of peak usage hours for your cluster, and avoid upgrading the cluster until all nodes have rebooted.

You need `cluster-admin` access. [Connect to the cluster](https://learn.microsoft.com/azure/openshift/tutorial-connect-cluster) and confirm the machine config pools are idle (`UPDATING` is `False` and `DEGRADED` is `False`):

```console
$ oc get machineconfigpool
```

The first time you change the default value, create the `KubeletConfig`. List the objects Azure Red Hat OpenShift already owns, and leave those objects unchanged:

```console
$ oc get kubeletconfig
```

On a cluster that has `dynamic-node` (the usual case), save the following manifest as `aro-pod-pids-limit.yaml`. Set `podPidsLimit` to an integer from 4096 through 16384. The manifest sets only `podPidsLimit`.

```yaml
apiVersion: machineconfiguration.openshift.io/v1
kind: KubeletConfig
metadata:
  name: aro-pod-pids-limit
spec:
  machineConfigPoolSelector:
    matchLabels:
      pools.operator.machineconfiguration.openshift.io/worker: ""
  kubeletConfig:
    podPidsLimit: 16384
```

On a cluster that has `aro-limits` instead of `dynamic-node`, use the same manifest and add the `evictionHard` values from `oc get kubeletconfig aro-limits -o yaml`. The service sets these values:

```yaml
apiVersion: machineconfiguration.openshift.io/v1
kind: KubeletConfig
metadata:
  name: aro-pod-pids-limit
spec:
  machineConfigPoolSelector:
    matchLabels:
      pools.operator.machineconfiguration.openshift.io/worker: ""
  kubeletConfig:
    podPidsLimit: 16384
    evictionHard:
      memory.available: "500Mi"
      nodefs.available: "10%"
      nodefs.inodesFree: "5%"
      imagefs.available: "15%"
```

Create the object:

```console
$ oc apply -f aro-pod-pids-limit.yaml
```

When you make subsequent changes, edit the existing `KubeletConfig` and change `podPidsLimit`:

```console
$ oc edit kubeletconfig aro-pod-pids-limit
```

For example, to set a maximum of 16,384 PIDs per pod when you have never changed the default value, apply the manifest above with `podPidsLimit: 16384`.

This triggers a rolling reboot of worker nodes in the cluster. Wait until the worker pool shows `UPDATED=True`, `UPDATING=False`, and `DEGRADED=False`:

```console
$ oc get machineconfigpool
```

After the worker pool finishes updating, verify the new setting in a worker node's `kubelet.conf` file. Select a worker and use its name for `<node_name>` in every check in this article, including the migration section. Control plane nodes are not changed by this manifest and still report `podPidsLimit: 4096`.

```console
$ oc get nodes -l node-role.kubernetes.io/worker
```

```console
$ oc debug node/<node_name>
$ chroot /host
$ grep podPidsLimit /etc/kubernetes/kubelet.conf
```

You can run the same check as a single command:

```console
$ oc debug node/<node_name> -- chroot /host grep podPidsLimit /etc/kubernetes/kubelet.conf
```

You should see the new `podPidsLimit` in the output, for example:

```text
podPidsLimit: 16384
```

## Migrating to a higher podPidsLimit on an existing Azure Red Hat OpenShift cluster

In Azure Red Hat OpenShift 4.11 and later, you can set a custom PIDs limit for your cluster by creating a `KubeletConfig` and setting the `podPidsLimit` parameter for each worker node in the cluster. If you previously configured your cluster to add a custom PIDs limit through a support exception, set the PIDs limit with the `KubeletConfig` in this article and then remove your customization.

**Important:** To change the `podPidsLimit` in an existing cluster, it must be running Azure Red Hat OpenShift 4.11 or later. Changing the `podPidsLimit` on an existing cluster will trigger non-control plane nodes in the cluster to reboot one at a time. Make this change outside of peak usage hours for your cluster, and avoid upgrading the cluster until all nodes have rebooted.

This migration preserves the kubelet per-pod limit, `podPidsLimit`. A `ContainerRuntimeConfig` field named `pidsLimit` is a separate per-container CRI-O limit. These commands do not read that container limit. Leave any `ContainerRuntimeConfig` in place, and ask support before removing it.

Determine the current kubelet `podPidsLimit` on a worker. Use a name from `oc get nodes -l node-role.kubernetes.io/worker`. A control plane node still reports the default of 4096, and using that value here would lower a customized worker limit.

```console
$ oc debug node/<node_name>
$ chroot /host
$ grep podPidsLimit /etc/kubernetes/kubelet.conf
```

Or run the same check as a single command:

```console
$ oc debug node/<node_name> -- chroot /host grep podPidsLimit /etc/kubernetes/kubelet.conf
```

```text
podPidsLimit: 4096
```

Create `aro-pod-pids-limit` with a `podPidsLimit` that matches your current, customized limit, using the manifest in [Setting a higher podPidsLimit](#setting-a-higher-podpidslimit-on-an-existing-azure-red-hat-openshift-cluster):

```console
$ oc apply -f aro-pod-pids-limit.yaml
```

If you set the `podPidsLimit` value to be different from your previous, customized limit, this change will trigger a rolling reboot of the worker nodes in the cluster.

Before you remove the earlier customization, confirm that `aro-pod-pids-limit` has reconciled and that the worker pool has converged. A matching value in `kubelet.conf` can still come from the old customization.

```console
$ oc get kubeletconfig aro-pod-pids-limit -o yaml
```

The object reports a `Success` condition with `status: "True"`. Every affected machine config pool shows `UPDATED=True`, `UPDATING=False`, and `DEGRADED=False`.

Then verify the PIDs limit on the same worker:

```console
$ oc debug node/<node_name>
$ chroot /host
$ grep podPidsLimit /etc/kubernetes/kubelet.conf
```

Or run the same check as a single command:

```console
$ oc debug node/<node_name> -- chroot /host grep podPidsLimit /etc/kubernetes/kubelet.conf
```

Remove the earlier kubelet customization that set `podPidsLimit`. For example, remove a `MachineConfig` or `KubeletConfig` created by a support case for that kubelet setting. Leave `dynamic-node`, `aro-limits`, and any `ContainerRuntimeConfig` in place.

Removing that customization can start another worker rollout. Watch the machine config pools until every affected pool shows `UPDATED=True`, `UPDATING=False`, and `DEGRADED=False`:

```console
$ oc get machineconfigpool
```

Repeat the `kubelet.conf` check on a worker and confirm that `podPidsLimit` is still the value you set. Wait until this rollout finishes before you upgrade the cluster.
