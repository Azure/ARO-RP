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

func expectedMasterMachineConfig() *mcv1.MachineConfig {
	mc := makeMachineConfig("master")
	mc.ResourceVersion = "1"
	return mc
}

func TestCopyFailWorkaround(t *testing.T) {
	errFail := errors.New("failed client")
	copyFailEnabled := map[string]string{
		"aro.workaround.copyfail.enabled": "true",
	}

	testCases := []struct {
		desc                  string
		expectedIsRequired    bool
		clusterFlags          map[string]string
		clusterVersion        apiversion.Version
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
			desc:                  "enabled, not a FIPS cluster",
			clusterFlags:          copyFailEnabled,
			expectedIsRequired:    true,
			expectedMachineConfig: expectedMasterMachineConfig(),
		},
		{
			desc:               "enabled, apply errors",
			clusterFlags:       copyFailEnabled,
			expectedIsRequired: true,
			expectedErr:        errFail,
			addHooks: func(hc *clienthelper.HookingClient) {
				hc.WithPreCreateHook(func(obj client.Object) error {
					return errFail
				})
			},
		},
		{
			desc:         "enabled, apply succeeds on FIPS cluster",
			clusterFlags: copyFailEnabled,
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
			expectedIsRequired:    true,
			expectedMachineConfig: expectedMasterMachineConfig(),
		},
		{
			desc:           "enabled, does not apply on clusterversion 4.22.0 or greater",
			clusterVersion: apiversion.NewVersion(4, 22, 0),
			clusterFlags:   copyFailEnabled,
		},
		{
			desc:           "enabled, does not apply on clusterversion 4.21.14",
			clusterVersion: apiversion.NewVersion(4, 21, 14),
			clusterFlags:   copyFailEnabled,
		},
		{
			desc:                  "enabled, does apply on clusterversion 4.21.13 or earlier 4.21.z",
			clusterVersion:        apiversion.NewVersion(4, 21, 13),
			clusterFlags:          copyFailEnabled,
			expectedIsRequired:    true,
			expectedMachineConfig: expectedMasterMachineConfig(),
		},
		{
			desc:           "enabled, does not apply on clusterversion 4.20.21",
			clusterVersion: apiversion.NewVersion(4, 20, 21),
			clusterFlags:   copyFailEnabled,
		},
		{
			desc:                  "enabled, does apply on clusterversion 4.20.20 or earlier 4.20.z",
			clusterVersion:        apiversion.NewVersion(4, 20, 20),
			clusterFlags:          copyFailEnabled,
			expectedIsRequired:    true,
			expectedMachineConfig: expectedMasterMachineConfig(),
		},
		{
			desc:           "enabled, does not apply on clusterversion 4.19.30",
			clusterVersion: apiversion.NewVersion(4, 19, 30),
			clusterFlags:   copyFailEnabled,
		},
		{
			desc:                  "enabled, does apply on clusterversion 4.19.29 or earlier 4.19.z",
			clusterVersion:        apiversion.NewVersion(4, 19, 29),
			clusterFlags:          copyFailEnabled,
			expectedIsRequired:    true,
			expectedMachineConfig: expectedMasterMachineConfig(),
		},
		{
			desc:           "enabled, does not apply on clusterversion 4.18.40",
			clusterVersion: apiversion.NewVersion(4, 18, 40),
			clusterFlags:   copyFailEnabled,
		},
		{
			desc:                  "enabled, does apply on clusterversion 4.18.39 or earlier 4.18.z",
			clusterVersion:        apiversion.NewVersion(4, 18, 39),
			clusterFlags:          copyFailEnabled,
			expectedIsRequired:    true,
			expectedMachineConfig: expectedMasterMachineConfig(),
		},
		{
			desc:           "enabled, does not apply on clusterversion 4.17.53",
			clusterVersion: apiversion.NewVersion(4, 17, 53),
			clusterFlags:   copyFailEnabled,
		},
		{
			desc:                  "enabled, does apply on clusterversion 4.17.52 or earlier 4.17.z",
			clusterVersion:        apiversion.NewVersion(4, 17, 52),
			clusterFlags:          copyFailEnabled,
			expectedIsRequired:    true,
			expectedMachineConfig: expectedMasterMachineConfig(),
		},
		{
			desc:           "enabled, does not apply on clusterversion 4.16.61",
			clusterVersion: apiversion.NewVersion(4, 16, 61),
			clusterFlags:   copyFailEnabled,
		},
		{
			desc:                  "enabled, does apply on clusterversion 4.16.60 or earlier 4.16.z",
			clusterVersion:        apiversion.NewVersion(4, 16, 60),
			clusterFlags:          copyFailEnabled,
			expectedIsRequired:    true,
			expectedMachineConfig: expectedMasterMachineConfig(),
		},
		{
			desc:           "enabled, does not apply on clusterversion 4.15.64",
			clusterVersion: apiversion.NewVersion(4, 15, 64),
			clusterFlags:   copyFailEnabled,
		},
		{
			desc:                  "enabled, does apply on clusterversion 4.15.63 or earlier 4.15.z",
			clusterVersion:        apiversion.NewVersion(4, 15, 63),
			clusterFlags:          copyFailEnabled,
			expectedIsRequired:    true,
			expectedMachineConfig: expectedMasterMachineConfig(),
		},
		{
			desc:           "enabled, does not apply on clusterversion 4.14.65",
			clusterVersion: apiversion.NewVersion(4, 14, 65),
			clusterFlags:   copyFailEnabled,
		},
		{
			desc:                  "enabled, does apply on clusterversion 4.14.64 or earlier 4.14.z",
			clusterVersion:        apiversion.NewVersion(4, 14, 64),
			clusterFlags:          copyFailEnabled,
			expectedIsRequired:    true,
			expectedMachineConfig: expectedMasterMachineConfig(),
		},
		{
			desc:           "enabled, does not apply on clusterversion 4.13.66",
			clusterVersion: apiversion.NewVersion(4, 13, 66),
			clusterFlags:   copyFailEnabled,
		},
		{
			desc:                  "enabled, does apply on clusterversion 4.13.65 or earlier 4.13.z",
			clusterVersion:        apiversion.NewVersion(4, 13, 65),
			clusterFlags:          copyFailEnabled,
			expectedIsRequired:    true,
			expectedMachineConfig: expectedMasterMachineConfig(),
		},
		{
			desc:           "enabled, does not apply on clusterversion 4.12.89",
			clusterVersion: apiversion.NewVersion(4, 12, 89),
			clusterFlags:   copyFailEnabled,
		},
		{
			desc:                  "enabled, does apply on clusterversion 4.12.88 or earlier 4.12.z",
			clusterVersion:        apiversion.NewVersion(4, 12, 88),
			clusterFlags:          copyFailEnabled,
			expectedIsRequired:    true,
			expectedMachineConfig: expectedMasterMachineConfig(),
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

			clusterVersion := tC.clusterVersion
			if clusterVersion == nil {
				var err error
				clusterVersion, err = apiversion.ParseVersion("4.0.0")
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
