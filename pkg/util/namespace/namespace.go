package namespace

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

// IsOpenShiftNamespace returns true if ns is a namespace in the defined hardcoded map.
// We should only add new namespaces into this hardcoded list but never delete
// the existing ones in the namespace list to avoid backward compatibility issues.
func IsOpenShiftNamespace(ns string) bool {
	nsmap := map[string]struct{}{
		// Allow validation against non-namespaced objects via Geneva actions
		"": {},

		// ARO specific namespaces
		"openshift-azure-logging":            {},
		"openshift-azure-operator":           {},
		"openshift-managed-upgrade-operator": {},

		// OCP namespaces
		"openshift":                                        {},
		"openshift-apiserver":                              {},
		"openshift-apiserver-operator":                     {},
		"openshift-authentication-operator":                {},
		"openshift-cloud-controller-manager":               {},
		"openshift-cloud-controller-manager-operator":      {},
		"openshift-cloud-credential-operator":              {},
		"openshift-cloud-network-config-controller":        {},
		"openshift-cluster-api":                            {},
		"openshift-cluster-csi-drivers":                    {},
		"openshift-cluster-machine-approver":               {},
		"openshift-cluster-node-tuning-operator":           {},
		"openshift-cluster-samples-operator":               {},
		"openshift-cluster-storage-operator":               {},
		"openshift-cluster-version":                        {},
		"openshift-config":                                 {},
		"openshift-config-managed":                         {},
		"openshift-config-operator":                        {},
		"openshift-console":                                {},
		"openshift-console-operator":                       {},
		"openshift-console-user-settings":                  {},
		"openshift-controller-manager":                     {},
		"openshift-controller-manager-operator":            {},
		"openshift-dns":                                    {},
		"openshift-dns-operator":                           {},
		"openshift-etcd":                                   {},
		"openshift-etcd-operator":                          {},
		"openshift-host-network":                           {},
		"openshift-image-registry":                         {},
		"openshift-ingress":                                {},
		"openshift-ingress-canary":                         {},
		"openshift-ingress-operator":                       {},
		"openshift-insights":                               {},
		"openshift-kni-infra":                              {},
		"openshift-kube-apiserver":                         {},
		"openshift-kube-apiserver-operator":                {},
		"openshift-kube-controller-manager":                {},
		"openshift-kube-controller-manager-operator":       {},
		"openshift-kube-scheduler":                         {},
		"openshift-kube-scheduler-operator":                {},
		"openshift-kube-storage-version-migrator":          {},
		"openshift-kube-storage-version-migrator-operator": {},
		"openshift-machine-api":                            {},
		"openshift-machine-config-operator":                {},
		"openshift-marketplace":                            {},
		"openshift-monitoring":                             {},
		"openshift-multus":                                 {},
		"openshift-network-diagnostics":                    {},
		"openshift-network-operator":                       {},
		"openshift-oauth-apiserver":                        {},
		"openshift-openstack-infra":                        {},
		"openshift-operator-lifecycle-manager":             {},
		"openshift-operators":                              {},
		"openshift-ovirt-infra":                            {},
		"openshift-sdn":                                    {},
		"openshift-service-ca":                             {},
		"openshift-service-ca-operator":                    {},
	}
	_, ok := nsmap[ns]
	return ok
}
