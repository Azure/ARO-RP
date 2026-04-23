# Guardrails policy development

This directory documents the two guardrails implementations currently used by the ARO operator:

- **Gatekeeper / Rego** for OpenShift **earlier than 4.17**
- **ValidatingAdmissionPolicy (VAP)** for OpenShift **4.17 and later**

At runtime the controller selects the implementation based on cluster version:

- pre-4.17: deploy and reconcile Gatekeeper resources
- 4.17+: reconcile VAP resources and clean up any leftover Gatekeeper resources from upgrades

The entry points for that behavior are in [pkg/operator/controllers/guardrails/guardrails_controller.go](../guardrails_controller.go), [pkg/operator/controllers/guardrails/guardrails_policy.go](../guardrails_policy.go), and [pkg/operator/controllers/guardrails/guardrails_vap.go](../guardrails_vap.go).

## Relevant folder structure

Under [pkg/operator/controllers/guardrails](..) there are two policy trees.

### Gatekeeper tree

- [policies/gktemplates](gktemplates) - generated `ConstraintTemplate` YAML. Do not edit directly.
- [policies/gkconstraints](gkconstraints) - Gatekeeper `Constraint` YAML consumed by the controller.
- [policies/gktemplates-src](gktemplates-src) - source for Gatekeeper policies.
- [policies/scripts](scripts) - Gatekeeper template generation and policy test scripts.

### VAP tree

- [../policies-vap/vap](../policies-vap/vap) - `ValidatingAdmissionPolicy` YAML files.
- [../policies-vap/vap-binding](../policies-vap/vap-binding) - `ValidatingAdmissionPolicyBinding` YAML templates.

## Common operator behavior

Both implementations are driven by the same operator flags:

- `aro.guardrails.enabled`
- `aro.guardrails.deploy.managed`
- `aro.guardrails.policies.<policy-name>.managed`
- `aro.guardrails.policies.<policy-name>.enforcement`

`<policy-name>` is the YAML file name without the `.yaml` suffix.

Examples:

- `aro.guardrails.policies.aro-machines-deny.managed`
- `aro.guardrails.policies.aro-machines-deny.enforcement`

The controller periodically re-applies the active policy resources so externally deleted resources are recreated.

## Gatekeeper approach (pre-4.17)

### What is deployed

For clusters below 4.17, the controller:

1. deploys Gatekeeper into the guardrails namespace
2. waits for `gatekeeper-audit` and `gatekeeper-controller-manager` to become ready
3. creates or updates generated `ConstraintTemplate` resources
4. creates or updates the `Constraint` resources from [gkconstraints](gkconstraints)
5. periodically re-applies the constraints via the reconciliation ticker

If `aro.guardrails.deploy.managed=false`, the controller cleans up Gatekeeper resources that it manages.

### Gatekeeper policy structure

Each Gatekeeper policy has two parts:

