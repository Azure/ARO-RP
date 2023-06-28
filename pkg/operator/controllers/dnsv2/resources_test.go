package dnsv2

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"encoding/json"
	"testing"

	ign2types "github.com/coreos/ignition/config/v2_2/types"
	ign3types "github.com/coreos/ignition/v2/config/v3_2/types"
	"github.com/google/go-cmp/cmp"
	"github.com/vincent-petithory/dataurl"
)

var (
	hostsfile = dataurl.EncodeBytes([]byte(`127.0.0.1   localhost localhost.localdomain localhost4 localhost4.localdomain4
::1         localhost localhost.localdomain localhost6 localhost6.localdomain6
# XXX image-registry.openshift-image-registry.svc image-registry.openshift-image-registry.svc.cluster.local # openshift-generated-node-resolver
10.194.0.1 api.cluster.example.com
10.194.0.1 api-int.cluster.example.com
# 10.194.0.2 *.apps.cluster.example.com
10.195.0.1 agentimagestorewus01.blob.core.windows.net
10.195.0.1 agentimagestorecus01.blob.core.windows.net
10.195.0.1 agentimagestoreeus01.blob.core.windows.net
10.195.0.1 agentimagestoreweu01.blob.core.windows.net
10.195.0.1 agentimagestoreeas01.blob.core.windows.net
10.195.0.1 eastus-shared.prod.warm.ingest.monitor.core.windows.net
10.195.0.1 gcs.prod.monitoring.core.windows.net
10.195.0.1 gsm1318942586eh.servicebus.windows.net
10.195.0.1 gsm1318942586xt.blob.core.windows.net
10.195.0.1 gsm1580628551eh.servicebus.windows.net
10.195.0.1 gsm1580628551xt.blob.core.windows.net
10.195.0.1 gsm479052001eh.servicebus.windows.net
10.195.0.1 gsm479052001xt.blob.core.windows.net
10.195.0.1 maupdateaccount.blob.core.windows.net
10.195.0.1 maupdateaccount2.blob.core.windows.net
10.195.0.1 maupdateaccount3.blob.core.windows.net
10.195.0.1 maupdateaccount4.blob.core.windows.net
10.195.0.1 production.diagnostics.monitoring.core.windows.net
10.195.0.1 qos.prod.warm.ingest.monitor.core.windows.net
10.195.0.1 login.microsoftonline.com
10.195.0.1 management.azure.com
10.195.0.1 arosvc.azurecr.io
10.195.0.1 arosvc.eastus.data.azurecr.io
10.195.0.1 imageregistry6rmpk.blob.core.windows.net
`))

	corefile = dataurl.EncodeBytes([]byte(`apps.cluster.example.com:53 {
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
`))
	zonefile = dataurl.EncodeBytes([]byte(`$ORIGIN apps.cluster.example.com.
@	3600 IN	SOA ns1-09.azure-dns.com. azuredns-hostmaster.microsoft.com. 1 3600 300 2419200 300

*        IN A     10.194.0.2
`))

	corednsServiceBytes, _ = json.Marshal(`[Unit]
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
`)
	corednsService = string(corednsServiceBytes)

	gatewayDomains = []string{
		"agentimagestorewus01.blob.core.windows.net",
		"agentimagestorecus01.blob.core.windows.net",
		"agentimagestoreeus01.blob.core.windows.net",
		"agentimagestoreweu01.blob.core.windows.net",
		"agentimagestoreeas01.blob.core.windows.net",
		"eastus-shared.prod.warm.ingest.monitor.core.windows.net",
		"gcs.prod.monitoring.core.windows.net",
		"gsm1318942586eh.servicebus.windows.net",
		"gsm1318942586xt.blob.core.windows.net",
		"gsm1580628551eh.servicebus.windows.net",
		"gsm1580628551xt.blob.core.windows.net",
		"gsm479052001eh.servicebus.windows.net",
		"gsm479052001xt.blob.core.windows.net",
		"maupdateaccount.blob.core.windows.net",
		"maupdateaccount2.blob.core.windows.net",
		"maupdateaccount3.blob.core.windows.net",
		"maupdateaccount4.blob.core.windows.net",
		"production.diagnostics.monitoring.core.windows.net",
		"qos.prod.warm.ingest.monitor.core.windows.net",
		"login.microsoftonline.com",
		"management.azure.com",
		"arosvc.azurecr.io",
		"arosvc.eastus.data.azurecr.io",
		"imageregistry6rmpk.blob.core.windows.net",
	}
)

