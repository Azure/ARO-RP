package workaround

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"errors"
	"testing"

	"github.com/go-test/deep"
	"github.com/stretchr/testify/require"

	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrlfake "sigs.k8s.io/controller-runtime/pkg/client/fake"

	mcv1 "github.com/openshift/machine-config-operator/pkg/apis/machineconfiguration.openshift.io/v1"

	apiversion "github.com/Azure/ARO-RP/pkg/api/util/version"
	"github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	"github.com/Azure/ARO-RP/test/util/clienthelper"
	testlog "github.com/Azure/ARO-RP/test/util/log"
)

func TestCopyFailWorkaround(t *testing.T) {
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
				"aro.workaround.copyfail.enabled": "false",
			},
		},
		{
			desc: "enabled, not a FIPS cluster",
			clusterFlags: map[string]string{
				"aro.workaround.copyfail.enabled": "true",
			},
			expectedIsRequired: true,
			expectedMachineConfig: &mcv1.MachineConfig{
				TypeMeta: metav1.TypeMeta{
					APIVersion: mcv1.SchemeGroupVersion.String(),
					Kind:       "MachineConfig",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: "99-master-disable-algif-aead",
					Labels: map[string]string{
						"machineconfiguration.openshift.io/role": "master",
					},
					ResourceVersion: "1",
				},
				Spec: mcv1.MachineConfigSpec{
					KernelArguments: []string{"initcall_blacklist=algif_aead_init"},
				},
			},
		},
		{
			desc: "enabled, apply errors",
			clusterFlags: map[string]string{
				"aro.workaround.copyfail.enabled": "true",
			},
			expectedIsRequired: true,
			expectedErr:        errFail,
			addHooks: func(hc *clienthelper.HookingClient) {
				hc.WithPreCreateHook(func(obj client.Object) error {
					return errFail
				})
			},
		},
		{
			desc: "enabled, isRequired errors",
			clusterFlags: map[string]string{
				"aro.workaround.copyfail.enabled": "true",
			},
			expectedIsRequired:    false,
			expectedErrIsRequired: errFail,
			addHooks: func(hc *clienthelper.HookingClient) {
				hc.WithPreGetHook(func(key client.ObjectKey, obj client.Object) error {
					if key.Name == "99-master-fips" {
						return errFail
					}
					return nil
				})
			},
		},
		{
			desc: "enabled, is a FIPS cluster",
			clusterFlags: map[string]string{
				"aro.workaround.copyfail.enabled": "true",
			},
			objects: []client.Object{
				&mcv1.MachineConfig{
					ObjectMeta: metav1.ObjectMeta{
						Name: "99-master-fips",
						Labels: map[string]string{
							"machineconfiguration.openshift.io/role": "master",
						},
					},
					Spec: mcv1.MachineConfigSpec{
						FIPS: true,
					},
				},
			},
			expectedIsRequired: false,
		},
	}
	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {
			r := require.New(t)
			_, log := testlog.LogForTesting(t)
			clientBuilder := ctrlfake.NewClientBuilder().WithObjects(tC.objects...)

			cl := clienthelper.NewHookingClient(clientBuilder.Build())
			if tC.addHooks != nil {
				tC.addHooks(cl)
			}

			workaround := NewCopyFailWorkaround(log, cl)

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
			err = cl.Get(t.Context(), types.NamespacedName{Name: "99-master-disable-algif-aead"}, got)
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
