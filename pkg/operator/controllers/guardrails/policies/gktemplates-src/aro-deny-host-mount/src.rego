package arohostmount

import future.keywords.in
import data.lib.common.is_priv_namespace
import data.lib.common.is_exempted_account
import data.lib.common.get_user

violation[{"msg": msg}] {

  is_pod(input.review.object)
  ns := input.review.object.metadata.namespace
  not is_priv_namespace(ns)

  # user := get_user(input.review)
  not is_exempted_account(input.review)

  c := input_containers[_]
  input_allow_privilege_escalation(c)

  volume := input_hostpath_volumes[_]
  writeable_volume(c, volume.name)

  msg := sprintf("HostPath volume %v is not allowed for write, pod %v.", [volume.name, input.review.object.metadata.name])
} {
  
  is_pv(input.review.object)

  # allow the exempted users?
  not is_exempted_account(input.review)

  has_field(input.review.object.spec, "hostPath")
  has_field(input.review.object.spec, "accessModes")
  writeable_pv(input.review.object.spec.accessModes)
  msg := sprintf("HostPath PersistentVolume %v is not allowed for write.", [input.review.object.metadata.name])

}

writeable_pv(accessModes) {
  mode := accessModes[_]
  mode in ["ReadWriteOnce", "ReadWriteMany", "ReadWriteOncePod"]
}

writeable_volume(container, volume_name) {
    mount := container.volumeMounts[_]
    mount.name == volume_name
    not mount.readOnly
}

input_allow_privilege_escalation(c) {
    not has_field(c, "securityContext")
}
input_allow_privilege_escalation(c) {
    not c.securityContext.allowPrivilegeEscalation == false
}

input_hostpath_volumes[v] {
    v := input.review.object.spec.volumes[_]
    has_field(v, "hostPath")
}

has_field(object, field) = true {
    object[field]
}

input_containers[c] {
    c := input.review.object.spec.containers[_]
}

input_containers[c] {
    c := input.review.object.spec.initContainers[_]
}

input_containers[c] {
    c := input.review.object.spec.ephemeralContainers[_]
}

is_pv(obj) {
  obj.apiVersion == "v1"
  obj.kind == "PersistentVolume"
}

is_pod(obj) {
  obj.apiVersion == "v1"
  obj.kind == "Pod"
}