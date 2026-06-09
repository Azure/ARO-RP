// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

package genevalogging

import (
	"bytes"
	"fmt"
	"text/template"
)

var otelConfigProfileTemplate = template.Must(template.New("otel-config-profile").Parse(
	`{{- if eq .Profile "` + string(otelProfileHighLogLevel) + `" -}}
{{- .HighLogLevel -}}
{{- else if eq .Profile "` + string(otelProfileMinimalLogs) + `" -}}
{{- .MinimalLogs -}}
{{- else -}}
{{- .ReducedLogs -}}
{{- end -}}`,
))

const otelConfigLogParityStatements = `          - delete_matching_keys(log.attributes, "^_")
          - delete_matching_keys(log.attributes, "^JOB_")
          - delete_matching_keys(log.attributes, "^NM_")
          - delete_matching_keys(log.attributes, "^COREDUMP_")
          - delete_key(log.attributes, "TIMESTAMP")
          - delete_key(log.attributes, "TIMESTAMP_MONOTONIC")
          - delete_key(log.attributes, "SYSLOG_FACILITY")
          - delete_key(log.attributes, "SYSLOG_TIMESTAMP")
          - delete_key(log.attributes, "SYSLOG_PID")
          - delete_key(log.attributes, "MESSAGE_ID")
          - delete_key(log.attributes, "INVOCATION_ID")
          - delete_key(log.attributes, "CPU_USAGE_NSEC")
          - delete_key(log.attributes, "BOOT_ID")
          - delete_key(log.attributes, "PENDING")
          - set(log.body["user_username"], log.body["user"]["username"]) where IsMap(log.body) and IsMap(log.body["user"]) and log.body["user"]["username"] != nil
          - set(log.body["user_uid"], log.body["user"]["uid"]) where IsMap(log.body) and IsMap(log.body["user"]) and log.body["user"]["uid"] != nil
          - set(log.body["user_groups"], log.body["user"]["groups"]) where IsMap(log.body) and IsMap(log.body["user"]) and log.body["user"]["groups"] != nil
          - set(log.body["user_extra"], log.body["user"]["extra"]) where IsMap(log.body) and IsMap(log.body["user"]) and log.body["user"]["extra"] != nil
          - set(log.body["impersonatedUser_username"], log.body["impersonatedUser"]["username"]) where IsMap(log.body) and IsMap(log.body["impersonatedUser"]) and log.body["impersonatedUser"]["username"] != nil
          - set(log.body["impersonatedUser_uid"], log.body["impersonatedUser"]["uid"]) where IsMap(log.body) and IsMap(log.body["impersonatedUser"]) and log.body["impersonatedUser"]["uid"] != nil
          - set(log.body["impersonatedUser_groups"], log.body["impersonatedUser"]["groups"]) where IsMap(log.body) and IsMap(log.body["impersonatedUser"]) and log.body["impersonatedUser"]["groups"] != nil
          - set(log.body["impersonatedUser_extra"], log.body["impersonatedUser"]["extra"]) where IsMap(log.body) and IsMap(log.body["impersonatedUser"]) and log.body["impersonatedUser"]["extra"] != nil
          - set(log.body["responseStatus_code"], log.body["responseStatus"]["code"]) where IsMap(log.body) and IsMap(log.body["responseStatus"]) and log.body["responseStatus"]["code"] != nil
          - set(log.body["responseStatus_reason"], log.body["responseStatus"]["reason"]) where IsMap(log.body) and IsMap(log.body["responseStatus"]) and log.body["responseStatus"]["reason"] != nil
          - set(log.body["responseStatus_status"], log.body["responseStatus"]["status"]) where IsMap(log.body) and IsMap(log.body["responseStatus"]) and log.body["responseStatus"]["status"] != nil
          - set(log.body["responseStatus_message"], log.body["responseStatus"]["message"]) where IsMap(log.body) and IsMap(log.body["responseStatus"]) and log.body["responseStatus"]["message"] != nil
          - set(log.body["responseStatus_metadata"], log.body["responseStatus"]["metadata"]) where IsMap(log.body) and IsMap(log.body["responseStatus"]) and log.body["responseStatus"]["metadata"] != nil
          - set(log.body["objectRef_resource"], log.body["objectRef"]["resource"]) where IsMap(log.body) and IsMap(log.body["objectRef"]) and log.body["objectRef"]["resource"] != nil
          - set(log.body["objectRef_namespace"], log.body["objectRef"]["namespace"]) where IsMap(log.body) and IsMap(log.body["objectRef"]) and log.body["objectRef"]["namespace"] != nil
          - set(log.body["objectRef_name"], log.body["objectRef"]["name"]) where IsMap(log.body) and IsMap(log.body["objectRef"]) and log.body["objectRef"]["name"] != nil
          - set(log.body["objectRef_uid"], log.body["objectRef"]["uid"]) where IsMap(log.body) and IsMap(log.body["objectRef"]) and log.body["objectRef"]["uid"] != nil
          - set(log.body["objectRef_apiGroup"], log.body["objectRef"]["apiGroup"]) where IsMap(log.body) and IsMap(log.body["objectRef"]) and log.body["objectRef"]["apiGroup"] != nil
          - set(log.body["objectRef_apiVersion"], log.body["objectRef"]["apiVersion"]) where IsMap(log.body) and IsMap(log.body["objectRef"]) and log.body["objectRef"]["apiVersion"] != nil
          - set(log.body["objectRef_resourceVersion"], log.body["objectRef"]["resourceVersion"]) where IsMap(log.body) and IsMap(log.body["objectRef"]) and log.body["objectRef"]["resourceVersion"] != nil
          - set(log.body["objectRef_subresource"], log.body["objectRef"]["subresource"]) where IsMap(log.body) and IsMap(log.body["objectRef"]) and log.body["objectRef"]["subresource"] != nil
          - set(log.attributes["CONTAINER"], resource.attributes["k8s.container.name"]) where resource.attributes["k8s.container.name"] != nil
          - set(log.attributes["POD"], resource.attributes["k8s.pod.name"]) where resource.attributes["k8s.pod.name"] != nil
          - set(log.attributes["NAMESPACE"], resource.attributes["k8s.namespace.name"]) where resource.attributes["k8s.namespace.name"] != nil`

func renderOTelConfig(profile otelProfile) (string, error) {
	var profileRendered bytes.Buffer
	err := otelConfigProfileTemplate.Execute(&profileRendered, struct {
		Profile      otelProfile
		HighLogLevel string
		ReducedLogs  string
		MinimalLogs  string
	}{
		Profile:      profile,
		HighLogLevel: otelConfigHighLogLevel,
		ReducedLogs:  otelConfigReducedLogs,
		MinimalLogs:  otelConfigMinimalLogs,
	})
	if err != nil {
		return "", fmt.Errorf("failed to render otel profile %q config template: %w", profile, err)
	}

	sharedTemplate, err := template.New("otel-config-shared").Parse(profileRendered.String())
	if err != nil {
		return "", fmt.Errorf("failed to parse shared otel config template for profile %q: %w", profile, err)
	}

	var rendered bytes.Buffer
	err = sharedTemplate.Execute(&rendered, struct {
		GatewayExporterID   string
		LogParityStatements string
	}{
		GatewayExporterID:   "otlp_grpc/gateway",
		LogParityStatements: otelConfigLogParityStatements,
	})
	if err != nil {
		return "", fmt.Errorf("failed to render shared otel config template for profile %q: %w", profile, err)
	}
	return rendered.String(), nil
}
