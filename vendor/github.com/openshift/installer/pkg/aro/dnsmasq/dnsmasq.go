package dnsmasq

import (
	"bytes"
	"encoding/json"
	"fmt"
	"text/template"

	ign2types "github.com/coreos/ignition/config/v2_2/types"
	ignutil "github.com/coreos/ignition/v2/config/util"
	ign3types "github.com/coreos/ignition/v2/config/v3_2/types"
	mcfgv1 "github.com/openshift/machine-config-operator/pkg/apis/machineconfiguration.openshift.io/v1"
	"github.com/vincent-petithory/dataurl"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

var t = template.Must(template.New("").Parse(`

{{ define "dnsmasq.conf" }}
resolv-file=/etc/resolv.conf.dnsmasq
strict-order
address=/api.{{ .ClusterDomain }}/{{ .APIIntIP }}
address=/api-int.{{ .ClusterDomain }}/{{ .APIIntIP }}
address=/.apps.{{ .ClusterDomain }}/{{ .IngressIP }}
{{- range $GatewayDomain := .GatewayDomains }}
address=/{{ $GatewayDomain }}/{{ $.GatewayPrivateEndpointIP }}
{{- end }}
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
# ExecStartPre will create a copy of the customer current resolv.conf file and make it upstream DNS.
# This file is a product of user DNS settings on the VNET. We will replace this file to point to
# dnsmasq instance on the node. dnsmasq will inject certain dns records we need and forward rest of the queries to
# resolv.conf.dnsmasq upstream customer dns.
ExecStartPre=/bin/bash /usr/local/bin/aro-dnsmasq-pre.sh
ExecStart=/usr/sbin/dnsmasq -k
ExecStopPost=/bin/bash -c '/bin/mv /etc/resolv.conf.dnsmasq /etc/resolv.conf; /usr/sbin/restorecon /etc/resolv.conf'
Restart=always

[Install]
WantedBy=multi-user.target
{{ end }}

{{ define "aro-dnsmasq-pre.sh" }}
#!/bin/bash
set -euo pipefail

# This bash script is a part of the ARO DnsMasq configuration
# It's deployed as part of the 99-aro-dns-* machine config
# See https://github.com/Azure/ARO-RP

# This file can be rerun and the effect is idempotent, output might change if the DHCP configuration changes

TMPSELFRESOLV=$(mktemp)
TMPNETRESOLV=$(mktemp)

echo "# Generated for dnsmasq.service - should point to self" > $TMPSELFRESOLV
echo "# Generated for dnsmasq.service - should contain DHCP configured DNS" > $TMPNETRESOLV

if nmcli device show br-ex; then
    echo "OVN mode - br-ex device exists"
    #getting DNS search strings
    SEARCH_RAW=$(nmcli --get IP4.DOMAIN device show br-ex)
    #getting DNS servers
    NAMESERVER_RAW=$(nmcli --get IP4.DNS device show br-ex | tr -s " | " "\n")
    LOCAL_IPS_RAW=$(nmcli --get IP4.ADDRESS device show br-ex)
else
    NETDEV=$(nmcli --get device connection show --active | head -n 1) #there should be only one active device
    echo "OVS SDN mode - br-ex not found, using device $NETDEV"
    SEARCH_RAW=$(nmcli --get IP4.DOMAIN device show $NETDEV)
    NAMESERVER_RAW=$(nmcli --get IP4.DNS device show $NETDEV | tr -s " | " "\n")
    LOCAL_IPS_RAW=$(nmcli --get IP4.ADDRESS device show $NETDEV)
fi

#search line
echo "search $SEARCH_RAW" | tr '\n' ' ' >> $TMPNETRESOLV
echo "" >> $TMPNETRESOLV
echo "search $SEARCH_RAW" | tr '\n' ' ' >> $TMPSELFRESOLV
echo "" >> $TMPSELFRESOLV

#nameservers as separate lines
echo "$NAMESERVER_RAW" | while read -r line
do
    echo "nameserver $line" >> $TMPNETRESOLV
done
# device IPs are returned in address/mask format
echo "$LOCAL_IPS_RAW" | while read -r line
do
    echo "nameserver $line" | cut -d'/' -f 1 >> $TMPSELFRESOLV
done

# done, copying files to destination locations and cleaning up
/bin/cp $TMPNETRESOLV /etc/resolv.conf.dnsmasq
chmod 0744 /etc/resolv.conf.dnsmasq
/bin/cp $TMPSELFRESOLV /etc/resolv.conf
/usr/sbin/restorecon /etc/resolv.conf
/bin/rm $TMPNETRESOLV
/bin/rm $TMPSELFRESOLV
{{ end }}
`))

func config(clusterDomain, apiIntIP, ingressIP string, gatewayDomains []string, gatewayPrivateEndpointIP string) ([]byte, error) {
	buf := &bytes.Buffer{}

	err := t.ExecuteTemplate(buf, "dnsmasq.conf", &struct {
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
	buf := &bytes.Buffer{}

	err := t.ExecuteTemplate(buf, "dnsmasq.service", nil)
	if err != nil {
		return "", err
	}

	return buf.String(), nil
}

func startpre() ([]byte, error) {
	buf := &bytes.Buffer{}

	err := t.ExecuteTemplate(buf, "aro-dnsmasq-pre.sh", nil)
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func Ignition2Config(clusterDomain, apiIntIP, ingressIP string, gatewayDomains []string, gatewayPrivateEndpointIP string) (*ign2types.Config, error) {
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

	return &ign2types.Config{
		Ignition: ign2types.Ignition{
			Version: ign2types.MaxVersion.String(),
		},
		Storage: ign2types.Storage{
			Files: []ign2types.File{
				{
					Node: ign2types.Node{
						Filesystem: "root",
						Overwrite:  ignutil.BoolToPtr(true),
						Path:       "/etc/dnsmasq.conf",
						User: &ign2types.NodeUser{
							Name: "root",
						},
					},
					FileEmbedded1: ign2types.FileEmbedded1{
						Contents: ign2types.FileContents{
							Source: dataurl.EncodeBytes(config),
						},
						Mode: ignutil.IntToPtr(0644),
					},
				},
				{
					Node: ign2types.Node{
						Filesystem: "root",
						Overwrite:  ignutil.BoolToPtr(true),
						Path:       "/usr/local/bin/aro-dnsmasq-pre.sh",
						User: &ign2types.NodeUser{
							Name: "root",
						},
					},
					FileEmbedded1: ign2types.FileEmbedded1{
						Contents: ign2types.FileContents{
							Source: dataurl.EncodeBytes(startpre),
						},
						Mode: ignutil.IntToPtr(0744),
					},
				},
			},
		},
		Systemd: ign2types.Systemd{
			Units: []ign2types.Unit{
				{
					Contents: service,
					Enabled:  ignutil.BoolToPtr(true),
					Name:     "dnsmasq.service",
				},
			},
		},
	}, nil
}

func Ignition3Config(clusterDomain, apiIntIP, ingressIP string, gatewayDomains []string, gatewayPrivateEndpointIP string) (*ign3types.Config, error) {
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

	return &ign3types.Config{
		Ignition: ign3types.Ignition{
			Version: ign3types.MaxVersion.String(),
		},
		Storage: ign3types.Storage{
			Files: []ign3types.File{
				{
					Node: ign3types.Node{
						Overwrite: ignutil.BoolToPtr(true),
						Path:      "/etc/dnsmasq.conf",
						User: ign3types.NodeUser{
							Name: ignutil.StrToPtr("root"),
						},
					},
					FileEmbedded1: ign3types.FileEmbedded1{
						Contents: ign3types.Resource{
							Source: ignutil.StrToPtr(dataurl.EncodeBytes(config)),
						},
						Mode: ignutil.IntToPtr(0644),
					},
				},
				{
					Node: ign3types.Node{
						Overwrite: ignutil.BoolToPtr(true),
						Path:      "/usr/local/bin/aro-dnsmasq-pre.sh",
						User: ign3types.NodeUser{
							Name: ignutil.StrToPtr("root"),
						},
					},
					FileEmbedded1: ign3types.FileEmbedded1{
						Contents: ign3types.Resource{
							Source: ignutil.StrToPtr(dataurl.EncodeBytes(startpre)),
						},
						Mode: ignutil.IntToPtr(0744),
					},
				},
			},
		},
		Systemd: ign3types.Systemd{
			Units: []ign3types.Unit{
				{
					Contents: &service,
					Enabled:  ignutil.BoolToPtr(true),
					Name:     "dnsmasq.service",
				},
			},
		},
	}, nil
}

func MachineConfig(clusterDomain, apiIntIP, ingressIP, role string, gatewayDomains []string, gatewayPrivateEndpointIP string) (*mcfgv1.MachineConfig, error) {
	ignConfig, err := Ignition2Config(clusterDomain, apiIntIP, ingressIP, gatewayDomains, gatewayPrivateEndpointIP)
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
