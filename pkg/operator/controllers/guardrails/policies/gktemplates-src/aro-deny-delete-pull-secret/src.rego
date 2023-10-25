package arodenydeletepullsecret

import data.lib.common.is_exempted_account

violation[{"msg": msg}] {
    input.review.operation == "DELETE"
    # Check if it is a regular user
    not is_exempted_account(input.review)
    input.review.object.metadata.name == "pull-secret"
    msg := "Deleting pull secret is not allowed"
}