func TestIgnition2Config(t *testing.T) {
	desiredIgnition2Config := `{
	"ignition": {
		"config": {},
		"security": {
			"tls": {}
		},
		"timeouts": {},
		"version": "` + ign2types.MaxVersion.String() + `"
	},
	"networkd": {},
	"passwd": {},
	"storage": {
		"files": [
			{
				"filesystem": "root",
				"overwrite": true,
				"path": "/etc/hosts.aro",
				"user": {
					"name": "root"
				},
				"contents": {
					"source": "` + hostsfile + `",
					"verification": {}
				},
				"mode": 420
			},
			{
				"filesystem": "root",
				"overwrite": true,
				"path": "/etc/coredns/Corefile",
				"user": {
					"name": "root"
				},
				"contents": {
					"source": "` + corefile + `",
					"verification": {}
				},
				"mode": 420
			},
			{
				"filesystem": "root",
				"overwrite": true,
				"path": "/etc/coredns/zones/db.apps",
				"user": {
					"name": "root"
				},
				"contents": {
					"source": "` + zonefile + `",
					"verification": {}
				},
				"mode": 420
			}
		]
	},
	"systemd": {
		"units": [
			{
				"contents": ` + corednsService + `,
				"enabled": true,
				"name": "aro-coredns.service"
			}
		]
	}
}`

	ignition, err := Ignition2Config(
		"cluster.example.com",
		"10.194.0.1",
		"10.194.0.2",
		gatewayDomains,
		"10.195.0.1",
	)
	if err != nil {
		t.Error(err)
	}

	j, err := json.MarshalIndent(ignition, "", "\t")
	if err != nil {
		t.Error(err)
	}
	s := string(j)

	if desiredIgnition2Config != s {
		t.Error(cmp.Diff(desiredIgnition2Config, s))
	}
}

func TestIgnition3Config(t *testing.T) {
	desiredIgnition3Config := `{
	"ignition": {
		"config": {
			"replace": {
				"verification": {}
			}
		},
		"proxy": {},
		"security": {
			"tls": {}
		},
		"timeouts": {},
		"version": "` + ign3types.MaxVersion.String() + `"
	},
	"passwd": {},
	"storage": {
		"files": [
			{
				"group": {},
				"overwrite": true,
				"path": "/etc/hosts.aro",
				"user": {
					"name": "root"
				},
				"contents": {
					"source": "` + hostsfile + `",
					"verification": {}
				},
				"mode": 420
			},
			{
				"group": {},
				"overwrite": true,
				"path": "/etc/coredns/Corefile",
				"user": {
					"name": "root"
				},
				"contents": {
					"source": "` + corefile + `",
					"verification": {}
				},
				"mode": 420
			},
			{
				"group": {},
				"overwrite": true,
				"path": "/etc/coredns/zones/db.apps",
				"user": {
					"name": "root"
				},
				"contents": {
					"source": "` + zonefile + `",
					"verification": {}
				},
				"mode": 420
			}
		]
	},
	"systemd": {
		"units": [
			{
				"contents": ` + corednsService + `,
				"enabled": true,
				"name": "aro-coredns.service"
			}
		]
	}
}`

	ignition, err := Ignition3Config(
		"cluster.example.com",
		"10.194.0.1",
		"10.194.0.2",
		gatewayDomains,
		"10.195.0.1",
	)
	if err != nil {
		t.Error(err)
	}

	j, err := json.MarshalIndent(ignition, "", "\t")
	if err != nil {
		t.Error(err)
	}
	s := string(j)

	if desiredIgnition3Config != s {
		t.Error(cmp.Diff(desiredIgnition3Config, s))
	}
}
