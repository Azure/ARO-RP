package arodenypvrwhostpath

import future.keywords.in
import data.lib.common.is_priv_namespace
import data.lib.common.is_exempted_account

violation[{"msg": msg}] {

  is_pv(input.review.object)

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


has_field(object, field) = true {
    object[field]
}

is_pv(obj) {
  obj.apiVersion == "v1"
  obj.kind == "PersistentVolume"
}
