package etchosts

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"bytes"
	"encoding/json"
	"fmt"
	"text/template"

	"github.com/Azure/go-autorest/autorest/to"
	ign3types "github.com/coreos/ignition/v2/config/v3_2/types"
	"github.com/pkg/errors"
	"github.com/vincent-petithory/dataurl"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	mcv1 "github.com/openshift/machine-config-operator/pkg/apis/machineconfiguration.openshift.io/v1"
)

const (
	configFileName = "aro.conf"
	tempFileName   = "aro.tmp"
	unitFileName   = "aro-etchosts-resolver.service"
	scriptFileName = "aro-etchosts-resolver.sh"
	scriptMarker   = "openshift-aro-etchosts-resolver"
)

type etcHostsAROConfTemplateData struct {
	ClusterDomain            string
	APIIntIP                 string
	GatewayDomains           []string
	GatewayPrivateEndpointIP string
}

type etcHostsAROScriptTemplateData struct {
	ConfigFileName string
	TempFileName   string
	UnitFileName   string
	ScriptFileName string
	ScriptMarker   string
}

var aroConfTemplate = template.Must(template.New("etchosts").Parse(`{{ .APIIntIP }}	api.{{ .ClusterDomain }} api-int.{{ .ClusterDomain }}
{{ $.GatewayPrivateEndpointIP }}	{{ range $i, $GatewayDomain := .GatewayDomains }}{{ if (gt $i 0) }} {{ end }}{{ $GatewayDomain }}{{ end }}
`))

var aroScriptTemplate = template.Must(template.New("etchostscript").Parse(`#!/bin/bash
set -uo pipefail

trap 'jobs -p | xargs kill || true; wait; exit 0' TERM

OPENSHIFT_MARKER="{{ .ScriptMarker }}"
HOSTS_FILE="/etc/hosts"
CONFIG_FILE="/etc/hosts.d/{{ .ConfigFileName }}"
TEMP_FILE="/etc/hosts.d/{{ .TempFileName }}"

# Make a temporary file with the old hosts file's data.
if ! cp -f "${HOSTS_FILE}" "${TEMP_FILE}"; then
  echo "Failed to preserve hosts file. Exiting."
  exit 1
fi

if ! sed --silent "/# ${OPENSHIFT_MARKER}/d; w ${TEMP_FILE}" "${HOSTS_FILE}"; then
  # Only continue rebuilding the hosts entries if its original content is preserved
  sleep 60 & wait
  continue
fi

while IFS= read -r line; do
    echo "${line} # ${OPENSHIFT_MARKER}" >> "${TEMP_FILE}"
done < "${CONFIG_FILE}"

# Replace /etc/hosts with our modified version if needed
cmp "${TEMP_FILE}" "${HOSTS_FILE}" || cp -f "${TEMP_FILE}" "${HOSTS_FILE}"
# TEMP_FILE is not removed to avoid file create/delete and attributes copy churn
`))

var aroUnitTemplate = template.Must(template.New("etchostservice").Parse(`[Unit]
Description=One shot service that appends static domains to etchosts
Before=network-online.target

[Service]
# ExecStart will copy the hosts defined in /etc/hosts.d/aro.conf to /etc/hosts
ExecStart=/bin/bash /usr/local/bin/{{ .ScriptFileName }}

[Install]
WantedBy=multi-user.target
`))

func GenerateEtcHostsAROConf(clusterDomain string, apiIntIP string, gatewayDomains []string, gatewayPrivateEndpointIP string) ([]byte, error) {
	buf := &bytes.Buffer{}
	templateData := etcHostsAROConfTemplateData{
		ClusterDomain:            clusterDomain,
		APIIntIP:                 apiIntIP,
		GatewayDomains:           gatewayDomains,
		GatewayPrivateEndpointIP: gatewayPrivateEndpointIP,
	}

	if err := aroConfTemplate.Execute(buf, templateData); err != nil {
		return nil, errors.Wrap(err, "failed to generate "+configFileName+" from template")
	}

	return buf.Bytes(), nil
}

func GenerateEtcHostsAROScript() ([]byte, error) {
	buf := &bytes.Buffer{}
	templateData := etcHostsAROScriptTemplateData{
		ConfigFileName: configFileName,
		TempFileName:   tempFileName,
		UnitFileName:   unitFileName,
		ScriptFileName: scriptFileName,
		ScriptMarker:   scriptMarker,
	}

	if err := aroScriptTemplate.Execute(buf, templateData); err != nil {
		return nil, errors.Wrap(err, "failed to generate "+scriptFileName+" from template")
	}

	return buf.Bytes(), nil
}

