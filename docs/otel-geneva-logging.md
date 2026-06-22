# Geneva Logging OTEL Developer Guide

This guide captures how OTEL collector behavior is wired in ARO, how gateway targeting is derived, and how to roll out collector changes via operator flags (including MIMO operator-flags updates).

## Ownership and code locations

- Controller: `pkg/operator/controllers/genevalogging/`
- OTEL template: `pkg/operator/controllers/genevalogging/staticfiles/otel-config.yaml.tmpl`
- OTEL template renderer: `pkg/operator/controllers/genevalogging/otel_config_template.go`
- Operator flags: `pkg/operator/flags.go`
- Gateway fields population: `pkg/operator/deploy/deploy.go`

## Gateway dependency and endpoint resolution

The OTEL daemonsets are created only when `spec.gatewayPrivateEndpointIP` is populated and valid. Until then, the controller still creates OTEL config resources and waits for gateway readiness.

When gateway is enabled and endpoint data is present, deploy logic sets:

- `spec.gatewayPrivateEndpointIP`
- `spec.gatewayTelemetryDomain` (formatted as `telemetry.<location>.<appSuffix>`)

Collector endpoint selection:

1. If `gatewayTelemetryDomain` exists: use `<gatewayTelemetryDomain>:4317` and add a host alias to `gatewayPrivateEndpointIP`.
2. Otherwise: use `<gatewayPrivateEndpointIP>:4317`.

The controller always creates the OTEL config ConfigMap first (`config.yaml`, `master-config.yaml`, `worker-config.yaml`) and only creates daemonsets once gateway endpoint data is ready.

## Current OTEL log shape

Top-level fields emitted for log source identification:

- `node`
- `namespace`
- `pod`
- `container`
- `source_name`
- `EventName`

Raw payload retention:

- `raw_json_body` contains the full original log body.

Worker collectors do not include the audit receiver; audit logs are collected on the master/control-plane collector config.

## OTEL operator flags

| Flag | Meaning | Default |
| --- | --- | --- |
| `aro.genevalogging.enabled` | Enables Geneva logging behavior. | `true` |
| `aro.genevalogging.otel.profile` | Global profile (`max-logs`, `reduced-logs`, `minimal-logs`). | `minimal-logs` |
| `aro.genevalogging.otel.master.profile` | Optional master override. | unset |
| `aro.genevalogging.otel.worker.profile` | Optional worker override. | unset |

## Rollout paths

### Admin Update (single-cluster)

PATCH admin cluster with `operatorFlags` and optional `operatorFlagsMergeStrategy`:

- `merge` (default): overlay provided flags on current cluster flags
- `reset`: reset to defaults, then overlay provided flags

### MIMO (fleet rollout)

MIMO task ID for operator flags update:

- `b41749fc-af26-4ab7-b5a1-e03f3ee4cba6` (`OPERATOR_FLAGS_UPDATE_ID`)

Use this task in manifests/schedules to roll out OTEL flag updates across selected clusters.
