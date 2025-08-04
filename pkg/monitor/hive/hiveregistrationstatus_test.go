package hive

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kruntime "k8s.io/apimachinery/pkg/runtime"

	"sigs.k8s.io/controller-runtime/pkg/client"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"

	hivev1 "github.com/openshift/hive/apis/hive/v1"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/hive"
	mock_metrics "github.com/Azure/ARO-RP/pkg/util/mocks/metrics"
	utilerror "github.com/Azure/ARO-RP/test/util/error"
)

func TestEmitHiveRegistrationStatus(t *testing.T) {
	fakeNamespace := "fake-namespace"

	for _, tt := range []struct {
		name       string
		oc         *api.OpenShiftCluster
		cd         kruntime.Object
		withClient bool
		wantErr    string
		wantLog    string
	}{
		{
			name:       "no hiveclient",
			withClient: false,
			wantLog:    "skipping: no hive cluster manager",
		},
		{
			name:       "no namespace in cosmosDB - not adopted yet",
			withClient: true,
			oc: &api.OpenShiftCluster{
				Name: "testcluster",
				Properties: api.OpenShiftClusterProperties{
					HiveProfile: api.HiveProfile{
						Namespace: "",
					},
				},
			},
			wantErr: "cluster testcluster not adopted. No namespace in the clusterdocument",
		},
		{
			name:       "clusterdeployment can not be retrieved",
			withClient: true,
			oc: &api.OpenShiftCluster{
				Name: "testcluster",
				Properties: api.OpenShiftClusterProperties{
					HiveProfile: api.HiveProfile{
						Namespace: fakeNamespace,
					},
				},
			},
			wantErr: "clusterdeployments.hive.openshift.io \"cluster\" not found",
		},
		{
			name:       "send metrics data",
			withClient: true,
			oc: &api.OpenShiftCluster{
				Name: "testcluster",
				Properties: api.OpenShiftClusterProperties{
					HiveProfile: api.HiveProfile{
						Namespace: fakeNamespace,
					},
				},
			},
			cd: &hivev1.ClusterDeployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      hive.ClusterDeploymentName,
					Namespace: fakeNamespace,
				},
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var hiveclient client.Client
			if tt.withClient {
				fakeclient := fakeclient.NewClientBuilder()
				if tt.cd != nil {
					fakeclient = fakeclient.WithRuntimeObjects(tt.cd)
				}
				hiveclient = fakeclient.Build()
			}

			logger, hook := test.NewNullLogger()
			log := logrus.NewEntry(logger)

			mon := &Monitor{
				hiveclientset: hiveclient,
				oc:            tt.oc,
				log:           log,
			}

			err := mon.emitHiveRegistrationStatus(context.Background())
			utilerror.AssertErrorMessage(t, err, tt.wantErr)

			if tt.wantLog != "" {
				x := hook.LastEntry()
				assert.Equal(t, tt.wantLog, x.Message)
			}
		})
	}
}

func TestFilterClusterDeploymentConditions(t *testing.T) {
	var testConditionList = map[hivev1.ClusterDeploymentConditionType]corev1.ConditionStatus{
		hivev1.ClusterReadyCondition: corev1.ConditionTrue,
		hivev1.UnreachableCondition:  corev1.ConditionFalse,
	}

	for _, tt := range []struct {
		name               string
		cd                 *hivev1.ClusterDeployment
		expectedConditions []hivev1.ClusterDeploymentCondition
	}{
		{
			name: "irrelevant condition",
			cd: &hivev1.ClusterDeployment{
				Status: hivev1.ClusterDeploymentStatus{
					Conditions: []hivev1.ClusterDeploymentCondition{
						{
							Type:   hivev1.ClusterHibernatingCondition,
							Status: corev1.ConditionTrue,
						},
					},
				},
			},
			expectedConditions: []hivev1.ClusterDeploymentCondition{},
		},
		{
			name: "mixed conditions",
			cd: &hivev1.ClusterDeployment{
				Status: hivev1.ClusterDeploymentStatus{
					Conditions: []hivev1.ClusterDeploymentCondition{
						{ //should be ignored
							Type:   hivev1.ClusterHibernatingCondition,
							Status: corev1.ConditionTrue,
						},
						{ //should be ignored
							Type:   hivev1.UnreachableCondition,
							Status: corev1.ConditionFalse,
						},
						{ //should be returned
							Type:   hivev1.ClusterReadyCondition,
							Status: corev1.ConditionFalse,
						},
					},
				},
			},
			expectedConditions: []hivev1.ClusterDeploymentCondition{
				{
					Type:   hivev1.ClusterReadyCondition,
					Status: corev1.ConditionFalse,
				},
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			mon := &Monitor{}
			conditions := mon.filterClusterDeploymentConditions(context.Background(), tt.cd, testConditionList)
			for _, c := range conditions {
				isExpected := false
				for _, ex := range tt.expectedConditions {
					if ex.Type == c.Type {
						isExpected = true
						break
					}
				}
				if !isExpected {
					t.Errorf("condition %s should not be returned", c.Type)
				}
			}

			for _, ex := range tt.expectedConditions {
				isReturned := false
				for _, c := range conditions {
					if ex.Type == c.Type {
						isReturned = true
						break
					}
				}
				if !isReturned {
					t.Errorf("expected condition %s not returned", ex.Type)
				}
			}
		})
	}
}

func TestEmitFilteredClusterDeploymentMetrics(t *testing.T) {
	conditions := []hivev1.ClusterDeploymentCondition{
		{
			Type:   hivev1.ClusterHibernatingCondition,
			Reason: "test",
		},
		{
			Type:   hivev1.ClusterReadyCondition,
			Reason: "second test",
		},
	}

	controller := gomock.NewController(t)
	defer controller.Finish()

	m := mock_metrics.NewMockEmitter(controller)
	mon := &Monitor{
		m: m,
	}

	for _, c := range conditions {
		m.EXPECT().EmitGauge("hive.clusterdeployment.conditions", int64(1), map[string]string{
			"type":   string(c.Type),
			"reason": c.Reason,
		})
	}

	mon.emitFilteredClusterDeploymentMetrics(conditions)
}
