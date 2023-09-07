package arodenycrd
import future.keywords.in
import data.lib.common.is_exempted_account

violation[{"msg": msg}] {
    input.review.operation in ["CREATE", "UPDATE", "DELETE"]
    name := input.review.object.metadata.name
    # Check if CRD name is ended with any of [gatekeeper.sh,openshift.io,metal3.io,coreos.com,cncf.io,k8s.io,ovn.org]
    regex.match("^.+(gatekeeper.sh|openshift.io|metal3.io|coreos.com|cncf.io|k8s.io|ovn.org)$", name)
    not is_exempted_account(input.review)
    msg := "Regular user not allowed to create or modify generated CRD"
}
