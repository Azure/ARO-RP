package dnsv2

import (
	"bytes"
	"text/template"

	"github.com/vincent-petithory/dataurl"

	ign2types "github.com/coreos/ignition/config/v2_2/types"
	ignutil "github.com/coreos/ignition/v2/config/util"
	ign3types "github.com/coreos/ignition/v2/config/v3_2/types"
)

// Potentially add the IP of -n openshift-image-registry Service/image-registry, marked with # openshift-generated-node-resolver
var t = template.Must(template.New("").Parse(`

{{ define "hosts" }}
127.0.0.1   localhost localhost.localdomain localhost4 localhost4.localdomain4
::1         localhost localhost.localdomain localhost6 localhost6.localdomain6
XXX image-registry.openshift-image-registry.svc image-registry.openshift-image-registry.svc.cluster.local # openshift-generated-node-resolver
{{ .APIIntIP }} api.{{ .ClusterDomain }}
{{ .APIIntIP }} api-int.{{ .ClusterDomain }}
# {{ .IngressIP }} *.apps.{{ .ClusterDomain }}/
{{- range $GatewayDomain := .GatewayDomains }}
{{ $.GatewayPrivateEndpointIP }} {{ $GatewayDomain }}
{{- end }}
{{ end }}

{{ define "Corefile" }}
apps.{{ .ClusterDomain }}:53 {
	root /etc/coredns/zones
	file db.apps
}
.:53 {
    bufsize 1232 # https://www.dnsflagday.net/2020/
    errors
    log . {
        class error
    }
    health {
        lameduck 20s
    }
    ready
    prometheus 127.0.0.1:9153
    hosts {
        fallthrough
    }
    forward . /etc/resolv.conf
    reload
}
hostname.bind:53 {
    chaos
}
{{ end }}

{{ define "db.apps" }}
$ORIGIN apps.{{ .ClusterDomain }}.
@	3600 IN	SOA ns1-09.azure-dns.com. azuredns-hostmaster.microsoft.com. 1 3600 300 2419200 300

*        IN A     {{ .IngressIP }}
{{ end }}

{{ define "aro-coredns.service" }}
[Unit]
Description=CoreDNS
RequiredMountsFor=/etc/coredns
After=network-online.target
Before=bootkube.service

[Service]
Environment=PODMAN_SYSTEMD_UNIT=%n
KillMode=mixed
ExecStartPre=cp /etc/hosts.aro /etc/hosts
ExecStart=/usr/bin/podman run --authfile=/var/lib/kubelet/config.json --rm --name aro-coredns -v /etc/coredns/:/etc/coredns/:Z --replace --cgroups=split --init --sdnotify=conmon --net=host --log-driver=journald -d quay.io/openshift-release-dev/ocp-v4.0-art-dev@sha256:19829e1922bbbb02496d8647ba197a6db4aa58522aa9db588ff08897cd15ade8 -conf /etc/coredns/Corefile
ExecStop=/usr/bin/podman rm -f -i aro-coredns
ExecStopPost=-/usr/bin/podman rm -f -i aro-coredns
Delegate=yes
Type=notify
NotifyAccess=all
SyslogIdentifier=%N
Restart=always

[Install]
WantedBy=multi-user.target
{{ end }}

`))

func Ignition2Config(clusterDomain, apiIntIP, ingressIP string, gatewayDomains []string, gatewayPrivateEndpointIP string) (*ign2types.Config, error) {
	hosts, err := renderTemplateBytes("hosts", clusterDomain, apiIntIP, ingressIP, gatewayDomains, gatewayPrivateEndpointIP)
	if err != nil {
		return nil, err
	}
	corefile, err := renderTemplateBytes("Corefile", clusterDomain, apiIntIP, ingressIP, gatewayDomains, gatewayPrivateEndpointIP)
	if err != nil {
		return nil, err
	}
	dbApps, err := renderTemplateBytes("db.apps", clusterDomain, apiIntIP, ingressIP, gatewayDomains, gatewayPrivateEndpointIP)
	if err != nil {
		return nil, err
	}
	aroCorednsService, err := renderTemplateString("aro-coredns.service", clusterDomain, apiIntIP, ingressIP, gatewayDomains, gatewayPrivateEndpointIP)
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
						Path:       "/etc/hosts.aro",
						User: &ign2types.NodeUser{
							Name: "root",
						},
					},
					FileEmbedded1: ign2types.FileEmbedded1{
						Contents: ign2types.FileContents{
							Source: dataurl.EncodeBytes(hosts),
						},
						Mode: ignutil.IntToPtr(0644),
					},
				},
				{
					Node: ign2types.Node{
						Filesystem: "root",
						Overwrite:  ignutil.BoolToPtr(true),
						Path:       "/etc/coredns/Corefile",
						User: &ign2types.NodeUser{
							Name: "root",
						},
					},
					FileEmbedded1: ign2types.FileEmbedded1{
						Contents: ign2types.FileContents{
							Source: dataurl.EncodeBytes(corefile),
						},
						Mode: ignutil.IntToPtr(0644),
					},
				},
				{
					Node: ign2types.Node{
						Filesystem: "root",
						Overwrite:  ignutil.BoolToPtr(true),
						Path:       "/etc/coredns/zones/db.apps",
						User: &ign2types.NodeUser{
							Name: "root",
						},
					},
					FileEmbedded1: ign2types.FileEmbedded1{
						Contents: ign2types.FileContents{
							Source: dataurl.EncodeBytes(dbApps),
						},
						Mode: ignutil.IntToPtr(0644),
					},
				},
			},
		},
		Systemd: ign2types.Systemd{
			Units: []ign2types.Unit{
				{
					Contents: aroCorednsService,
					Enabled:  ignutil.BoolToPtr(true),
					Name:     "aro-coredns.service",
				},
			},
		},
	}, nil
}

