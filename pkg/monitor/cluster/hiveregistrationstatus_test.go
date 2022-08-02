package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"testing"

	"github.com/golang/mock/gomock"
	hivev1 "github.com/openshift/hive/apis/hive/v1"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"

	"github.com/Azure/ARO-RP/pkg/api"
	mock_hiveclient "github.com/Azure/ARO-RP/pkg/util/mocks/hive/clientset/versioned"
	mock_hiveclientv1 "github.com/Azure/ARO-RP/pkg/util/mocks/hive/clientset/versioned/typed/hive/v1"
	mock_metrics "github.com/Azure/ARO-RP/pkg/util/mocks/metrics"
)

func TestEmitHiveRegistrationStatus(t *testing.T) {
	ctx := context.Background()

	controller := gomock.NewController(t)

	logger, hook := test.NewNullLogger()
	log := logrus.NewEntry(logger)

	m := mock_metrics.NewMockEmitter(controller)

	oc := &api.OpenShiftCluster{
		Name: "testcluster",
		Properties: api.OpenShiftClusterProperties{
			HiveProfile: api.HiveProfile{
				Namespace: "",
			},
		},
	}

	mon := &Monitor{
		hiveclientset: nil,
		m:             m,
		oc:            oc,
		log:           log,
	}

	// no hive client set
	err := mon.emitHiveRegistrationStatus(ctx)
	if err != nil {
		t.Fatal("should not fail")
	}
	x := hook.LastEntry()
	assert.Equal(t, "skipping: no hive cluster manager", x.Message)

	// no namespace
	hiveclientset := mock_hiveclient.NewMockInterface(controller)
	mon.hiveclientset = hiveclientset
	err = mon.emitHiveRegistrationStatus(ctx)
	if err == nil {
		t.Fatal("should not fail")
	}
	if err.Error() != "cluster testcluster not adopted. No namespace in the clusterdocument" {
		t.Fatal("Expecting error")
	}

	// happy path
	mon.oc.Properties.HiveProfile.Namespace = "abc"

	cds := mock_hiveclientv1.NewMockClusterDeploymentInterface(controller)
	v1 := mock_hiveclientv1.NewMockHiveV1Interface(controller)
	cd := &hivev1.ClusterDeployment{}
	hiveclientset.EXPECT().HiveV1().Return(v1)
	v1.EXPECT().ClusterDeployments(gomock.Any()).Return(cds)
	cds.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).Return(cd, nil)

	err = mon.emitHiveRegistrationStatus(ctx)
	if err != nil {
		t.Fatal("should not fail")
	}
}

func TestValidateHiveConditions(t *testing.T) {
	ctx := context.Background()

	for _, tt := range []struct {
		name        string
		cd          *hivev1.ClusterDeployment
		expectError error
	}{{
		name:        "Regular case",
		cd:          &hivev1.ClusterDeployment{},
		expectError: nil,
	}, {

		name:        "no cluster deployment found in hive",
		cd:          nil,
		expectError: fmt.Errorf("not found error message"),
	}} {
		controller := gomock.NewController(t)

		hiveclientset := mock_hiveclient.NewMockInterface(controller)
		cds := mock_hiveclientv1.NewMockClusterDeploymentInterface(controller)
		v1 := mock_hiveclientv1.NewMockHiveV1Interface(controller)

		hiveclientset.EXPECT().HiveV1().Return(v1)
		v1.EXPECT().ClusterDeployments(gomock.Any()).Return(cds)
		cds.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).Return(tt.cd, tt.expectError)

		mon := &Monitor{
			hiveclientset: hiveclientset,
			oc: &api.OpenShiftCluster{
				Name: "testcluster",
				Properties: api.OpenShiftClusterProperties{
					HiveProfile: api.HiveProfile{
						Namespace: "abc",
					},
				},
			},
		}

		err := mon.validateHiveConditions(ctx)
		if err != tt.expectError {
			t.Fatal("Unexpected error handling")
		}
	}
}

func TestFilterConditions(t *testing.T) {
	var testConditionList = map[hivev1.ClusterDeploymentConditionType]corev1.ConditionStatus{
		hivev1.ClusterReadyCondition:  corev1.ConditionTrue,
		hivev1.UnreachableCondition:   corev1.ConditionFalse,
		hivev1.SyncSetFailedCondition: corev1.ConditionFalse,
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
		ctx := context.Background()

		mon := &Monitor{}
		conditions := mon.filterConditions(ctx, tt.cd, testConditionList)
		for _, c := range conditions {
			isExpected := false
			for _, ex := range tt.expectedConditions {
				if ex.Type == c.Type {
					isExpected = true
					break
				}
			}
			if !isExpected {
				t.Fatalf("condition %s should not be returned", c.Type)
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
				t.Fatalf("expected condition %s not returned", ex.Type)
			}
		}
	}
}

func TestEmitMetrics(t *testing.T) {
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

	mon.emitMetrics(conditions)
}
