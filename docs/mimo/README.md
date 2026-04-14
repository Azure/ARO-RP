# MIMO Documentation

The Managed Infrastructure Maintenance Operator, or MIMO, is a component of the Azure Red Hat OpenShift Resource Provider (ARO-RP) which is responsible for automated maintenance of clusters provisioned by the platform.
MIMO specifically focuses on "managed infrastructure", the parts of ARO that are deployed and maintained by the RP and ARO Operator instead of by OCP (in-cluster) or Hive (out-of-cluster).

MIMO consists of two main components, the [Actuator](./actuator.md) and the [Scheduler](./scheduler.md). It is primarily interfaced with via the [Admin API](./admin-api.md).

For a detailed understanding of how maintenance manifests work throughout their lifecycle, see the [Maintenance Manifest Lifecycle](./maintenance-manifest-lifecycle.md) document.

## A Primer On MIMO

The smallest thing that you can tell MIMO to run is a **Task** (see [`pkg/mimo/tasks/`](../../pkg/mimo/tasks/)).
A Task is composed of reusable **Steps** (see [`pkg/mimo/steps/`](../../pkg/mimo/steps/)), reusing the framework utilised by AdminUpdate/Update/Install methods in `pkg/cluster/`.
A Task only runs in the scope of a singular cluster.
These steps are run in sequence and can return either **Terminal** errors (causing the ran Task to fail and not be retried) or **Transient** errors (which indicates that the Task can be retried later).

Tasks are executed by the **Actuator** by way of creation of a **Maintenance Manifest**.
This Manifest is created with the cluster ID (which is elided from the cluster-scoped Admin APIs), the Task ID (which is currently a UUID), and optional priority, "start after", and "start before" times which are filled in with defaults if not provided.
The Actuator will treat these Maintenance Manifests as a work queue, taking ones which are past their "start after" time and executing them in order of earliest start-after and priority.
After running each, a state will be written into the Manifest (with optional free-form status text) with the result of the ran Task.
Manifests past their start-before times are marked as having a "timed out" state and not ran.

Currently, Manifests are created by the Admin API.
In the future, the Scheduler will create some these Manifests depending on cluster state/version and wall-clock time, providing the ability to perform tasks like rotations of secrets autonomously.
