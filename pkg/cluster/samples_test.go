package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"testing"

	"github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
	"go.uber.org/mock/gomock"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kruntime "k8s.io/apimachinery/pkg/runtime"

	operatorv1 "github.com/openshift/api/operator/v1"
	samplesv1 "github.com/openshift/api/samples/v1"
	samplesfake "github.com/openshift/client-go/samples/clientset/versioned/fake"

	"github.com/Azure/ARO-RP/pkg/api"
	mock_env "github.com/Azure/ARO-RP/pkg/util/mocks/env"
	utilerror "github.com/Azure/ARO-RP/test/util/error"
	testlog "github.com/Azure/ARO-RP/test/util/log"
)

func Test_manager_disableSamples(t *testing.T) {
	ctx := context.Background()
	samplesConfig := &samplesv1.Config{
		ObjectMeta: metav1.ObjectMeta{
			Name: "cluster",
		},
		Spec:   samplesv1.ConfigSpec{},
		Status: samplesv1.ConfigStatus{},
	}
	tests := []struct {
		name                   string
		samplesConfig          *samplesv1.Config
		isLocalDevelopmentMode bool
		pullSecret             string
		wantedLog              testlog.ExpectedLogEntry
		wantErr                string
	}{
		{
			name:                   "Running in local development mode, samples disabled successfully",
			isLocalDevelopmentMode: true,
			pullSecret:             "",
			wantedLog: testlog.ExpectedLogEntry{
				"level": gomega.Equal(logrus.InfoLevel),
				"msg":   gomega.Equal("Running in local development mode, disabling samples"),
			},
		},
		{
			name:                   "No pull secret found, samples disabled successfully",
			isLocalDevelopmentMode: false,
			pullSecret:             "",
			wantedLog: testlog.ExpectedLogEntry{
				"level": gomega.Equal(logrus.InfoLevel),
				"msg":   gomega.Equal("No pull secret found, disabling samples"),
			},
		},
		{
			name:                   "Samples CR is found and updated successfully",
			samplesConfig:          samplesConfig,
			isLocalDevelopmentMode: false,
			pullSecret:             "",
		},
		{
			name:                   "Samples CR is not found, creates new resource with management state removed",
			isLocalDevelopmentMode: false,
			pullSecret:             "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			objects := []kruntime.Object{}
			if tt.samplesConfig != nil {
				objects = append(objects, tt.samplesConfig)
			}

			samplescli := samplesfake.NewSimpleClientset(objects...)

			env := mock_env.NewMockInterface(controller)
			env.EXPECT().IsLocalDevelopmentMode().Return(tt.isLocalDevelopmentMode)

			h, log := testlog.New()
			m := &manager{
				log: log,
				env: env,
				doc: &api.OpenShiftClusterDocument{
					OpenShiftCluster: &api.OpenShiftCluster{
						Properties: api.OpenShiftClusterProperties{
							ClusterProfile: api.ClusterProfile{
								ResourceGroupID: "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/clusterRGName",
								PullSecret:      api.SecureString(tt.pullSecret),
							},
						},
					},
				},
				samplescli: samplescli,
			}

			err := m.disableSamples(ctx)
			utilerror.AssertErrorMessage(t, err, tt.wantErr)
			testlog.AssertLoggingOutput(h, []testlog.ExpectedLogEntry{tt.wantedLog})
			got, err := samplescli.SamplesV1().Configs().Get(ctx, "cluster", metav1.GetOptions{})
			if err != nil {
				t.Fatal(err)
			}

			if got.Spec.ManagementState != operatorv1.Removed {
				t.Errorf("wanted ManagementState %s but got %s", operatorv1.Removed, got.Spec.ManagementState)
			}
		})
	}
}
