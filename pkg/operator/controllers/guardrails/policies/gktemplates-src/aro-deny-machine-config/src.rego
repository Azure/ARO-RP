package arodenymachineconfig
import future.keywords.in
import data.lib.common.is_exempted_account
import data.lib.common.get_username

violation[{"msg": msg}] {
    input.review.operation in ["CREATE", "UPDATE", "DELETE"]

    # Check if it is a exempted user
    not is_exempted_account(input.review)
    username := get_username(input.review)

    # Check if it is a protected machine config
    mc := input.review.object.metadata.name
    is_protected_mc(mc)

    msg := sprintf("user %v not allowed to %v machine config %v", [username, input.review.operation, mc])
}

is_protected_mc(mc) = true {
    is_ocp_mc(mc)
} {
    # for rendered-master-542f4aec7e9ca2afda1955ea19266af9
    regex.match("^rendered-(master|worker)-.+$", mc)
}

is_ocp_mc(mc) = true {
    ocp_mc[mc]
}
ocp_mc = {
    # protected ocp machine configs
    "00-master",
    "00-worker",
    "01-master-container-runtime",
    "01-master-kubelet",
    "01-worker-container-runtime",
    "01-worker-kubelet",
    "90-aro-worker-registries",
    "97-master-generated-kubelet",
    "97-worker-generated-kubelet",
    "98-master-generated-kubelet",
    "98-worker-generated-kubelet",
    "99-master-aro-dns",
    "99-master-aro-etc-hosts-gateway-domains",
    "99-master-generated-kubelet",
    "99-master-generated-registries",
    "99-master-ssh",
    "99-worker-aro-dns",
    "99-worker-aro-etc-hosts-gateway-domains",
    "99-worker-generated-kubelet",
    "99-worker-generated-registries",
    "99-worker-ssh"
}