func Ignition3Config(clusterDomain, apiIntIP, ingressIP string, gatewayDomains []string, gatewayPrivateEndpointIP string) (*ign3types.Config, error) {
	hosts, err := renderTemplateBytes("hosts", clusterDomain, apiIntIP, ingressIP, gatewayDomains, gatewayPrivateEndpointIP)
	if err != nil {
		return nil, err
	}
	corefile, err := renderTemplateBytes("Corefile", clusterDomain, apiIntIP, ingressIP, gatewayDomains, gatewayPrivateEndpointIP)
	if err != nil {
		return nil, err
	}
	dbApps, err := renderTemplateBytes("db.apps", clusterDomain, apiIntIP, ingressIP, gatewayDomains, gatewayPrivateEndpointIP)
	if err != nil {
		return nil, err
	}
	aroCorednsService, err := renderTemplateString("aro-coredns.service", clusterDomain, apiIntIP, ingressIP, gatewayDomains, gatewayPrivateEndpointIP)
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
						Path:      "/etc/hosts.aro",
						User: ign3types.NodeUser{
							Name: ignutil.StrToPtr("root"),
						},
					},
					FileEmbedded1: ign3types.FileEmbedded1{
						Contents: ign3types.Resource{
							Source: ignutil.StrToPtr(dataurl.EncodeBytes(hosts)),
						},
						Mode: ignutil.IntToPtr(0644),
					},
				},
				{
					Node: ign3types.Node{
						Overwrite: ignutil.BoolToPtr(true),
						Path:      "/etc/coredns/Corefile",
						User: ign3types.NodeUser{
							Name: ignutil.StrToPtr("root"),
						},
					},
					FileEmbedded1: ign3types.FileEmbedded1{
						Contents: ign3types.Resource{
							Source: ignutil.StrToPtr(dataurl.EncodeBytes(corefile)),
						},
						Mode: ignutil.IntToPtr(0644),
					},
				},
				{
					Node: ign3types.Node{
						Overwrite: ignutil.BoolToPtr(true),
						Path:      "/etc/coredns/zones/db.apps",
						User: ign3types.NodeUser{
							Name: ignutil.StrToPtr("root"),
						},
					},
					FileEmbedded1: ign3types.FileEmbedded1{
						Contents: ign3types.Resource{
							Source: ignutil.StrToPtr(dataurl.EncodeBytes(dbApps)),
						},
						Mode: ignutil.IntToPtr(0644),
					},
				},
			},
		},
		Systemd: ign3types.Systemd{
			Units: []ign3types.Unit{
				{
					Contents: &aroCorednsService,
					Enabled:  ignutil.BoolToPtr(true),
					Name:     "aro-coredns.service",
				},
			},
		},
	}, nil
}

func renderTemplateBytes(file, clusterDomain, apiIntIP, ingressIP string, gatewayDomains []string, gatewayPrivateEndpointIP string) ([]byte, error) {
	buf := &bytes.Buffer{}

	err := t.ExecuteTemplate(buf, file, &struct {
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

func renderTemplateString(file, clusterDomain, apiIntIP, ingressIP string, gatewayDomains []string, gatewayPrivateEndpointIP string) (string, error) {
	buf := &bytes.Buffer{}

	err := t.ExecuteTemplate(buf, file, &struct {
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
		return "", err
	}

	return buf.String(), nil
}
