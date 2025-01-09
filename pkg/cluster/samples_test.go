package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"testing"

	operatorv1 "github.com/openshift/api/operator/v1"
	samplesv1 "github.com/openshift/api/samples/v1"
	"github.com/sirupsen/logrus"
	"go.uber.org/mock/gomock"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/Azure/ARO-RP/pkg/api"
	mock_env "github.com/Azure/ARO-RP/pkg/util/mocks/env"
	mock_samples "github.com/Azure/ARO-RP/pkg/util/mocks/samples"
	mock_samplesclient "github.com/Azure/ARO-RP/pkg/util/mocks/samplesclient"
	utilerror "github.com/Azure/ARO-RP/test/util/error"
)

func Test_manager_disableSamples(t *testing.T) {
	ctx := context.Background()
	samplesConfig := &samplesv1.Config{
		TypeMeta:   metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{},
		Spec:       samplesv1.ConfigSpec{},
		Status:     samplesv1.ConfigStatus{},
	}
	tests := []struct {
		name                        string
		samplesConfig               *samplesv1.Config
		samplesCRGetError           error
		samplesCRUpdateError        error
		expectedMinNumberOfGetCalls int
		expectedMaxNumberOfGetCalls int
		wantErr                     string
	}{
		{
			name:                        "samples cr is found and updated",
			samplesConfig:               samplesConfig,
			expectedMinNumberOfGetCalls: 1,
			expectedMaxNumberOfGetCalls: 1,
			wantErr:                     "",
		},
		{
			name:                        "samples cr is not found and retried",
			samplesCRGetError:           kerrors.NewNotFound(schema.GroupResource{}, "samples"),
			expectedMinNumberOfGetCalls: 2,
			expectedMaxNumberOfGetCalls: 15,
			wantErr:                     "",
		},
		{
			name:                        "samples cr update is conflicting and retried",
			samplesConfig:               samplesConfig,
			expectedMinNumberOfGetCalls: 2,
			expectedMaxNumberOfGetCalls: 15,
			samplesCRUpdateError:        kerrors.NewConflict(schema.GroupResource{}, "samples", errors.New("conflict")),
			wantErr:                     "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()
			env := mock_env.NewMockInterface(controller)
			samplescli := mock_samplesclient.NewMockInterface(controller)
			samplesInterface := mock_samples.NewMockSamplesV1Interface(controller)
			configInterface := mock_samples.NewMockConfigInterface(controller)

			env.EXPECT().IsLocalDevelopmentMode().Return(false)
			samplescli.EXPECT().SamplesV1().AnyTimes().Return(samplesInterface)
			samplesInterface.EXPECT().Configs().AnyTimes().Return(configInterface)
			configInterface.EXPECT().Get(gomock.Any(), "cluster", metav1.GetOptions{}).
				MinTimes(tt.expectedMinNumberOfGetCalls).
				MaxTimes(tt.expectedMaxNumberOfGetCalls).
				Return(tt.samplesConfig, tt.samplesCRGetError)

			if tt.samplesConfig != nil {
				samplesConfig.Spec.ManagementState = operatorv1.Removed
				configInterface.EXPECT().Update(gomock.Any(), samplesConfig, metav1.UpdateOptions{}).AnyTimes().Return(samplesConfig, tt.samplesCRUpdateError)
			}

			m := &manager{
				log: logrus.NewEntry(logrus.StandardLogger()),
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
			utilerror.AssertErrorMessage(t, err, tt.wantErr)
		})
	}
}
