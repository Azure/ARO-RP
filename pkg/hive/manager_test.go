package hive

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"reflect"
	"testing"

	hivev1 "github.com/openshift/hive/apis/hive/v1"
	hivefake "github.com/openshift/hive/pkg/client/clientset/versioned/fake"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kruntime "k8s.io/apimachinery/pkg/runtime"

	"github.com/Azure/ARO-RP/pkg/util/cmp"
)

func TestIsClusterDeploymentReady(t *testing.T) {
	fakeNamespace := "fake-namespace"

	for _, tt := range []struct {
		name       string
		cd         kruntime.Object
		wantResult bool
		wantErr    string
	}{
		{
			name: "is ready",
			cd: &hivev1.ClusterDeployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      ClusterDeploymentName,
					Namespace: fakeNamespace,
				},
				Status: hivev1.ClusterDeploymentStatus{
					Conditions: []hivev1.ClusterDeploymentCondition{
						{
							Type:   hivev1.ClusterReadyCondition,
							Status: corev1.ConditionTrue,
						},
					},
				},
			},
			wantResult: true,
		},
		{
			name: "is not ready",
			cd: &hivev1.ClusterDeployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      ClusterDeploymentName,
					Namespace: fakeNamespace,
				},
				Status: hivev1.ClusterDeploymentStatus{
					Conditions: []hivev1.ClusterDeploymentCondition{
						{
							Type:   hivev1.ClusterReadyCondition,
							Status: corev1.ConditionFalse,
						},
					},
				},
			},
			wantResult: false,
		},
		{
			name: "is not ready - condition is missing",
			cd: &hivev1.ClusterDeployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      ClusterDeploymentName,
					Namespace: fakeNamespace,
				},
			},
			wantResult: false,
		},
		{
			name:       "error - ClusterDeployment is missing",
			wantResult: false,
			wantErr:    "clusterdeployments.hive.openshift.io \"cluster\" not found",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			fakeClientset := hivefake.NewSimpleClientset()
			if tt.cd != nil {
				fakeClientset.Tracker().Add(tt.cd)
			}
			c := clusterManager{
				hiveClientset: fakeClientset,
			}

			result, err := c.IsClusterDeploymentReady(context.Background(), fakeNamespace)
			if err != nil && err.Error() != tt.wantErr ||
				err == nil && tt.wantErr != "" {
				t.Error(err)
			}

			if tt.wantResult != result {
				t.Error(result)
			}
		})
	}
}

func TestResetCorrelationData(t *testing.T) {
	fakeNamespace := "fake-namespace"

	for _, tt := range []struct {
		name            string
		cd              kruntime.Object
		wantAnnotations map[string]string
		wantErr         string
	}{
		{
			name: "success",
			cd: &hivev1.ClusterDeployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      ClusterDeploymentName,
					Namespace: fakeNamespace,
					Annotations: map[string]string{
						"hive.openshift.io/additional-log-fields": `{
							"correlation_id": "existing-fake-correlation-id"
						}`,
					},
				},
			},
			wantAnnotations: map[string]string{
				"hive.openshift.io/additional-log-fields": "{}",
			},
		},
		{
			name:    "error - ClusterDeployment is missing",
			wantErr: "clusterdeployments.hive.openshift.io \"cluster\" not found",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			fakeClientset := hivefake.NewSimpleClientset()
			if tt.cd != nil {
				fakeClientset.Tracker().Add(tt.cd)
			}
			c := clusterManager{
				hiveClientset: fakeClientset,
			}

			err := c.ResetCorrelationData(context.Background(), fakeNamespace)
			if err != nil && err.Error() != tt.wantErr ||
				err == nil && tt.wantErr != "" {
				t.Error(err)
			}

			if err == nil {
				cd, err := c.hiveClientset.HiveV1().ClusterDeployments(fakeNamespace).Get(context.Background(), ClusterDeploymentName, metav1.GetOptions{})
				if err != nil {
					t.Fatal(err)
				}
				if !reflect.DeepEqual(tt.wantAnnotations, cd.Annotations) {
					t.Error(cmp.Diff(tt.wantAnnotations, cd.Annotations))
				}
			}
		})
	}
}
