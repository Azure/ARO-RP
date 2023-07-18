package arodenymachineconfig
import future.keywords.in

violation[{"msg": msg}] {
    input.review.operation in ["CREATE", "UPDATE", "DELETE"]
    name := input.review.object.metadata.name
    regex.match("^.+(-master|-worker|-master-.+|-worker-.+|-kubelet|-container-runtime|-aro-.+|-ssh|-generated-.+)$", name)
    msg := "Modify cluster machine config is not allowed"
}
