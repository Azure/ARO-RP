package namespace

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"strings"
)

// IsOpenShift returns true if ns is an openshift managed namespace, including the default namespace
func IsOpenShift(ns string) bool {
	return ns == "" ||
		ns == "default" ||
		ns == "openshift" ||
		strings.HasPrefix(ns, "kube-") ||
		strings.HasPrefix(ns, "openshift-")
}

// IsOpenShiftSystemNamespace returns true if ns is an openshift managed namespace, without the default namespace
func IsOpenShiftSystemNamespace(ns string) bool {
	return ns == "" ||
		ns == "openshift" ||
		strings.HasPrefix(ns, "kube-") ||
		strings.HasPrefix(ns, "openshift-")
}

// FilteredOpenShiftNamespace returns true if ns is a namespace in the defined hardcoded map
func FilteredOpenShiftNamespace(ns string) bool {
	filtered_namespaces := map[string]bool{
		"openshift":                                        true,
		"openshift-apiserver":                              true,
		"openshift-apiserver-operator":                     true,
		"openshift-authentication":                         true,
		"openshift-authentication-operator":                true,
		"openshift-cloud-controller-manager":               true,
		"openshift-cloud-controller-manager-operator":      true,
		"openshift-cloud-credential-operator":              true,
		"openshift-cluster-csi-drivers":                    true,
		"openshift-cluster-machine-approver":               true,
		"openshift-cluster-node-tuning-operator":           true,
		"openshift-cluster-samples-operator":               true,
		"openshift-cluster-storage-operator":               true,
		"openshift-config":                                 true,
		"openshift-config-managed":                         true,
		"openshift-config-operator":                        true,
		"openshift-console":                                true,
		"openshift-console-operator":                       true,
		"openshift-console-user-settings":                  true,
		"openshift-controller-manager":                     true,
		"openshift-controller-manager-operator":            true,
		"openshift-dns":                                    true,
		"openshift-dns-operator":                           true,
		"openshift-etcd":                                   true,
		"openshift-etcd-operator":                          true,
		"openshift-host-network":                           true,
		"openshift-image-registry":                         true,
		"openshift-ingress":                                true,
		"openshift-ingress-canary":                         true,
		"openshift-ingress-operator":                       true,
		"openshift-insights":                               true,
		"openshift-kni-infra":                              true,
		"openshift-kube-apiserver":                         true,
		"openshift-kube-apiserver-operator":                true,
		"openshift-kube-controller-manager":                true,
		"openshift-kube-controller-manager-operator":       true,
		"openshift-kube-scheduler":                         true,
		"openshift-kube-scheduler-operator":                true,
		"openshift-kube-storage-version-migrator":          true,
		"openshift-kube-storage-version-migrator-operator": true,
		"openshift-machine-api":                            true,
		"openshift-machine-config-operator":                true,
		"openshift-marketplace":                            true,
		"openshift-monitoring":                             true,
		"openshift-multus":                                 true,
		"openshift-network-diagnostics":                    true,
		"openshift-network-operator":                       true,
		"openshift-oauth-apiserver":                        true,
		"openshift-openstack-infra":                        true,
		"openshift-operator-lifecycle-manager":             true,
		"openshift-operators":                              true,
		"openshift-ovirt-infra":                            true,
		"openshift-sdn":                                    true,
		"openshift-service-ca":                             true,
		"openshift-service-ca-operator":                    true,
		"openshift-user-workload-monitoring":               true,
		"openshift-vsphere-infra":                          true,
		"openshift-azure-operator":                         true,
		"openshift-managed-upgrade-operator":               true,
	}
	return filtered_namespaces[ns]
}
