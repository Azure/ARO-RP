package machinehealthcheck

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

/*

The controller in this package aims to ensure the ARO MachineHealthCheck CR and MHC Remediation Alert
exist and are correctly configured to automatically mitigate non-ready worker nodes and create an in-cluster alert
when remediation is occurring frequently.

There are two flags which control the operations performed by the controller:

aro.machinehealthcheck.enabled:
- When set to false, the controller will noop and not perform any further action
- When set to true, the controller continues on to check the managed flag

aro.machinehealthcheck.managed
- When set to false, the controller will attempt to remove the aro-machinehealthcheck CR and the MHC Remediation alert from the cluster.
  This should effectively disable the MHC we deploy and prevent the automatic reconciliation of nodes.
- When set to true, the controller will deploy/overwrite the aro-machinehealthcheck CR and the MHC Remediation alert to the cluster.
  This enables the cluster to self heal when at most 1 worker node goes not ready for at least 15 minutes and alert when remediation
  occurs 2 or more times within an hour.

The aro-machinehealth check is configured in a way that if 2 worker nodes go not ready it will not take any action.
More information about how the MHC works can be found here:
https://docs.openshift.com/container-platform/4.12/machine_management/deploying-machine-health-checks.html

*/
