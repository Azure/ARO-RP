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
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"

	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrlfake "sigs.k8s.io/controller-runtime/pkg/client/fake"

	mcv1 "github.com/openshift/machine-config-operator/pkg/apis/machineconfiguration.openshift.io/v1"

	apiversion "github.com/Azure/ARO-RP/pkg/api/util/version"
	"github.com/Azure/ARO-RP/pkg/operator"
	"github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	"github.com/Azure/ARO-RP/test/util/clienthelper"
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

	testCases := []struct {
		desc                  string
		expectedIsRequired    bool
		clusterFlags          map[string]string
		addHooks              func(*clienthelper.HookingClient)
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
			desc: "enabled, network configuration not present",
			clusterFlags: map[string]string{
				operator.DirtyfragWorkaroundEnabled: operator.FlagTrue,
			},
			expectedIsRequired:    true,
			expectedMachineConfig: expectedMasterDirtyfragMachineConfig(),
		},
		{
			desc: "enabled, non-ipsec cluster",
			clusterFlags: map[string]string{
				operator.DirtyfragWorkaroundEnabled: operator.FlagTrue,
			},
			objects: []client.Object{
				// The necessary parameters for this ipsec config were introduced in
				// OpenShift 4.15, and our vendored APIs are pinned to 4.12.
				// We need to handle this object as unstructured as a result.
				&unstructured.Unstructured{
					Object: map[string]interface{}{
						"apiVersion": "operator.openshift.io/v1",
						"kind":       "Network",
						"metadata": map[string]interface{}{
							"name": "cluster",
						},
						"spec": map[string]interface{}{
							"defaultNetwork": map[string]interface{}{
								"ovnKubernetesConfig": map[string]interface{}{
									"ipsecConfig": map[string]interface{}{
										"mode": "Disabled",
									},
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
			desc: "enabled, ipsec mode not a string",
			clusterFlags: map[string]string{
				operator.DirtyfragWorkaroundEnabled: operator.FlagTrue,
			},
			objects: []client.Object{
				&unstructured.Unstructured{
					Object: map[string]interface{}{
						"apiVersion": "operator.openshift.io/v1",
						"kind":       "Network",
						"metadata": map[string]interface{}{
							"name": "cluster",
						},
						"spec": map[string]interface{}{
							"defaultNetwork": map[string]interface{}{
								"ovnKubernetesConfig": map[string]interface{}{
									"ipsecConfig": map[string]interface{}{
										"mode": true,
									},
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
			desc: "enabled, ipsec cluster",
			clusterFlags: map[string]string{
				operator.DirtyfragWorkaroundEnabled: operator.FlagTrue,
			},
			objects: []client.Object{
				// The necessary parameters for this ipsec config were introduced in
				// OpenShift 4.15, and our vendored APIs are pinned to 4.12.
				// We need to handle this object as unstructured as a result.
				&unstructured.Unstructured{
					Object: map[string]interface{}{
						"apiVersion": "operator.openshift.io/v1",
						"kind":       "Network",
						"metadata": map[string]interface{}{
							"name": "cluster",
						},
						"spec": map[string]interface{}{
							"defaultNetwork": map[string]interface{}{
								"ovnKubernetesConfig": map[string]interface{}{
									"ipsecConfig": map[string]interface{}{
										"mode": "Full",
									},
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
			desc: "enabled, apply errors",
			clusterFlags: map[string]string{
				operator.DirtyfragWorkaroundEnabled: operator.FlagTrue,
			},
			expectedIsRequired: true,
			expectedErr:        errFail,
			addHooks: func(hc *clienthelper.HookingClient) {
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

			// Create a scheme that allows unstructured objects to pass through
			// without validation against the vendored 4.12 openshift/api types
			scheme := runtime.NewScheme()
			// Register only MachineConfig types to allow the test to fetch them
			err := mcv1.Install(scheme)
			r.NoError(err)

			clientBuilder := ctrlfake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(tC.objects...)

			cl := clienthelper.NewHookingClient(clientBuilder.Build())
			if tC.addHooks != nil {
				tC.addHooks(cl)
			}

			workaround := NewDirtyfragWorkaround(log, cl)

			clusterVersion, err := apiversion.ParseVersion("4.99.0")
			r.NoError(err)

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

	marshalDirtyfragIgnition = func(v interface{}) ([]byte, error) {
		return nil, errors.New("marshal failed")
	}
	t.Cleanup(func() {
		marshalDirtyfragIgnition = json.Marshal
	})

	workaround := NewDirtyfragWorkaround(log, ctrlfake.NewClientBuilder().Build())

	err := workaround.Ensure(t.Context())
	r.EqualError(err, "failed to marshal dirtyfrag ignition config: marshal failed")
}
