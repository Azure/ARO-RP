package cluster

import (
	"context"
	"sync"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	operatorv1 "github.com/openshift/api/operator/v1"
	operatorfake "github.com/openshift/client-go/operator/clientset/versioned/fake"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	mock_metrics "github.com/Azure/ARO-RP/pkg/util/mocks/metrics"
)

func TestEmitClusterAuthenticationType(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockMetrics := mock_metrics.NewMockEmitter(ctrl)
	fakeOperatorClient := operatorfake.NewSimpleClientset()

	mon := &Monitor{
		operatorcli: fakeOperatorClient,
		m:           mockMetrics,
		log:         logrus.NewEntry(logrus.New()),
		wg:          &sync.WaitGroup{},
	}

	_, err := fakeOperatorClient.OperatorV1().CloudCredentials().Create(context.Background(), &operatorv1.CloudCredential{
		ObjectMeta: metav1.ObjectMeta{
			Name: "cluster",
		},
		Spec: operatorv1.CloudCredentialSpec{
			CredentialsMode: operatorv1.CloudCredentialsModeDefault,
		},
	}, metav1.CreateOptions{})
	require.NoError(t, err)

	tests := []struct {
		name            string
		credentialsMode operatorv1.CloudCredentialsMode
		expectMetric    map[string]string
		expectErr       bool
	}{
		{
			name:            "credentials mode is Manual",
			credentialsMode: operatorv1.CloudCredentialsModeManual,
			expectMetric: map[string]string{
				"type": "managedIdentity",
			},
			expectErr: false,
		},
		{
			name:            "credentials mode is ''",
			credentialsMode: operatorv1.CloudCredentialsModeDefault,
			expectMetric: map[string]string{
				"type": "servicePrincipal",
			},
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cloudCredential := &operatorv1.CloudCredential{
				ObjectMeta: metav1.ObjectMeta{
					Name: "cluster",
				},
				Spec: operatorv1.CloudCredentialSpec{
					CredentialsMode: tt.credentialsMode,
				},
			}

			_, err := fakeOperatorClient.OperatorV1().CloudCredentials().Update(context.Background(), cloudCredential, metav1.UpdateOptions{})
			require.NoError(t, err)

			mockMetrics.EXPECT().
				EmitGauge(authenticationTypeMetricsTopic, int64(1), tt.expectMetric).
				Times(1)

			err = mon.emitClusterAuthenticationType(context.Background())

			if tt.expectErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
