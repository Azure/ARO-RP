package aroprivilegednamespace

violation[{"msg": msg}] {
  ns := input.review.object.metadata.namespace
  user := get_service_account(input.review.object)
  is_priv_namespace(ns)
  not_priv_user(user)
  msg := sprintf("User %v not allowed to operate in namespace %v", [user, ns])
}

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

not_priv_user(user) {
  not privileged_user[user]
}

privileged_user = {
  "aro-sre",
  "automated"
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
