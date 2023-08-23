package dnsmasq

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"text/template"

	"github.com/Azure/go-autorest/autorest/to"
	ign2types "github.com/coreos/ignition/config/v2_2/types"
	mcv1 "github.com/openshift/machine-config-operator/pkg/apis/machineconfiguration.openshift.io/v1"
	"github.com/vincent-petithory/dataurl"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

const (
	configFileName    = "dnsmasq.conf"
	unitFileName      = "dnsmasq.service"
	prescriptFileName = "aro-dnsmasq-pre.sh"
)

func config(clusterDomain, apiIntIP, ingressIP string, gatewayDomains []string, gatewayPrivateEndpointIP string) ([]byte, error) {
	t := template.Must(template.New(configFileName).Parse(configFile))
	buf := &bytes.Buffer{}

	err := t.ExecuteTemplate(buf, configFileName, &struct {
		ClusterDomain            string
		APIIntIP                 string
		IngressIP                string
		GatewayDomains           []string
		GatewayPrivateEndpointIP string
	}{
		ClusterDomain:            clusterDomain,
		APIIntIP:                 apiIntIP,
		IngressIP:                ingressIP,
		GatewayDomains:           gatewayDomains,
		GatewayPrivateEndpointIP: gatewayPrivateEndpointIP,
	})
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func service() (string, error) {
	t := template.Must(template.New(unitFileName).Parse(unitFile))
	buf := &bytes.Buffer{}

	err := t.ExecuteTemplate(buf, unitFileName, nil)
	if err != nil {
		return "", err
	}

	return buf.String(), nil
}

func startpre() ([]byte, error) {
	t := template.Must(template.New(prescriptFileName).Parse(preScriptFile))
	buf := &bytes.Buffer{}

	err := t.ExecuteTemplate(buf, prescriptFileName, nil)
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func ignition2Config(clusterDomain, apiIntIP, ingressIP string, gatewayDomains []string, gatewayPrivateEndpointIP string, restartDnsmasq bool) (*ign2types.Config, error) {
	service, err := service()
	if err != nil {
		return nil, err
	}

	config, err := config(clusterDomain, apiIntIP, ingressIP, gatewayDomains, gatewayPrivateEndpointIP)
	if err != nil {
		return nil, err
	}

	startpre, err := startpre()
	if err != nil {
		return nil, err
	}

	ign := &ign2types.Config{
		Ignition: ign2types.Ignition{
			Version: ign2types.MaxVersion.String(),
		},
		Storage: ign2types.Storage{
			Files: []ign2types.File{
				{
					Node: ign2types.Node{
						Filesystem: "root",
						Overwrite:  to.BoolPtr(true),
						Path:       "/etc/" + configFileName,
						User: &ign2types.NodeUser{
							Name: "root",
						},
					},
					FileEmbedded1: ign2types.FileEmbedded1{
						Contents: ign2types.FileContents{
							Source: dataurl.EncodeBytes(config),
						},
						Mode: to.IntPtr(0644),
					},
				},
				{
					Node: ign2types.Node{
						Filesystem: "root",
						Overwrite:  to.BoolPtr(true),
						Path:       "/usr/local/bin/" + prescriptFileName,
						User: &ign2types.NodeUser{
							Name: "root",
						},
					},
					FileEmbedded1: ign2types.FileEmbedded1{
						Contents: ign2types.FileContents{
							Source: dataurl.EncodeBytes(startpre),
						},
						Mode: to.IntPtr(0744),
					},
				},
			},
		},
		Systemd: ign2types.Systemd{
			Units: []ign2types.Unit{
				{
					Contents: service,
					Enabled:  to.BoolPtr(true),
					Name:     unitFileName,
				},
			},
		},
	}

	if restartDnsmasq {
		restartDnsmasqScript, err := nmDispatcherRestartDnsmasq()
		if err != nil {
			return nil, err
		}

		ign.Storage.Files = append(ign.Storage.Files, restartScriptIgnFile(restartDnsmasqScript))
	}

	return ign, nil
}

func dnsmasqMachineConfig(clusterDomain, apiIntIP, ingressIP, role string, gatewayDomains []string, gatewayPrivateEndpointIP string, restartDnsmasq bool) (*mcv1.MachineConfig, error) {
	ignConfig, err := ignition2Config(clusterDomain, apiIntIP, ingressIP, gatewayDomains, gatewayPrivateEndpointIP, restartDnsmasq)
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
			Name: fmt.Sprintf("99-%s-aro-dns", role),
			Labels: map[string]string{
				"machineconfiguration.openshift.io/role": role,
			},
		},
		Spec: mcv1.MachineConfigSpec{
			Config: rawExt,
		},
	}, nil
}
