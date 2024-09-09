package cluster

import (
	"context"
	"errors"
	"testing"

	//	"github.com/Azure/ARO-RP/pkg/operator/clientset/versioned/scheme"
	mock_hive "github.com/Azure/ARO-RP/pkg/util/mocks/hive"
	mock_metrics "github.com/Azure/ARO-RP/pkg/util/mocks/metrics"
	"github.com/golang/mock/gomock"
	hivev1alpha1 "github.com/openshift/hive/apis/hiveinternal/v1alpha1"
	"github.com/stretchr/testify/assert"
	"k8s.io/client-go/kubernetes/scheme"
)

func init() {
	// Register the ClusterSync type with the scheme
	hivev1alpha1.AddToScheme(scheme.Scheme)
}

func TestEmitSyncSetStatus(t *testing.T) {
	for _, tt := range []struct {
		name              string
		clusterSync       *hivev1alpha1.ClusterSync
		getClusterSyncErr error
		expectedError     error
		expectedGauges    map[string]int64
		expectedLabels    map[string]string
	}{
		/*{
			name: "SyncSets has elements",
			clusterSync: &hivev1alpha1.ClusterSync{
				Status: hivev1alpha1.ClusterSyncStatus{
					SyncSets: []hivev1alpha1.SyncStatus{
						{
							Name:               "syncset1",
							Result:             "Success",
							FirstSuccessTime:   &metav1.Time{Time: time.Now()},
							LastTransitionTime: metav1.Time{Time: time.Now()},
							FailureMessage:     "",
						},
					},
				},
			},
			expectedError: nil,
			expectedGauges: map[string]int64{
				"syncsets.count": 1,
			},
			expectedLabels: map[string]string{
				"name":               "syncset1",
				"result":             "Success",
				"firstSuccessTime":   "2024-09-09T14:44:45Z",
				"lastTransitionTime": "2024-09-09T14:44:45Z",
				"failureMessage":     "",
			},
		},
		{
			name: "SelectorSyncSets is nil",
			clusterSync: &hivev1alpha1.ClusterSync{
				Status: hivev1alpha1.ClusterSyncStatus{
					SelectorSyncSets: nil,
				},
			},
			expectedError:  nil,
			expectedGauges: map[string]int64{"selectorsyncsets.count": 0},
		},*/
		{
			name:              "GetClusterSyncforClusterDeployment returns error",
			getClusterSyncErr: errors.New("some error"),
			expectedError:     nil,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockHiveClusterManager := mock_hive.NewMockClusterManager(ctrl)

			m := mock_metrics.NewMockEmitter(ctrl)
			mockMonitor := &Monitor{
				hiveClusterManager: mockHiveClusterManager,
				m:                  m,
			}

			ctx := context.Background()

			mockHiveClusterManager.EXPECT().GetSyncSetResources(ctx, mockMonitor.doc).Return(tt.clusterSync, tt.getClusterSyncErr).AnyTimes()

			if tt.expectedGauges != nil {
				for gauge, value := range tt.expectedGauges {
					m.EXPECT().EmitGauge(gauge, value, tt.expectedLabels).Times(1)
				}
			}
			err := mockMonitor.emitSyncSetStatus(ctx)
			assert.Equal(t, tt.expectedError, err)
		})
	}
}
