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
- When set to false, the controller deletes the aro-machinehealthcheck CR and the MHC Remediation alert.
- When set to true, the controller creates the MHC from a manifest if it does not exist, then on
  subsequent reconciles only enforces:
  - maxUnhealthy defaults to 1 if set to 0 (customer overrides to other values are preserved)
  - Required selector matchExpressions (exclude masters, require machineset membership) are restored
    if removed or corrupted; customer-added expressions are preserved
  - Pause annotation is added during cluster upgrades and removed when complete

More information about how the MHC works can be found here:
https://docs.openshift.com/container-platform/4.12/machine_management/deploying-machine-health-checks.html

*/
