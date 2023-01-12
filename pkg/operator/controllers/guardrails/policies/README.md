# Gatekeeper Policy development and testing

## What's Gatekeeper
see https://open-policy-agent.github.io/gatekeeper/website/docs/

## Folder structures

There are several folders under guardrails:

* gktemplates - the Constraint Templates used by gatekeeper, which are generated through generate.sh, do *not* modify them.
* gkconstraints - the Constraints that are used by gatekeeper together with Constraint Templates.

* gktemplates-src - the rego src file for Constraint Templates, consumed by generate.sh
* scripts - generate.sh will combine src.rego and *.tmpl to form actual Constraint Templates under gktemplates. test.sh executes the rego tests under each gktemplates-src subfolder.
* staticresources - yaml resources for gatekeeper deployment


## Rego Development

* Create a new subfolder for each new Constraint Template under gktemplates-src
* Create a tmpl file with unique and meaningful name in above subfolder, which contains everything except for the rego, refer https://open-policy-agent.github.io/gatekeeper/website/docs/constrainttemplates/, example:

```yaml
apiVersion: templates.gatekeeper.sh/v1
kind: ConstraintTemplate
metadata:
  name: aroprivilegednamespace
  annotations:
    metadata.gatekeeper.sh/title: "Privileged Namespace"
    metadata.gatekeeper.sh/version: 1.0.0
    description: >-
      Disallows creating, updating or deleting resources in privileged namespaces.
      including, ["^kube.*|^openshift.*|^default$|^redhat.*|^com$|^io$|^in$"]
spec:
  crd:
    spec:
      names:
        kind: AROPrivilegedNamespace
      validation:
        # Schema for the `parameters` field
        openAPIV3Schema:
          type: object
          description: >-
            Disallows creating, updating or deleting resources in privileged namespaces.
            including, ["^kube.*|^openshift.*|^default$|^redhat.*|^com$|^io$|^in$"]
  targets:
    - target: admission.k8s.gatekeeper.sh
      rego: |
{{ file.Read "gktemplates-src/aro-deny-privileged-namespace/src.rego" | strings.Indent 8 | strings.TrimSuffix "\n" }}
```


* Create the src.rego file in the same folder, howto https://www.openpolicyagent.org/docs/latest/policy-language/, example:
```
package aroprivilegednamespace

violation[{"msg": msg, "details": {}}] {
  input_priv_namespace(input.review.object.metadata.namespace)
  msg := sprintf("Operation in privileged namespace %v is not allowed", [input.review.object.metadata.namespace])
}

input_priv_namespace(ns) {
  any([ns == "default",
  ns == "com",
  ns == "io",
  ns == "in",
  startswith(ns, "openshift"),
  startswith(ns, "kube"),
  startswith(ns, "redhat")])
}
```
* Create src_test.rego for unit tests in the same foler, which will be called by test.sh, howto https://www.openpolicyagent.org/docs/latest/policy-testing/, example:
```
package aroprivilegednamespace

test_input_allowed_ns {
  input := { "review": input_ns(input_allowed_ns) }
  results := violation with input as input
  count(results) == 0
}

test_input_disallowed_ns1 {
  input := { "review": input_review(input_disallowed_ns1) }
  results := violation with input as input
  count(results) == 1
}

input_ns(ns) = output {
  output = {
    "object": {
      "metadata": {
        "namespace": ns
      }
    }
  }
}

input_allowed_ns = "mytest"

input_disallowed_ns1 = "openshift-config"
```

## Rego Testing

* install opa cli, refer https://www.openpolicyagent.org/docs/v0.11.0/get-started/

* after _test.go is done, test it out, and fix the problem
  ```sh
  opa test src.rego
  ```

## Generate the Constraint Templates

* install gomplate which is used by generate.sh, see https://docs.gomplate.ca/installing/

* execute test.sh under policies for unit testing:
  ```sh
  ARO-RP/pkg/operator/controllers/guardrails/policies$ ./scripts/test.sh
  ```

* execute generate.sh under policies, which will generate the acutal Constraint Templates under gktemplates folder, example:

  ```sh
  ARO-RP/pkg/operator/controllers/guardrails/policies$ ./scripts/generate.sh 
  Generating gktemplates/aro-deny-delete.yaml from gktemplates-src/aro-deny-delete/aro-deny-delete.tmpl
  Generating gktemplates/aro-deny-privileged-namespace.yaml from gktemplates-src/aro-deny-privileged-namespace/aro-deny-privileged-namespace.tmpl
  Generating gktemplates/aro-deny-labels.yaml from gktemplates-src/aro-deny-labels/aro-deny-labels.tmpl
  ```