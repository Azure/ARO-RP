package arodenylabels
import data.lib.common.is_exempted_account
import data.lib.common.get_username

violation[{"msg": msg}] {
  input.review.operation == "DELETE"
  label_value := input.review.object.metadata.labels[key]
  deny_label := input.parameters.labels[_]
  deny_label.key == key
  # An undefined denyRegex, should have the same effect as an empty denyRegex
  deny_regex := object.get(deny_label, "denyRegex", "")
  re_match(deny_regex, label_value)
  # Check if it is an exempted user
  not is_exempted_account(input.review)
  username := get_username(input.review)
  msg := sprintf("user <%v> not allowed to delete resource with label <%v: %v>", [username, key, label_value])
}
