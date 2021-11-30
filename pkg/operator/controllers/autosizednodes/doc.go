package autosizednodes

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

// autosizednodes monitors/creates/removes "dynamic-node" KubeletConfig
// that tells machine-config-operator to turn on auto sized nodes feature
// the code that is executed by the mco:
// - https://github.com/openshift/machine-config-operator/blob/fbc4d8e46a7746442f4de3651113d2181d458b12/templates/common/_base/files/kubelet-auto-sizing.yaml
