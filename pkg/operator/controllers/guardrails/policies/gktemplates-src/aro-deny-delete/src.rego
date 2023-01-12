package arodenydelete

violation[{"msg": msg}] {
  input.review.operation == "DELETE"
  deny_resource_name := input.parameters.name
  deny_resource_name == input.review.object.metadata.name
  msg := sprintf("Deleting resource - %v is not allowed", [deny_resource_name])
}
