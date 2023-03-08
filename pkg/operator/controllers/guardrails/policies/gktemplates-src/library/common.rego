package lib.common

# shared structures, functions, etc.

is_priv_namespace(ns) {
  privileged_ns[ns]
}

privileged_ns = {
		# ARO specific namespaces
		"openshift-azure-logging",
		"openshift-azure-operator",
		"openshift-managed-upgrade-operator",

		# OCP namespaces
		"openshift",
		"openshift-apiserver",
		"openshift-apiserver-operator",
		"openshift-authentication-operator",
		"openshift-cloud-controller-manager",
		"openshift-cloud-controller-manager-operator",
		"openshift-cloud-credential-operator",
		"openshift-cluster-csi-drivers",
		"openshift-cluster-machine-approver",
		"openshift-cluster-node-tuning-operator",
		"openshift-cluster-samples-operator",
		"openshift-cluster-storage-operator",
		"openshift-cluster-version",
		"openshift-config",
		"openshift-config-managed",
		"openshift-config-operator",
		"openshift-console",
		"openshift-console-operator",
		"openshift-console-user-settings",
		"openshift-controller-manager",
		"openshift-controller-manager-operator",
		"openshift-dns",
		"openshift-dns-operator",
		"openshift-etcd",
		"openshift-etcd-operator",
		"openshift-host-network",
		"openshift-image-registry",
		"openshift-ingress",
		"openshift-ingress-canary",
		"openshift-ingress-operator",
		"openshift-insights",
		"openshift-kni-infra",
		"openshift-kube-apiserver",
		"openshift-kube-apiserver-operator",
		"openshift-kube-controller-manager",
		"openshift-kube-controller-manager-operator",
		"openshift-kube-scheduler",
		"openshift-kube-scheduler-operator",
		"openshift-kube-storage-version-migrator",
		"openshift-kube-storage-version-migrator-operator",
		"openshift-machine-api",
		"openshift-machine-config-operator",
		"openshift-marketplace",
		"openshift-monitoring",
		"openshift-multus",
		"openshift-network-diagnostics",
		"openshift-network-operator",
		"openshift-oauth-apiserver",
		"openshift-openstack-infra",
		"openshift-operators",
		"openshift-operator-lifecycle-manager",
		"openshift-ovirt-infra",
		"openshift-sdn",
		"openshift-service-ca",
		"openshift-service-ca-operator"
}

is_exempted_user(user) {
  exempted_user[user]
}

exempted_user = {
  # "default",
  "aro-sre",
  "openshift-apiserver-operator",
  "openshift-apiserver-sa",
  "authentication-operator",
  "geneva",
  "aro-operator-worker",
  "cluster-cloud-controller-manager",
  "cloud-credential-operator",
  "azure-disk-csi-driver-controller-sa",
  "azure-disk-csi-driver-node-sa",
  "azure-disk-csi-driver-operator",
  "machine-approver-sa",
  "cluster-node-tuning-operator",
  "tuned",
  "cluster-samples-operator",
  "cluster-storage-operator",
  "csi-snapshot-controller",
  "csi-snapshot-controller-operator",
  "openshift-config-operator",
  "console-operator",
  "console",
  "openshift-controller-manager-operator",
  "openshift-controller-manager-sa",
  "dns-operator",
  "dns",
  "node-resolver",
  "etcd-operator",
  "cluster-image-registry-operator",
  "registry",
  "node-ca",
  "ingress-operator",
  "router",
  "operator",
  "kube-apiserver-operator",
  "kube-controller-manager-operator",
  "openshift-kube-scheduler-operator",
  "kube-storage-version-migrator-operator",
  "kube-storage-version-migrator-sa",
  "cluster-autoscaler-operator",
  "cluster-baremetal-operator",
  "cluster-baremetal-operator",
  "machine-api-controllers",
  "machine-api-operator",
  "machine-config-controller",
  "machine-config-daemon",
  "machine-config-server",
  "managed-upgrade-operator",
  "marketplace-operator",
  "alertmanager-main",
  "cluster-monitoring-operator",
  "grafana",
  "kube-state-metrics",
  "node-exporter",
  "openshift-state-metrics",
  "prometheus-adapter",
  "prometheus-k8s",
  "prometheus-operator",
  "thanos-querier",
  "multus",
  "metrics-daemon-sa",
  "network-diagnostics",
  "oauth-apiserver-sa",
  "collect-profiles",
  "olm-operator-serviceaccount",
  "sdn",
  "sdn-controller",
  "service-ca-operator",
  "service-ca",
  "pruner",
  "installer-sa"
}

get_service_account(obj) = spec {
  obj.kind == "Pod"
  spec := obj.spec.serviceAccountName
} {
  obj.kind == "ReplicationController"
  spec := obj.spec.template.spec.serviceAccountName
} {
  obj.kind == "ReplicaSet"
  spec := obj.spec.template.spec.serviceAccountName
} {
  obj.kind == "Deployment"
  spec := obj.spec.template.spec.serviceAccountName
} {
  obj.kind == "StatefulSet"
  spec := obj.spec.template.spec.serviceAccountName
} {
  obj.kind == "DaemonSet"
  spec := obj.spec.template.spec.serviceAccountName
} {
  obj.kind == "Job"
  spec := obj.spec.template.spec.serviceAccountName
} {
  obj.kind == "CronJob"
  spec := obj.spec.jobTemplate.spec.template.spec.serviceAccountName
}
