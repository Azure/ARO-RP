package arodenycrd
import future.keywords.in
import data.lib.common.is_exempted_account
import data.lib.common.is_critical_crd

violation[{"msg": msg}] {
    input.review.operation in ["CREATE", "UPDATE", "DELETE"]
    name := input.review.object.metadata.name
    # Check if it is a regular user
    not is_exempted_account(input.review)
    # Check if CRD name falls into the critical CRD list
    is_critical_crd(name)
    msg := "Regular user not allowed to create or modify generated CRD"
}
