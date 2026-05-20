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

	mcv1 "github.com/openshift/api/machineconfiguration/v1"
	operatorv1 "github.com/openshift/api/operator/v1"

	apiversion "github.com/Azure/ARO-RP/pkg/api/util/version"
	"github.com/Azure/ARO-RP/pkg/operator"
	"github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	testclienthelper "github.com/Azure/ARO-RP/test/util/clienthelper"
	testlog "github.com/Azure/ARO-RP/test/util/log"
)

func expectedMasterDirtyfragMachineConfig() *mcv1.MachineConfig {
	mc, err := makeDirtyfragMachineConfig("master")
	if err != nil {
		panic(err)
	}
	mc.ResourceVersion = "1"
	return mc
}

func TestDirtyfragWorkaround(t *testing.T) {
	errFail := errors.New("failed client")
	dirtyfragEnabled := map[string]string{
		operator.DirtyfragWorkaroundEnabled: operator.FlagTrue,
	}

	testCases := []struct {
		desc                  string
		expectedIsRequired    bool
		clusterFlags          map[string]string
		clusterVersion        apiversion.Version
		addHooks              func(*testclienthelper.HookingClient)
		objects               []client.Object
		expectedMachineConfig *mcv1.MachineConfig
		expectedErr           error
		expectedErrIsRequired error
	}{
		{
			desc:         "disabled implicitly, nothing done",
			clusterFlags: map[string]string{},
		},
		{
			desc: "disabled explicitly, nothing done",
			clusterFlags: map[string]string{
				operator.DirtyfragWorkaroundEnabled: operator.FlagFalse,
			},
		},
		{
			desc:                  "enabled, network configuration not present",
			clusterFlags:          dirtyfragEnabled,
			clusterVersion:        apiversion.NewVersion(4, 21, 0),
			expectedIsRequired:    true,
			expectedMachineConfig: expectedMasterDirtyfragMachineConfig(),
		},
		{
			desc:           "enabled, non-ipsec cluster",
			clusterFlags:   dirtyfragEnabled,
			clusterVersion: apiversion.NewVersion(4, 21, 0),
			objects: []client.Object{
				&operatorv1.Network{
					ObjectMeta: metav1.ObjectMeta{
						Name: "cluster",
					},
					Spec: operatorv1.NetworkSpec{
						DefaultNetwork: operatorv1.DefaultNetworkDefinition{
							OVNKubernetesConfig: &operatorv1.OVNKubernetesConfig{
								IPsecConfig: &operatorv1.IPsecConfig{
									Mode: operatorv1.IPsecModeDisabled,
								},
							},
						},
					},
				},
			},
			expectedIsRequired:    true,
			expectedMachineConfig: expectedMasterDirtyfragMachineConfig(),
		},
		{
			desc:           "enabled, ipsec mode not a string",
			clusterFlags:   dirtyfragEnabled,
			clusterVersion: apiversion.NewVersion(4, 21, 0),
			objects: []client.Object{
				&operatorv1.Network{
					ObjectMeta: metav1.ObjectMeta{
						Name: "cluster",
					},
					Spec: operatorv1.NetworkSpec{},
				},
			},
			expectedIsRequired:    true,
			expectedMachineConfig: expectedMasterDirtyfragMachineConfig(),
		},
		{
			desc: "enabled, ipsec mode missing",
			clusterFlags: map[string]string{
				operator.DirtyfragWorkaroundEnabled: operator.FlagTrue,
			},
			objects: []client.Object{
				&operatorv1.Network{
					ObjectMeta: metav1.ObjectMeta{
						Name: "cluster",
					},
					Spec: operatorv1.NetworkSpec{
						DefaultNetwork: operatorv1.DefaultNetworkDefinition{
							OVNKubernetesConfig: &operatorv1.OVNKubernetesConfig{},
						},
					},
				},
			},
			expectedIsRequired:    true,
			expectedMachineConfig: expectedMasterDirtyfragMachineConfig(),
		},
		{
			desc:           "enabled, ipsec cluster",
			clusterFlags:   dirtyfragEnabled,
			clusterVersion: apiversion.NewVersion(4, 21, 0),
			objects: []client.Object{
				&operatorv1.Network{
					ObjectMeta: metav1.ObjectMeta{
						Name: "cluster",
					},
					Spec: operatorv1.NetworkSpec{
						DefaultNetwork: operatorv1.DefaultNetworkDefinition{
							OVNKubernetesConfig: &operatorv1.OVNKubernetesConfig{
								IPsecConfig: &operatorv1.IPsecConfig{
									Mode: operatorv1.IPsecModeFull,
								},
							},
						},
					},
				},
			},
			expectedIsRequired:    false,
			expectedMachineConfig: nil,
		},
		{
			desc:               "enabled, apply errors",
			clusterFlags:       dirtyfragEnabled,
			clusterVersion:     apiversion.NewVersion(4, 21, 0),
			expectedIsRequired: true,
			expectedErr:        errFail,
			addHooks: func(hc *testclienthelper.HookingClient) {
				hc.WithPreCreateHook(func(obj client.Object) error {
					return errFail
				})
			},
		},
		{
			desc:           "enabled, does not apply on clusterversion 4.22.0 or greater",
			clusterVersion: apiversion.NewVersion(4, 22, 0),
			clusterFlags:   dirtyfragEnabled,
		},
		{
			desc:           "enabled, does not apply on clusterversion 4.21.15",
			clusterVersion: apiversion.NewVersion(4, 21, 15),
			clusterFlags:   dirtyfragEnabled,
		},
		{
			desc:                  "enabled, does apply on clusterversion 4.21.14 or earlier 4.21.z",
			clusterVersion:        apiversion.NewVersion(4, 21, 14),
			clusterFlags:          dirtyfragEnabled,
			expectedIsRequired:    true,
			expectedMachineConfig: expectedMasterDirtyfragMachineConfig(),
		},
		{
			desc:           "enabled, does not apply on clusterversion 4.20.22",
			clusterVersion: apiversion.NewVersion(4, 20, 22),
			clusterFlags:   dirtyfragEnabled,
		},
		{
			desc:                  "enabled, does apply on clusterversion 4.20.21 or earlier 4.20.z",
			clusterVersion:        apiversion.NewVersion(4, 20, 21),
			clusterFlags:          dirtyfragEnabled,
			expectedIsRequired:    true,
			expectedMachineConfig: expectedMasterDirtyfragMachineConfig(),
		},
		{
			desc:           "enabled, does not apply on clusterversion 4.19.31",
			clusterVersion: apiversion.NewVersion(4, 19, 31),
			clusterFlags:   dirtyfragEnabled,
		},
		{
			desc:                  "enabled, does apply on clusterversion 4.19.30 or earlier 4.19.z",
			clusterVersion:        apiversion.NewVersion(4, 19, 30),
			clusterFlags:          dirtyfragEnabled,
			expectedIsRequired:    true,
			expectedMachineConfig: expectedMasterDirtyfragMachineConfig(),
		},
		{
			desc:           "enabled, does not apply on clusterversion 4.18.41",
			clusterVersion: apiversion.NewVersion(4, 18, 41),
			clusterFlags:   dirtyfragEnabled,
		},
		{
			desc:                  "enabled, does apply on clusterversion 4.18.40 or earlier 4.18.z",
			clusterVersion:        apiversion.NewVersion(4, 18, 40),
			clusterFlags:          dirtyfragEnabled,
			expectedIsRequired:    true,
			expectedMachineConfig: expectedMasterDirtyfragMachineConfig(),
		},
		{
			desc:           "enabled, does not apply on clusterversion 4.16.62",
			clusterVersion: apiversion.NewVersion(4, 16, 62),
			clusterFlags:   dirtyfragEnabled,
		},
		{
			desc:                  "enabled, does apply on clusterversion 4.16.61 or earlier 4.16.z",
			clusterVersion:        apiversion.NewVersion(4, 16, 61),
			clusterFlags:          dirtyfragEnabled,
			expectedIsRequired:    true,
			expectedMachineConfig: expectedMasterDirtyfragMachineConfig(),
		},
		{
			desc:           "enabled, does not apply on clusterversion 4.14.66",
			clusterVersion: apiversion.NewVersion(4, 14, 66),
			clusterFlags:   dirtyfragEnabled,
		},
		{
			desc:                  "enabled, does apply on clusterversion 4.14.65 or earlier 4.14.z",
			clusterVersion:        apiversion.NewVersion(4, 14, 65),
			clusterFlags:          dirtyfragEnabled,
			expectedIsRequired:    true,
			expectedMachineConfig: expectedMasterDirtyfragMachineConfig(),
		},
		{
			desc:           "enabled, does not apply on clusterversion 4.12.90",
			clusterVersion: apiversion.NewVersion(4, 12, 90),
			clusterFlags:   dirtyfragEnabled,
		},
		{
			desc:                  "enabled, does apply on clusterversion 4.12.89 or earlier 4.12.z",
			clusterVersion:        apiversion.NewVersion(4, 12, 89),
			clusterFlags:          dirtyfragEnabled,
			expectedIsRequired:    true,
			expectedMachineConfig: expectedMasterDirtyfragMachineConfig(),
		},
		{
			desc:                  "enabled, does apply on unlisted minor versions (4.11)",
			clusterVersion:        apiversion.NewVersion(4, 11, 99),
			clusterFlags:          dirtyfragEnabled,
			expectedIsRequired:    true,
			expectedMachineConfig: expectedMasterDirtyfragMachineConfig(),
		},
	}
	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {
			r := require.New(t)
			_, log := testlog.LogForTesting(t)

			clientBuilder := testclienthelper.NewAROFakeClientBuilder(tC.objects...)

			cl := testclienthelper.NewHookingClient(clientBuilder.Build())
			if tC.addHooks != nil {
				tC.addHooks(cl)
			}

			workaround := NewDirtyfragWorkaround(log, cl)

			clusterVersion := tC.clusterVersion
			if clusterVersion == nil {
				var err error
				clusterVersion, err = apiversion.ParseVersion("4.21.0")
				r.NoError(err)
			}

			isRequired, err := workaround.IsRequired(t.Context(), clusterVersion, &v1alpha1.Cluster{Spec: v1alpha1.ClusterSpec{
				OperatorFlags: v1alpha1.OperatorFlags(tC.clusterFlags),
			}})
			if tC.expectedErrIsRequired != nil {
				r.ErrorIs(err, tC.expectedErrIsRequired)
			} else {
				r.NoError(err)
			}
			r.Equal(tC.expectedIsRequired, isRequired, "should have not registered as isRequired")

			if tC.expectedErrIsRequired == nil {
				if isRequired {
					err = workaround.Ensure(t.Context())
					if tC.expectedErr != nil {
						r.ErrorIs(err, tC.expectedErr)
					} else {
						r.NoError(err)
					}
				} else {
					err = workaround.Remove(t.Context())
					if tC.expectedErr != nil {
						r.ErrorIs(err, tC.expectedErr)
					} else {
						r.NoError(err)
					}
				}
			}

			got := &mcv1.MachineConfig{}
			err = cl.Get(t.Context(), types.NamespacedName{Name: "99-master-disable-dirtyfrag"}, got)
			if tC.expectedMachineConfig == nil {
				if err == nil {
					t.Error("found machineconfig when it should not exist")
				} else if !kerrors.IsNotFound(err) {
					t.Errorf("error when fetching machineconfig: %v", err.Error())
				}
			} else {
				r.NoError(err)

				r.Equal(tC.expectedMachineConfig, got, "failed: %v", deep.Equal(got, tC.expectedMachineConfig))
			}
		})
	}
}

func TestDirtyfragWorkaroundEnsureMarshalError(t *testing.T) {
	r := require.New(t)
	_, log := testlog.LogForTesting(t)
	expectedErr := errors.New("marshal failed")
	ensureCalled := false

	marshalDirtyfragIgnition = func(v interface{}) ([]byte, error) {
		return nil, expectedErr
	}
	t.Cleanup(func() {
		marshalDirtyfragIgnition = json.Marshal
	})

	cl := testclienthelper.NewHookingClient(testclienthelper.NewAROFakeClientBuilder().Build())
	cl.WithPreCreateHook(func(obj client.Object) error {
		ensureCalled = true
		return nil
	})

	workaround := NewDirtyfragWorkaround(log, cl)

	err := workaround.Ensure(t.Context())
	r.ErrorIs(err, expectedErr)
	r.EqualError(err, "failed to marshal dirtyfrag ignition config: marshal failed")
	r.False(ensureCalled)
}
