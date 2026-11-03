package workaround

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/go-test/deep"
	"github.com/stretchr/testify/require"

	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"

	mcv1 "github.com/openshift/api/machineconfiguration/v1"

	apiversion "github.com/Azure/ARO-RP/pkg/api/util/version"
	"github.com/Azure/ARO-RP/pkg/operator"
	"github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	"github.com/Azure/ARO-RP/pkg/util/clienthelper"
	testclienthelper "github.com/Azure/ARO-RP/test/util/clienthelper"
	testlog "github.com/Azure/ARO-RP/test/util/log"
)

// MachineConfig provided for the CVE-2026-98432 mitigation.
const cve202698432ProvidedMachineConfig = `apiVersion: machineconfiguration.openshift.io/v1
kind: MachineConfig
metadata:
  name: 99-master-cve-2026-98432
  labels:
    machineconfiguration.openshift.io/role: master
spec:
  config:
    ignition:
      version: 3.2.0
    storage:
      files:
      - path: /usr/local/bin/cve-2026-98432-mitigation.sh
        mode: 493
        overwrite: true
        contents:
          source: data:text/plain;charset=utf-8;base64,IyEvYmluL2Jhc2gKIyBJbnRlcmltIGhvc3QgbmZ0YWJsZXMgbWl0aWdhdGlvbiBmb3IgQ1ZFLTIwMjYtOTg0MzIuCiMgRHJvcHMgSVB2NCBUQ1AgU1lOIHBhY2tldHMgdGhhdCBjYXJyeSBJUCBvcHRpb25zLCBvbiBwcmVyb3V0aW5nLAojIGZvcndhcmQsIGlucHV0LCBhbmQgb3V0cHV0LiBSZW1vdmUgdGhpcyB1bml0IGFmdGVyIHRoZSBub2RlIGlzCiMgcnVubmluZyBhIGtlcm5lbCB0aGF0IGNvbnRhaW5zIHRoZSBmaXguCnNldCAtZXVvIHBpcGVmYWlsCgpORlQ9L3Vzci9zYmluL25mdApUQUJMRT0naW5ldCBjdmVfMjAyNl85ODQzMicKTUFUQ0g9J2lwIHByb3RvY29sIHRjcCBpcCBoZHJsZW5ndGggPiA1IHRjcCBmbGFncyAmIChmaW4gfCBzeW4gfCByc3QgfCBwc2ggfCBhY2sgfCB1cmcpID09IHN5biBjb3VudGVyIGRyb3AnCgplbnN1cmVfY2hhaW4oKSB7CiAgbG9jYWwgY2hhaW49IiQxIgogIGxvY2FsIGhvb2s9IiQyIgogIGxvY2FsIHByaW9yaXR5PSIkMyIKICBpZiAhICIkTkZUIiBsaXN0IGNoYWluICRUQUJMRSAiJGNoYWluIiA+L2Rldi9udWxsIDI+JjE7IHRoZW4KICAgICIkTkZUIiBhZGQgY2hhaW4gJFRBQkxFICIkY2hhaW4iICJ7IHR5cGUgZmlsdGVyIGhvb2sgJGhvb2sgcHJpb3JpdHkgJHByaW9yaXR5OyBwb2xpY3kgYWNjZXB0OyB9IgogICAgZWNobyAiJChkYXRlIC1JcykgY3JlYXRlZCAkY2hhaW4iCiAgZmkKICBpZiAhICIkTkZUIiBsaXN0IGNoYWluICRUQUJMRSAiJGNoYWluIiB8IGdyZXAgLUZxICdpcCBoZHJsZW5ndGggPiA1JzsgdGhlbgogICAgIiRORlQiIGFkZCBydWxlICRUQUJMRSAiJGNoYWluIiAiJE1BVENIIgogICAgZWNobyAiJChkYXRlIC1JcykgaW5zdGFsbGVkIG1pdGlnYXRpb24gcnVsZSBpbiAkY2hhaW4iCiAgZmkKfQoKd2hpbGUgdHJ1ZTsgZG8KICBpZiAhICIkTkZUIiBsaXN0IHRhYmxlICRUQUJMRSA+L2Rldi9udWxsIDI+JjE7IHRoZW4KICAgICIkTkZUIiBhZGQgdGFibGUgJFRBQkxFCiAgICBlY2hvICIkKGRhdGUgLUlzKSBjcmVhdGVkICRUQUJMRSIKICBmaQogIGVuc3VyZV9jaGFpbiBwcmVyb3V0aW5nIHByZXJvdXRpbmcgLTMxMAogIGVuc3VyZV9jaGFpbiBmb3J3YXJkIGZvcndhcmQgLTMxMAogIGVuc3VyZV9jaGFpbiBpbnB1dCBpbnB1dCAtMzEwCiAgZW5zdXJlX2NoYWluIG91dHB1dCBvdXRwdXQgLTMxMAogIHNsZWVwIDYwCmRvbmUK
    systemd:
      units:
      - name: cve-2026-98432-mitigation.service
        enabled: true
        contents: |
          [Unit]
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

func expectedCVE202698432MachineConfig() *mcv1.MachineConfig {
	mc, err := makeCVE202698432MachineConfig(json.Marshal)
	if err != nil {
		panic(err)
	}
	mc.ResourceVersion = "1"
	return mc
}

func TestCVE202698432MachineConfigMatchesProvided(t *testing.T) {
	r := require.New(t)

	want := &mcv1.MachineConfig{}
	r.NoError(yaml.Unmarshal([]byte(cve202698432ProvidedMachineConfig), want))

	got, err := makeCVE202698432MachineConfig(json.Marshal)
	r.NoError(err)

	r.Equal(want.Name, got.Name)
	r.Equal(want.Labels, got.Labels)
	r.JSONEq(string(want.Spec.Config.Raw), string(got.Spec.Config.Raw))
}

func TestCVE202698432Workaround(t *testing.T) {
	errFail := errors.New("failed client")
	enabled := map[string]string{
		operator.CVE202698432WorkaroundEnabled: operator.FlagTrue,
	}

	testCases := []struct {
		desc                  string
		clusterFlags          map[string]string
		clusterVersion        apiversion.Version
		objects               []client.Object
		addHooks              func(*testclienthelper.HookingClient)
		expectedIsRequired    bool
		expectedMachineConfig *mcv1.MachineConfig
		expectedErr           error
	}{
		{
			desc:         "disabled implicitly, nothing done",
			clusterFlags: map[string]string{},
		},
		{
			desc: "disabled explicitly, existing machineconfig removed",
			clusterFlags: map[string]string{
				operator.CVE202698432WorkaroundEnabled: operator.FlagFalse,
			},
			objects: []client.Object{
				&mcv1.MachineConfig{ObjectMeta: metav1.ObjectMeta{Name: cve202698432MachineConfigName}},
			},
		},
		{
			desc:                  "enabled, applies on 4.21",
			clusterFlags:          enabled,
			expectedIsRequired:    true,
			expectedMachineConfig: expectedCVE202698432MachineConfig(),
		},
		{
			desc:                  "enabled, applies on 4.22 and later",
			clusterFlags:          enabled,
			clusterVersion:        apiversion.NewVersion(4, 22, 0),
			expectedIsRequired:    true,
			expectedMachineConfig: expectedCVE202698432MachineConfig(),
		},
		{
			desc:                  "enabled, applies on older versions",
			clusterFlags:          enabled,
			clusterVersion:        apiversion.NewVersion(4, 12, 0),
			expectedIsRequired:    true,
			expectedMachineConfig: expectedCVE202698432MachineConfig(),
		},
		{
			desc:               "enabled, apply errors",
			clusterFlags:       enabled,
			expectedIsRequired: true,
			expectedErr:        errFail,
			addHooks: func(hc *testclienthelper.HookingClient) {
				hc.WithPreCreateHook(func(obj client.Object) error {
					return errFail
				})
			},
		},
	}
	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {
			r := require.New(t)
			_, log := testlog.LogForTesting(t)

			cl := testclienthelper.NewHookingClient(testclienthelper.NewAROFakeClientBuilder(tC.objects...).Build())
			if tC.addHooks != nil {
				tC.addHooks(cl)
			}

			workaround := NewCVE202698432Workaround(log, cl)

			clusterVersion := tC.clusterVersion
			if clusterVersion == nil {
				clusterVersion = apiversion.NewVersion(4, 21, 0)
			}

			isRequired, err := workaround.IsRequired(t.Context(), clusterVersion, &v1alpha1.Cluster{Spec: v1alpha1.ClusterSpec{
				OperatorFlags: v1alpha1.OperatorFlags(tC.clusterFlags),
			}})
			r.NoError(err)
			r.Equal(tC.expectedIsRequired, isRequired)

			if isRequired {
				err = workaround.Ensure(t.Context())
			} else {
				err = workaround.Remove(t.Context())
			}
			if tC.expectedErr != nil {
				r.ErrorIs(err, tC.expectedErr)
			} else {
				r.NoError(err)
			}

			got := &mcv1.MachineConfig{}
			err = cl.Get(t.Context(), types.NamespacedName{Name: cve202698432MachineConfigName}, got)
			if tC.expectedMachineConfig == nil {
				r.True(kerrors.IsNotFound(err), "expected machineconfig to be absent, got err %v", err)
			} else {
				r.NoError(err)
				r.Equal(tC.expectedMachineConfig, got, "failed: %v", deep.Equal(got, tC.expectedMachineConfig))
			}
		})
	}
}

func TestCVE202698432WorkaroundEnsureMarshalError(t *testing.T) {
	r := require.New(t)
	_, log := testlog.LogForTesting(t)
	expectedErr := errors.New("marshal failed")
	ensureCalled := false

	cl := testclienthelper.NewHookingClient(testclienthelper.NewAROFakeClientBuilder().Build())
	cl.WithPreCreateHook(func(obj client.Object) error {
		ensureCalled = true
		return nil
	})

	workaround := &cve202698432workaround{
		log: log,
		ch:  clienthelper.NewWithClient(log, cl),
		marshal: func(v any) ([]byte, error) {
			return nil, expectedErr
		},
	}

	err := workaround.Ensure(t.Context())
	r.ErrorIs(err, expectedErr)
	r.EqualError(err, "failed to marshal CVE-2026-98432 ignition config: marshal failed")
	r.False(ensureCalled)
}
