package hive

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"reflect"
	"testing"

	hivev1 "github.com/openshift/hive/apis/hive/v1"
	hivefake "github.com/openshift/hive/pkg/client/clientset/versioned/fake"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kruntime "k8s.io/apimachinery/pkg/runtime"
	kubernetesfake "k8s.io/client-go/kubernetes/fake"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/util/cmp"
	"github.com/Azure/ARO-RP/pkg/util/uuid"
	"github.com/Azure/ARO-RP/pkg/util/uuid/fake"
)

func TestIsClusterDeploymentReady(t *testing.T) {
	fakeNamespace := "fake-namespace"
	doc := &api.OpenShiftClusterDocument{
		OpenShiftCluster: &api.OpenShiftCluster{
			Properties: api.OpenShiftClusterProperties{
				HiveProfile: api.HiveProfile{
					Namespace: fakeNamespace,
				},
			},
		},
	}

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
							Type:   hivev1.ProvisionedCondition,
							Status: corev1.ConditionTrue,
						},
						{
							Type:   hivev1.SyncSetFailedCondition,
							Status: corev1.ConditionFalse,
						},
						{
							Type:   hivev1.ControlPlaneCertificateNotFoundCondition,
							Status: corev1.ConditionFalse,
						},
						{
							Type:   hivev1.UnreachableCondition,
							Status: corev1.ConditionFalse,
						},
					},
				},
			},
			wantResult: true,
		},
		{
			name: "is not ready: unreachable",
			cd: &hivev1.ClusterDeployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      ClusterDeploymentName,
					Namespace: fakeNamespace,
				},
				Status: hivev1.ClusterDeploymentStatus{
					Conditions: []hivev1.ClusterDeploymentCondition{
						{
							Type:   hivev1.ProvisionedCondition,
							Status: corev1.ConditionTrue,
						},
						{
							Type:   hivev1.SyncSetFailedCondition,
							Status: corev1.ConditionFalse,
						},
						{
							Type:   hivev1.ControlPlaneCertificateNotFoundCondition,
							Status: corev1.ConditionFalse,
						},
						{
							Type:   hivev1.UnreachableCondition,
							Status: corev1.ConditionTrue,
						},
					},
				},
			},
			wantResult: false,
		},
		{
			name: "is not ready: syncset failed",
			cd: &hivev1.ClusterDeployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      ClusterDeploymentName,
					Namespace: fakeNamespace,
				},
				Status: hivev1.ClusterDeploymentStatus{
					Conditions: []hivev1.ClusterDeploymentCondition{
						{
							Type:   hivev1.ProvisionedCondition,
							Status: corev1.ConditionTrue,
						},
						{
							Type:   hivev1.SyncSetFailedCondition,
							Status: corev1.ConditionTrue,
						},
						{
							Type:   hivev1.ControlPlaneCertificateNotFoundCondition,
							Status: corev1.ConditionFalse,
						},
						{
							Type:   hivev1.UnreachableCondition,
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
				log:           logrus.NewEntry(logrus.StandardLogger()),
			}

			result, err := c.IsClusterDeploymentReady(context.Background(), doc)
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

func TestIsClusterInstallationComplete(t *testing.T) {
	fakeNamespace := "fake-namespace"
	doc := &api.OpenShiftClusterDocument{
		OpenShiftCluster: &api.OpenShiftCluster{
			Properties: api.OpenShiftClusterProperties{
				HiveProfile: api.HiveProfile{
					Namespace: fakeNamespace,
				},
			},
		},
	}

	for _, tt := range []struct {
		name       string
		cd         kruntime.Object
		wantResult bool
		wantErr    string
	}{
		{
			name: "is installed",
			cd: &hivev1.ClusterDeployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      ClusterDeploymentName,
					Namespace: fakeNamespace,
				},
				Spec: hivev1.ClusterDeploymentSpec{
					Installed: true,
				},
				Status: hivev1.ClusterDeploymentStatus{
					Conditions: []hivev1.ClusterDeploymentCondition{
						{
							Type:   hivev1.ProvisionFailedCondition,
							Status: corev1.ConditionFalse,
						},
					},
				},
			},
			wantResult: true,
		},
		{
			name: "is not installed yet",
			cd: &hivev1.ClusterDeployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      ClusterDeploymentName,
					Namespace: fakeNamespace,
				},
				Spec: hivev1.ClusterDeploymentSpec{
					Installed: false,
				},
				Status: hivev1.ClusterDeploymentStatus{
					Conditions: []hivev1.ClusterDeploymentCondition{
						{
							Type:   hivev1.ProvisionFailedCondition,
							Status: corev1.ConditionFalse,
						},
					},
				},
			},
			wantResult: false,
		},
		{
			name: "has failed provisioning",
			cd: &hivev1.ClusterDeployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      ClusterDeploymentName,
					Namespace: fakeNamespace,
				},
				Status: hivev1.ClusterDeploymentStatus{
					Conditions: []hivev1.ClusterDeploymentCondition{
						{
							Type:   hivev1.ProvisionFailedCondition,
							Status: corev1.ConditionTrue,
						},
					},
				},
			},
			wantErr:    "clusterdeployment has failed: ProvisionFailed == True",
			wantResult: false,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			fakeClientset := hivefake.NewSimpleClientset()
			if tt.cd != nil {
				fakeClientset.Tracker().Add(tt.cd)
			}
			c := clusterManager{
				hiveClientset: fakeClientset,
				log:           logrus.NewEntry(logrus.StandardLogger()),
			}

			result, err := c.IsClusterInstallationComplete(context.Background(), doc)
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
	doc := &api.OpenShiftClusterDocument{
		OpenShiftCluster: &api.OpenShiftCluster{
			Properties: api.OpenShiftClusterProperties{
				HiveProfile: api.HiveProfile{
					Namespace: fakeNamespace,
				},
			},
		},
	}

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

			err := c.ResetCorrelationData(context.Background(), doc)
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