func GenerateEtcHostsAROUnit() (string, error) {
	buf := &bytes.Buffer{}
	templateData := etcHostsAROScriptTemplateData{
		ConfigFileName: configFileName,
		TempFileName:   tempFileName,
		UnitFileName:   unitFileName,
		ScriptFileName: scriptFileName,
		ScriptMarker:   scriptMarker,
	}

	if err := aroUnitTemplate.Execute(buf, templateData); err != nil {
		return "", errors.Wrap(err, "failed to generate "+unitFileName+" from template")
	}

	return buf.String(), nil
}

func EtcHostsIgnitionConfig(clusterDomain string, apiIntIP string, gatewayDomains []string, gatewayPrivateEndpointIP string) (*ign3types.Config, error) {
	aroconf, err := GenerateEtcHostsAROConf(clusterDomain, apiIntIP, gatewayDomains, gatewayPrivateEndpointIP)
	if err != nil {
		return nil, errors.Wrap(err, "failed to generate addtional hosts for etc hosts")
	}

	aroscript, err := GenerateEtcHostsAROScript()
	if err != nil {
		return nil, errors.Wrap(err, "failed to generate template")
	}

	arounit, err := GenerateEtcHostsAROUnit()
	if err != nil {
		return nil, errors.Wrap(err, "failed to generate template")
	}

	ign := &ign3types.Config{
		Ignition: ign3types.Ignition{
			Version: ign3types.MaxVersion.String(),
		},
		Storage: ign3types.Storage{
			Files: []ign3types.File{
				{
					Node: ign3types.Node{
						Path:      "/etc/hosts.d/" + configFileName,
						Overwrite: to.BoolPtr(true),
						User: ign3types.NodeUser{
							Name: to.StringPtr("root"),
						},
					},
					FileEmbedded1: ign3types.FileEmbedded1{
						Contents: ign3types.Resource{
							Source: to.StringPtr(dataurl.EncodeBytes(aroconf)),
						},
						Mode: to.IntPtr(0644),
					},
				},
				{
					Node: ign3types.Node{
						Overwrite: to.BoolPtr(true),
						Path:      "/usr/local/bin/" + scriptFileName,
						User: ign3types.NodeUser{
							Name: to.StringPtr("root"),
						},
					},
					FileEmbedded1: ign3types.FileEmbedded1{
						Contents: ign3types.Resource{
							Source: to.StringPtr(dataurl.EncodeBytes(aroscript)),
						},
						Mode: to.IntPtr(0744),
					},
				},
			},
		},
		Systemd: ign3types.Systemd{
			Units: []ign3types.Unit{
				{
					Contents: &arounit,
					Enabled:  to.BoolPtr(true),
					Name:     unitFileName,
				},
			},
		},
	}

	return ign, nil
}

func EtcHostsMachineConfig(clusterDomain string, apiIntIP string, gatewayDomains []string, gatewayPrivateEndpointIP string, role string) (*mcv1.MachineConfig, error) {
	ignConfig, err := EtcHostsIgnitionConfig(clusterDomain, apiIntIP, gatewayDomains, gatewayPrivateEndpointIP)
	if err != nil {
		return nil, err
	}

	b, err := json.Marshal(ignConfig)
	if err != nil {
		return nil, err
	}

	// canonicalise the machineconfig payload the same way as MCO
	var i interface{}
	err = json.Unmarshal(b, &i)
	if err != nil {
		return nil, err
	}

	rawExt := runtime.RawExtension{}
	rawExt.Raw, err = json.Marshal(i)
	if err != nil {
		return nil, err
	}

	return &mcv1.MachineConfig{
		TypeMeta: metav1.TypeMeta{
			APIVersion: mcv1.SchemeGroupVersion.String(),
			Kind:       "MachineConfig",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: fmt.Sprintf("99-%s-aro-etc-hosts-gateway-domains", role),
			Labels: map[string]string{
				"machineconfiguration.openshift.io/role": role,
			},
		},
		Spec: mcv1.MachineConfigSpec{
			Config: rawExt,
		},
	}, nil
}
