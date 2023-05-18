package lib.common
import future.keywords.in

# shared structures, functions, etc.

is_priv_namespace(ns) {
  privileged_ns[ns]
}

privileged_ns = {
  # Kubernetes specific namespaces
  "kube-node-lease",
  "kube-public",
  "kube-system",

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

exempted_service_account = {
  "default",
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
  obj.kind == "CronJob"
  spec := obj.spec.jobTemplate.spec.template.spec.serviceAccountName
} {
  obj.kind in ["ReplicationController","ReplicaSet","Deployment","StatefulSet","DaemonSet","Job"]
  spec := obj.spec.template.spec.serviceAccountName
}

has_service_account(obj) {
  obj.kind in ["Pod","CronJob","ReplicationController","ReplicaSet","Deployment","StatefulSet","DaemonSet","Job"]
}

get_user_info(review) = info {
  has_field(review.userInfo, "username")
  username := get_user(review)
  info := sprintf("user name %v", [username])
} {
  not has_field(review.userInfo, "username")
  has_service_account(review.object)
  sa := get_service_account(review.object)
  info := sprintf("service account %v", [sa])
}

# this setup is to handle below case:
# user default::notfound not allowed to operate in namespace openshift-kube-scheduler
#
# assume for cmdline operations, userInfo is always present, which is the only key for user identity
# while for serviceAccount operations, no userInfo is present, and we have to rely on the serviceAccountName field in the object

is_exempted_account(review) {
  has_field(review.userInfo, "username")
  username := get_user(review)
  is_exempted_user(username)
  print("exempted user:", username)
} {
  not has_field(review.userInfo, "username")
  sa := get_service_account(review.object)
  is_exempted_service_account(sa)
  print("exempted account:", sa)
}

is_exempted_service_account(user) {
  exempted_service_account[user]
}

get_user(review) = name {
  not has_field(review.userInfo, "username")
  name = "notfound"
} {
  has_field(review.userInfo, "username")
  name = review.userInfo.username
  print(name)
}

has_field(object, field) = true {
    object[field]
}

is_exempted_user(user) {
  exempted_user[user]
}

exempted_user = {
  "system:admin" # comment out temporarily for testing in console
}