package arodenymachineconfig
import future.keywords.in
import data.lib.common.is_exempted_account

violation[{"msg": msg}] {
    input.review.operation in ["CREATE", "UPDATE", "DELETE"]

    # Check if it is a regular user
    not is_exempted_account(input.review)

    # Check if the object name matches the regex for generated machine configs
    name := input.review.object.metadata.name
    regex.match("^.+(-master|-worker|-master-.+|-worker-.+|-kubelet|-container-runtime|-aro-.+|-ssh|-generated-.+)$", name)
    msg := "Modify cluster machine config is not allowed"
}
