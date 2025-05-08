package lib.common
import future.keywords.in

# shared structures, functions, etc.

is_exempted_account(review) = true {
  has_field(review, "userInfo")
  has_field(review.userInfo, "username")
  username := get_username(review)
  groups := get_user_group(review)
  is_exempted_user_or_groups(username, groups)
} {
  not has_field(review, "userInfo")
} {
  has_field(review, "userInfo")
  not has_field(review.userInfo, "username")
}

get_username(review) = name {
  not has_field(review.userInfo, "username")
  name = "notfound"
} {
  has_field(review.userInfo, "username")
  name = review.userInfo.username
  print(name)
}

get_user_group(review) = group {
    not review.userInfo
    group = []
} {
    not review.userInfo.groups
    group = []
} {
    group = review.userInfo.groups
}

is_exempted_user_or_groups(user, groups) = true {
  exempted_user[user]
  print("exempted user:", user)
} {
  group := [ g | g := groups[_]; (g in cast_set(exempted_groups)) ]
  count(group) > 0
  print("exempted group:", group)
}

has_field(object, field) = true {
    object[field]
}

is_exempted_user(user) = true {
  exempted_user[user]
}

is_priv_namespace(ns) = true {
  privileged_ns[ns]
}

exempted_user = {
  "system:kube-controller-manager",
  "system:kube-scheduler",
  "system:admin"
}

exempted_groups = {
  # "system:cluster-admins", # dont allow kube:admin
  "system:nodes",
  "system:serviceaccounts", # allow all system service accounts
  "system:masters"
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
  "openshift-azure-guardrails",

  # OCP namespaces
  "openshift",
  "openshift-apiserver",
  "openshift-apiserver-operator",
  "openshift-authentication-operator",
  "openshift-cloud-controller-manager",
  "openshift-cloud-controller-manager-operator",
  "openshift-cloud-credential-operator",
  "openshift-cluster-machine-approver",
  "openshift-cluster-storage-operator",
  "openshift-cluster-version",
  "openshift-config-managed",
  "openshift-config-operator",
  "openshift-console",
  "openshift-console-operator",
  "openshift-controller-manager",
  "openshift-controller-manager-operator",
  "openshift-dns",
  "openshift-dns-operator",
  "openshift-etcd",
  "openshift-etcd-operator",
  "openshift-host-network",
  "openshift-image-registry",
  "openshift-ingress",
  "openshift-ingress-operator",
  "openshift-kube-apiserver",
  "openshift-kube-apiserver-operator",
  "openshift-kube-controller-manager",
  "openshift-kube-controller-manager-operator",
  "openshift-kube-scheduler",
  "openshift-kube-scheduler-operator",
  "openshift-machine-api",
  "openshift-machine-config-operator",
  "openshift-monitoring",
  "openshift-multus",
  "openshift-network-operator",
  "openshift-oauth-apiserver",
  "openshift-operator-lifecycle-manager",
  "openshift-ovn-kubernetes",
  "openshift-sdn",
  "openshift-service-ca",
  "openshift-service-ca-operator"
}
