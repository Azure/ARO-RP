package arodenylabels

violation[{"msg": msg}] {
  input.review.operation == "DELETE"
  label_value := input.review.object.metadata.labels[key]
  deny_label := input.parameters.labels[_]
  deny_label.key == key
  # An undefined denyRegex, should have the same effect as an empty denyRegex
  deny_regex := object.get(deny_label, "denyRegex", "")
  re_match(deny_regex, label_value)
  def_msg := sprintf("Operation not allowed. Label <%v: %v> matches deny regex: <%v>", [key, label_value, deny_regex])
  msg := get_message(input.parameters, def_msg)
}

get_message(parameters, _default) = msg {
  not parameters.message
  msg := _default
}

get_message(parameters, _default) = msg {
  msg := parameters.message
}
