package workaround

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/sirupsen/logrus"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"

	"sigs.k8s.io/controller-runtime/pkg/client"

	mcv1 "github.com/openshift/api/machineconfiguration/v1"

	"github.com/Azure/ARO-RP/pkg/operator"
	"github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	"github.com/Azure/ARO-RP/pkg/util/clienthelper"
	"github.com/Azure/ARO-RP/pkg/util/version"
)

const (
	cve202698432MachineConfigName = "99-master-cve-2026-98432"

	// Script that keeps nftables rules dropping IPv4 TCP SYN packets carrying IP options.
	cve202698432ScriptSource = "data:text/plain;charset=utf-8;base64,IyEvYmluL2Jhc2gKIyBJbnRlcmltIGhvc3QgbmZ0YWJsZXMgbWl0aWdhdGlvbiBmb3IgQ1ZFLTIwMjYtOTg0MzIuCiMgRHJvcHMgSVB2NCBUQ1AgU1lOIHBhY2tldHMgdGhhdCBjYXJyeSBJUCBvcHRpb25zLCBvbiBwcmVyb3V0aW5nLAojIGZvcndhcmQsIGlucHV0LCBhbmQgb3V0cHV0LiBSZW1vdmUgdGhpcyB1bml0IGFmdGVyIHRoZSBub2RlIGlzCiMgcnVubmluZyBhIGtlcm5lbCB0aGF0IGNvbnRhaW5zIHRoZSBmaXguCnNldCAtZXVvIHBpcGVmYWlsCgpORlQ9L3Vzci9zYmluL25mdApUQUJMRT0naW5ldCBjdmVfMjAyNl85ODQzMicKTUFUQ0g9J2lwIHByb3RvY29sIHRjcCBpcCBoZHJsZW5ndGggPiA1IHRjcCBmbGFncyAmIChmaW4gfCBzeW4gfCByc3QgfCBwc2ggfCBhY2sgfCB1cmcpID09IHN5biBjb3VudGVyIGRyb3AnCgplbnN1cmVfY2hhaW4oKSB7CiAgbG9jYWwgY2hhaW49IiQxIgogIGxvY2FsIGhvb2s9IiQyIgogIGxvY2FsIHByaW9yaXR5PSIkMyIKICBpZiAhICIkTkZUIiBsaXN0IGNoYWluICRUQUJMRSAiJGNoYWluIiA+L2Rldi9udWxsIDI+JjE7IHRoZW4KICAgICIkTkZUIiBhZGQgY2hhaW4gJFRBQkxFICIkY2hhaW4iICJ7IHR5cGUgZmlsdGVyIGhvb2sgJGhvb2sgcHJpb3JpdHkgJHByaW9yaXR5OyBwb2xpY3kgYWNjZXB0OyB9IgogICAgZWNobyAiJChkYXRlIC1JcykgY3JlYXRlZCAkY2hhaW4iCiAgZmkKICBpZiAhICIkTkZUIiBsaXN0IGNoYWluICRUQUJMRSAiJGNoYWluIiB8IGdyZXAgLUZxICdpcCBoZHJsZW5ndGggPiA1JzsgdGhlbgogICAgIiRORlQiIGFkZCBydWxlICRUQUJMRSAiJGNoYWluIiAiJE1BVENIIgogICAgZWNobyAiJChkYXRlIC1JcykgaW5zdGFsbGVkIG1pdGlnYXRpb24gcnVsZSBpbiAkY2hhaW4iCiAgZmkKfQoKd2hpbGUgdHJ1ZTsgZG8KICBpZiAhICIkTkZUIiBsaXN0IHRhYmxlICRUQUJMRSA+L2Rldi9udWxsIDI+JjE7IHRoZW4KICAgICIkTkZUIiBhZGQgdGFibGUgJFRBQkxFCiAgICBlY2hvICIkKGRhdGUgLUlzKSBjcmVhdGVkICRUQUJMRSIKICBmaQogIGVuc3VyZV9jaGFpbiBwcmVyb3V0aW5nIHByZXJvdXRpbmcgLTMxMAogIGVuc3VyZV9jaGFpbiBmb3J3YXJkIGZvcndhcmQgLTMxMAogIGVuc3VyZV9jaGFpbiBpbnB1dCBpbnB1dCAtMzEwCiAgZW5zdXJlX2NoYWluIG91dHB1dCBvdXRwdXQgLTMxMAogIHNsZWVwIDYwCmRvbmUK"

	cve202698432Unit = `[Unit]
Description=Interim nftables mitigation for CVE-2026-98432
After=network-online.target
Wants=network-online.target
[Service]
Type=simple
ExecStart=/usr/local/bin/cve-2026-98432-mitigation.sh
ExecStop=-/usr/sbin/nft delete table inet cve_2026_98432
Restart=always
RestartSec=10
[Install]
WantedBy=multi-user.target
`
)

type cve202698432workaround struct {
	log *logrus.Entry
	ch  clienthelper.Interface

	marshal func(v any) ([]byte, error)
}

var _ Workaround = &cve202698432workaround{}

func NewCVE202698432Workaround(log *logrus.Entry, client client.Client) *cve202698432workaround {
	ch := clienthelper.NewWithClient(log, client)
	return &cve202698432workaround{log: log, ch: ch, marshal: json.Marshal}
}

// IsRequired implements [Workaround].
func (a *cve202698432workaround) IsRequired(ctx context.Context, clusterVersion version.Version, cluster *v1alpha1.Cluster) (bool, error) {
	// No OpenShift release contains the fix yet, so this applies to all versions.
	return cluster.Spec.OperatorFlags.GetSimpleBoolean(operator.CVE202698432WorkaroundEnabled), nil
}

// Ensure implements [Workaround].
func (a *cve202698432workaround) Ensure(ctx context.Context) error {
	mc, err := makeCVE202698432MachineConfig(a.marshal)
	if err != nil {
		return err
	}

	return a.ch.Ensure(ctx, mc)
}

// Name implements [Workaround].
func (a *cve202698432workaround) Name() string {
	return "workaround for CVE-2026-98432 on control plane"
}

// Remove implements [Workaround].
func (a *cve202698432workaround) Remove(ctx context.Context) error {
	return a.ch.EnsureDeleted(
		ctx,
		mcv1.GroupVersion.WithKind("MachineConfig"),
		types.NamespacedName{Name: cve202698432MachineConfigName},
	)
}

func makeCVE202698432MachineConfig(marshal func(any) ([]byte, error)) (*mcv1.MachineConfig, error) {
	ignitionConfig := map[string]any{
		"ignition": map[string]any{
			"version": "3.2.0",
		},
		"storage": map[string]any{
			"files": []map[string]any{
				{
					"path":      "/usr/local/bin/cve-2026-98432-mitigation.sh",
					"mode":      493, // 0755 in octal
					"overwrite": true,
					"contents": map[string]any{
						"source": cve202698432ScriptSource,
					},
				},
			},
		},
		"systemd": map[string]any{
			"units": []map[string]any{
				{
					"name":     "cve-2026-98432-mitigation.service",
					"enabled":  true,
					"contents": cve202698432Unit,
				},
			},
		},
	}

	ignitionJSON, err := marshal(ignitionConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal CVE-2026-98432 ignition config: %w", err)
	}

	return &mcv1.MachineConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name: cve202698432MachineConfigName,
			Labels: map[string]string{
				"machineconfiguration.openshift.io/role": "master",
			},
		},
		Spec: mcv1.MachineConfigSpec{
			Config: runtime.RawExtension{
				Raw: ignitionJSON,
			},
		},
	}, nil
}
