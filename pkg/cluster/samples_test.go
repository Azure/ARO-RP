package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"testing"

	operatorv1 "github.com/openshift/api/operator/v1"
	samplesv1 "github.com/openshift/api/samples/v1"
	samplesfake "github.com/openshift/client-go/samples/clientset/versioned/fake"
	"go.uber.org/mock/gomock"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kruntime "k8s.io/apimachinery/pkg/runtime"

	"github.com/Azure/ARO-RP/pkg/api"
	mock_env "github.com/Azure/ARO-RP/pkg/util/mocks/env"
	utilerror "github.com/Azure/ARO-RP/test/util/error"
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
		name          string
		samplesConfig *samplesv1.Config
		wantErr       string
	}{
		{
			name:          "samples cr is found and updated",
			samplesConfig: samplesConfig,
			wantErr:       "",
		},
		{
			name:    "samples cr is not found and retried",
			wantErr: `configs.samples.operator.openshift.io "cluster" not found`,
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
			env.EXPECT().IsLocalDevelopmentMode().Return(false)

			m := &manager{
				env: env,
				doc: &api.OpenShiftClusterDocument{
					OpenShiftCluster: &api.OpenShiftCluster{
						Properties: api.OpenShiftClusterProperties{
							ClusterProfile: api.ClusterProfile{
								ResourceGroupID: "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/clusterRGName",
							},
						},
					},
				},
				samplescli: samplescli,
			}

			err := m.disableSamples(ctx)

			if tt.samplesConfig != nil {
				got, err := samplescli.SamplesV1().Configs().Get(ctx, "cluster", metav1.GetOptions{})
				if err != nil {
					t.Error(err)
				}

				if got.Spec.ManagementState != operatorv1.Removed {
					t.Errorf("wanted ManagementState %s but got %s", operatorv1.Removed, got.Spec.ManagementState)
				}
			}
			utilerror.AssertErrorMessage(t, err, tt.wantErr)
		})
	}
}