func TestCreateNamespace(t *testing.T) {
	for _, tc := range []struct {
		name             string
		nsNames          []string
		useFakeGenerator bool
		shouldFail       bool
	}{
		{
			name:             "not conflict names and real generator",
			nsNames:          []string{"namespace1", "namespace2"},
			useFakeGenerator: false,
			shouldFail:       false,
		},
		{
			name:             "conflict names and real generator",
			nsNames:          []string{"namespace", "namespace"},
			useFakeGenerator: false,
			shouldFail:       false,
		},
		{
			name:             "not conflict names and fake generator",
			nsNames:          []string{"namespace1", "namespace2"},
			useFakeGenerator: true,
			shouldFail:       false,
		},
		{
			name:             "conflict names and fake generator",
			nsNames:          []string{"namespace", "namespace"},
			useFakeGenerator: true,
			shouldFail:       true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fakeClientset := kubernetesfake.NewSimpleClientset()
			c := clusterManager{
				kubernetescli: fakeClientset,
			}

			if tc.useFakeGenerator {
				uuid.DefaultGenerator = fake.NewGenerator(tc.nsNames)
			}

			ns, err := c.CreateNamespace(context.Background())
			if err != nil && !tc.shouldFail {
				t.Error(err)
			}

			if err == nil {
				res, err := fakeClientset.CoreV1().Namespaces().Get(context.Background(), ns.Name, metav1.GetOptions{})
				if err != nil {
					t.Error(err)
				}
				if !reflect.DeepEqual(ns, res) {
					t.Errorf("results are not equal: \n wanted: %+v \n got %+v", ns, res)
				}
			}
		})
	}
}

func TestGetClusterDeployment(t *testing.T) {
	fakeNamespace := "fake-namespace"
	doc := &api.OpenShiftClusterDocument{
		OpenShiftCluster: &api.OpenShiftCluster{
			Properties: api.OpenShiftClusterProperties{
				HiveProfile: api.HiveProfile{
					Namespace: fakeNamespace,
				},
			},
		},
	}

	cd := &hivev1.ClusterDeployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ClusterDeploymentName,
			Namespace: fakeNamespace,
		},
	}

	for _, tt := range []struct {
		name    string
		wantErr string
	}{
		{name: "cd exists and is returned"},
		{name: "cd does not exist err returned", wantErr: `clusterdeployments.hive.openshift.io "cluster" not found`},
	} {
		t.Run(tt.name, func(t *testing.T) {
			fakeClientset := hivefake.NewSimpleClientset()
			if tt.wantErr == "" {
				_ = fakeClientset.Tracker().Add(cd)
			}
			c := clusterManager{
				hiveClientset: fakeClientset,
				log:           logrus.NewEntry(logrus.StandardLogger()),
			}

			result, err := c.GetClusterDeployment(context.Background(), doc)
			if err != nil && err.Error() != tt.wantErr ||
				err == nil && tt.wantErr != "" {
				t.Fatal(err)
			}

			if result != nil && result.Name != cd.Name && result.Namespace != cd.Namespace {
				t.Fatal("Unexpected cluster deployment returned", result)
			}
		})
	}
}
