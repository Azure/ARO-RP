package dnsmasq

import (
	"bytes"
	"fmt"
	"text/template"

	ignutil "github.com/coreos/ignition/v2/config/util"
	igntypes "github.com/coreos/ignition/v2/config/v3_1/types"
	"github.com/openshift/installer/pkg/asset/ignition"
	mcfgv1 "github.com/openshift/machine-config-operator/pkg/apis/machineconfiguration.openshift.io/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var t = template.Must(template.New("").Parse(`

{{ define "dnsmasq.conf" }}
resolv-file=/etc/resolv.conf.dnsmasq
strict-order
address=/api.{{ .ClusterDomain }}/{{ .APIIntIP }}
address=/api-int.{{ .ClusterDomain }}/{{ .APIIntIP }}
address=/.apps.{{ .ClusterDomain }}/{{ .IngressIP }}
user=dnsmasq
group=dnsmasq
no-hosts
cache-size=0
{{ end }}

{{ define "dnsmasq.service" }}
[Unit]
Description=DNS caching server.
After=network-online.target
Before=bootkube.service

[Service]
ExecStartPre=/bin/cp /etc/resolv.conf /etc/resolv.conf.dnsmasq
ExecStartPre=/bin/bash -c '/bin/sed -ni -e "/^nameserver /!p; \\$$a nameserver $$(hostname -I)" /etc/resolv.conf'
ExecStart=/usr/sbin/dnsmasq -k
ExecStop=/bin/mv /etc/resolv.conf.dnsmasq /etc/resolv.conf
Restart=always

[Install]
WantedBy=multi-user.target
{{ end }}

`))

func config(clusterDomain, apiIntIP, ingressIP string) ([]byte, error) {
	buf := &bytes.Buffer{}

	err := t.ExecuteTemplate(buf, "dnsmasq.conf", &struct {
		ClusterDomain string
		APIIntIP      string
		IngressIP     string
	}{
		ClusterDomain: clusterDomain,
		APIIntIP:      apiIntIP,
		IngressIP:     ingressIP,
	})
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func service() (string, error) {
	buf := &bytes.Buffer{}

	err := t.ExecuteTemplate(buf, "dnsmasq.service", nil)
	if err != nil {
		return "", err
	}

	return buf.String(), nil
}

func IgnitionConfig(clusterDomain, apiIntIP, ingressIP string) (*igntypes.Config, error) {
	service, err := service()
	if err != nil {
		return nil, err
	}

	config, err := config(clusterDomain, apiIntIP, ingressIP)
	if err != nil {
		return nil, err
	}

	return &igntypes.Config{
		Ignition: igntypes.Ignition{
			Version: igntypes.MaxVersion.String(),
		},
		Storage: igntypes.Storage{
			Files: []igntypes.File{
				ignition.FileFromBytes("/etc/dnsmasq.conf", "root", 0644, config),
			},
		},
		Systemd: igntypes.Systemd{
			Units: []igntypes.Unit{
				{
					Contents: &service,
					Enabled:  ignutil.BoolToPtr(true),
					Name:     "dnsmasq.service",
				},
			},
		},
	}, nil
}

func MachineConfig(clusterDomain, apiIntIP, ingressIP, role string) (*mcfgv1.MachineConfig, error) {
	ignConfig, err := IgnitionConfig(clusterDomain, apiIntIP, ingressIP)
	if err != nil {
		return nil, err
	}

	rawExt, err := ignition.ConvertToRawExtension(*ignConfig)
	if err != nil {
		return nil, err
	}

	return &mcfgv1.MachineConfig{
		TypeMeta: metav1.TypeMeta{
			APIVersion: mcfgv1.SchemeGroupVersion.String(),
			Kind:       "MachineConfig",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: fmt.Sprintf("99-%s-aro-dns", role),
			Labels: map[string]string{
				"machineconfiguration.openshift.io/role": role,
			},
		},
		Spec: mcfgv1.MachineConfigSpec{
			Config: rawExt,
		},
	}, nil
}