- a [ConstraintTemplate](https://open-policy-agent.github.io/gatekeeper/website/docs/constrainttemplates/)
- a [Constraint](https://open-policy-agent.github.io/gatekeeper/website/docs/howto/#constraints)

`ConstraintTemplate` files in [gktemplates](gktemplates) are generated from:

- a template file in `gktemplates-src/<policy>/<policy>.tmpl`
- a Rego program in `gktemplates-src/<policy>/src.rego`

Do not edit generated files in [gktemplates](gktemplates) directly.

### Create a new Gatekeeper policy

1. Create a new directory under [gktemplates-src](gktemplates-src).
2. Add `<policy>.tmpl` with the non-Rego portion of the template.
3. Add `src.rego` with the policy logic.
4. Add `src_test.rego` with unit tests for the Rego.
5. Add the matching `Constraint` YAML under [gkconstraints](gkconstraints).

Example `ConstraintTemplate` source:

```yaml
apiVersion: templates.gatekeeper.sh/v1
kind: ConstraintTemplate
metadata:
  name: arodenyprivilegednamespace
  annotations:
    metadata.gatekeeper.sh/title: "Privileged Namespace"
    metadata.gatekeeper.sh/version: 1.0.0
    description: >-
      Disallows creating, updating or deleting resources in privileged namespaces.
spec:
  crd:
    spec:
      names:
        kind: ARODenyPrivilegedNamespace
      validation:
        openAPIV3Schema:
          type: object
          description: >-
            Disallows creating, updating or deleting resources in privileged namespaces.
  targets:
    - target: admission.k8s.gatekeeper.sh
      rego: |
{{ file.Read "gktemplates-src/aro-deny-privileged-namespace/src.rego" | strings.Indent 8 | strings.TrimSuffix "\n" }}
      libs:
        - |
{{ file.Read "gktemplates-src/library/common.rego" | strings.Indent 10 | strings.TrimSuffix "\n" }}
```

Example `src.rego`:

```rego
package arodenyprivilegednamespace

import data.lib.common.is_priv_namespace
import data.lib.common.is_exempted_account
import data.lib.common.get_username

violation[{"msg": msg}] {
  ns := input.review.object.metadata.namespace
  is_priv_namespace(ns)
  not is_exempted_account(input.review)
  username := get_username(input.review)
  msg := sprintf("user %v not allowed to operate in namespace %v", [username, ns])
}
```

Example `src_test.rego`:

```rego
package arodenyprivilegednamespace

test_input_allowed_ns {
  input := {"review": input_ns(input_allowed_ns)}
  results := violation with input as input
  count(results) == 0
}

test_input_disallowed_ns1 {
  input := {"review": input_ns(input_disallowed_ns1)}
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
input_disallowed_ns1 = "openshift-etcd"
```

Example `Constraint`:

```yaml
apiVersion: constraints.gatekeeper.sh/v1beta1
kind: ARODenyPrivilegedNamespace
metadata:
  name: aro-privileged-namespace-deny
spec:
  enforcementAction: {{.Enforcement}}
  match:
    kinds:
      - apiGroups: [""]
        kinds: [
        "Pod",
        "Secret",
        "Service",
        "ServiceAccount",
        "ReplicationController",
        "ResourceQuota",
        ]
      - apiGroups: ["apps"]
        kinds: ["Deployment", "ReplicaSet", "StatefulSet", "DaemonSet"]
      - apiGroups: ["batch"]
        kinds: ["Job", "CronJob"]
      - apiGroups: ["rbac.authorization.k8s.io"]
        kinds: ["Role", "RoleBinding"]
      - apiGroups: ["policy"]
        kinds: ["PodDisruptionBudget"]
```

Make sure the constraint file name matches `.metadata.name`, because the file name is also used as the per-policy operator flag suffix.

### Test Gatekeeper Rego source

Install the OPA CLI: https://github.com/open-policy-agent/opa/releases/

Run OPA unit tests for a policy:

```sh
opa test ../library/common.rego *.rego
```

### Generate Gatekeeper templates

Install `gomplate`: https://docs.gomplate.ca/installing/

Generate all templates:

```sh
cd pkg/operator/controllers/guardrails/policies
./scripts/generate.sh
```

Generate one template:

```sh
cd pkg/operator/controllers/guardrails/policies
./scripts/generate.sh aro-deny-machine-config
```

### Test Gatekeeper policy end-to-end with gator

Create a `Suite` and test inputs under the policy directory. The existing [scripts/test.sh](scripts/test.sh) helper:

- runs `opa test`
- expands `{{.Enforcement}}` to `deny` for test constraints
- runs `gator verify`

Run all Gatekeeper policy tests:

```sh
cd pkg/operator/controllers/guardrails/policies
./scripts/test.sh
```

Run one Gatekeeper policy:

```sh
cd pkg/operator/controllers/guardrails/policies
./scripts/test.sh aro-deny-machine-config aro-machine-config-deny.yaml
```

You can also run `gator verify` directly after preparing the expanded constraints.

When a policy depends on request operation or old/new object state, use a mocked `AdmissionReview` input in gator. Example:

```yaml
kind: AdmissionReview
apiVersion: admission.k8s.io/v1
request:
  uid: d700ab7f-8f42-45ff-83f5-782c739806d9
  operation: UPDATE
  userInfo:
    username: kube-review
    uid: 45884572-1cab-49e5-be4c-1d2eb0299776
  object:
    kind: MachineConfig
    apiVersion: machineconfiguration.openshift.io/v1
    metadata:
      name: 99-worker-generated-crio-fake
  oldObject:
    kind: MachineConfig
    apiVersion: machineconfiguration.openshift.io/v1
    metadata:
      name: 99-worker-generated-crio-seccomp-use-default
  dryRun: true
```

The `admr-gen` tool can help generate mocked admission review payloads: https://github.com/ArrisLee/admr-gen

### Validate Gatekeeper on a dev cluster

Use this path only for clusters **below 4.17**.

Set up the dev RP environment first: https://github.com/Azure/ARO-RP/blob/master/docs/deploy-development-rp.md

Example flow:

```sh
CLUSTER=my-test-cluster go run ./hack/cluster create
oc scale -n openshift-azure-operator deployment/aro-operator-master --replicas=0
CLUSTER=my-test-cluster go run -tags aro,containers_image_openpgp ./cmd/aro operator master
```

Enable managed guardrails in this order:

```sh
oc patch cluster.aro.openshift.io cluster --type json -p '[{"op":"replace","path":"/spec/operatorflags/aro.guardrails.deploy.managed","value":"true"}]'
oc patch cluster.aro.openshift.io cluster --type json -p '[{"op":"replace","path":"/spec/operatorflags/aro.guardrails.enabled","value":"true"}]'
```

That order matters.

Verify Gatekeeper pods:

```sh
oc get all -n openshift-azure-guardrails
```

Verify templates:

```sh
oc get constrainttemplate
```

Enable a Gatekeeper policy and set enforcement:

```sh
oc patch cluster.aro.openshift.io cluster --type json -p '[{"op":"replace","path":"/spec/operatorflags/aro.guardrails.policies.aro-machines-deny.managed","value":"true"}]'
oc patch cluster.aro.openshift.io cluster --type json -p '[{"op":"replace","path":"/spec/operatorflags/aro.guardrails.policies.aro-machines-deny.enforcement","value":"deny"}]'
```

Verify the constraint:

```sh
oc get constraint
```

## VAP approach (4.17+)

### What is deployed

For clusters 4.17 and later, the controller does **not** deploy Gatekeeper. Instead it:

1. detects and removes old Gatekeeper resources left behind by upgraded clusters
2. reads policy YAML from [../policies-vap/vap](../policies-vap/vap)
3. reads binding templates from [../policies-vap/vap-binding](../policies-vap/vap-binding)
4. reconciles `ValidatingAdmissionPolicy` and `ValidatingAdmissionPolicyBinding` resources
5. periodically re-applies them via the reconciliation ticker

### Current VAP-backed policies

Today the VAP tree contains:

- `aro-machines-deny`
- `aro-machine-config-deny`
- `aro-privileged-namespace-deny`

### VAP policy structure

Each VAP policy is split into:

- one static `ValidatingAdmissionPolicy` YAML file in [../policies-vap/vap](../policies-vap/vap)
- one `ValidatingAdmissionPolicyBinding` template in [../policies-vap/vap-binding](../policies-vap/vap-binding)

Unlike Gatekeeper templates, VAP policies are authored directly as Kubernetes resources. There is no Rego layer and no `gomplate` generation step for the policy itself.

Bindings are still templated because the operator injects the `validationActions` field from the per-policy enforcement flag.

Example binding template:

```yaml
apiVersion: admissionregistration.k8s.io/v1
kind: ValidatingAdmissionPolicyBinding
metadata:
  name: aro-machines-deny-binding
spec:
  policyName: aro-machines-deny
  validationActions:
  - {{.ValidationAction}}
  matchResources:
    namespaceSelector:
      matchExpressions:
      - key: kubernetes.io/metadata.name
        operator: In
        values: ["openshift-machine-api"]
```

### VAP enforcement mapping

The operator maps the Gatekeeper-style policy flag to a VAP `validationAction` as follows:

- `deny` -> `Deny`
- `warn` -> `Warn`
- `dryrun` or `audit` -> `Audit`
- anything else -> `Warn`

### Create a new VAP policy

1. Add `<policy>.yaml` under [../policies-vap/vap](../policies-vap/vap).
2. Add `<policy>-binding.yaml` under [../policies-vap/vap-binding](../policies-vap/vap-binding).
3. Make sure the base file name matches the operator flag suffix.
4. Add or update unit tests in [../guardrails_controller_test.go](../guardrails_controller_test.go).
5. Add or update e2e coverage in [../../../../../test/e2e/operator.go](../../../../../test/e2e/operator.go).

The controller discovers VAP policies by reading the embedded files in the `vap` directory, so new files are picked up automatically as long as they follow the existing naming convention.

### How VAP reconciliation works

- policy YAML is decoded to `unstructured.Unstructured`
- binding templates are rendered with the selected `ValidationAction`
- both resources are applied via `dynamichelper.Ensure()`
- because these are native Kubernetes resources, `Ensure()` uses server-side apply

### Test VAP policies

There is currently no parallel Rego/gator flow for VAP. VAP coverage is maintained through:

- unit tests in [../guardrails_controller_test.go](../guardrails_controller_test.go)
- controller logic tests covering creation, deletion, enforcement mapping, and upgrade cleanup
- e2e coverage in [../../../../../test/e2e/operator.go](../../../../../test/e2e/operator.go)

Recent e2e coverage includes:

- policy and binding existence checks
- managed flag toggle behavior

### Validate VAP on a dev cluster

Use this path only for clusters **4.17 and later**, where VAP is natively supported.

Run the operator locally as above, then enable managed guardrails:

```sh
oc patch cluster.aro.openshift.io cluster --type json -p '[{"op":"replace","path":"/spec/operatorflags/aro.guardrails.deploy.managed","value":"true"}]'
oc patch cluster.aro.openshift.io cluster --type json -p '[{"op":"replace","path":"/spec/operatorflags/aro.guardrails.enabled","value":"true"}]'
```

Enable one VAP policy:

```sh
oc patch cluster.aro.openshift.io cluster --type json -p '[{"op":"replace","path":"/spec/operatorflags/aro.guardrails.policies.aro-machine-config-deny.managed","value":"true"}]'
oc patch cluster.aro.openshift.io cluster --type json -p '[{"op":"replace","path":"/spec/operatorflags/aro.guardrails.policies.aro-machine-config-deny.enforcement","value":"deny"}]'
```

Verify policies and bindings:

```sh
oc get validatingadmissionpolicies.admissionregistration.k8s.io
oc get validatingadmissionpolicybindings.admissionregistration.k8s.io
```

For clusters upgraded from a pre-4.17 version, the controller should also remove the old Gatekeeper resources during reconciliation.