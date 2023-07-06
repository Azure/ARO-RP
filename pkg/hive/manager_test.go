package hive

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"net/http"
	"reflect"
	"testing"

	hivev1 "github.com/openshift/hive/apis/hive/v1"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kruntime "k8s.io/apimachinery/pkg/runtime"
	kubernetesfake "k8s.io/client-go/kubernetes/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/util/cmp"
	"github.com/Azure/ARO-RP/pkg/util/uuid"
	uuidfake "github.com/Azure/ARO-RP/pkg/util/uuid/fake"
	utilerror "github.com/Azure/ARO-RP/test/util/error"
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
			fakeClientBuilder := fake.NewClientBuilder()
			if tt.cd != nil {
				fakeClientBuilder.WithRuntimeObjects(tt.cd)
			}
			c := clusterManager{
				hiveClientset: fakeClientBuilder.Build(),
				log:           logrus.NewEntry(logrus.StandardLogger()),
			}

			result, err := c.IsClusterDeploymentReady(context.Background(), doc)
			utilerror.AssertErrorMessage(t, err, tt.wantErr)

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
		cd         *hivev1.ClusterDeployment
		cp         *hivev1.ClusterProvision
		wantResult bool
		wantErr    error
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
			name: "has failed provisioning - no Reason",
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
			wantErr: &api.CloudError{
				StatusCode: http.StatusInternalServerError,
				CloudErrorBody: &api.CloudErrorBody{
					Code:    api.CloudErrorCodeInternalServerError,
					Message: "Deployment failed.",
				},
			},
			wantResult: false,
		},
		// TODO: move test cases for handleProvisionFailed here
	} {
		t.Run(tt.name, func(t *testing.T) {
			fakeClientBuilder := fake.NewClientBuilder()
			if tt.cd != nil {
				fakeClientBuilder = fakeClientBuilder.WithRuntimeObjects(tt.cd)
			}
			if tt.cp != nil {
				fakeClientBuilder = fakeClientBuilder.WithRuntimeObjects(tt.cp)
			}
			c := clusterManager{
				hiveClientset: fakeClientBuilder.Build(),
				log:           logrus.NewEntry(logrus.StandardLogger()),
			}

			result, err := c.IsClusterInstallationComplete(context.Background(), doc)

			if diff := cmp.Diff(tt.wantErr, err); diff != "" {
				t.Error(diff)
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
			fakeClientBuilder := fake.NewClientBuilder()
			if tt.cd != nil {
				fakeClientBuilder = fakeClientBuilder.WithRuntimeObjects(tt.cd)
			}
			c := clusterManager{
				hiveClientset: fakeClientBuilder.Build(),
			}

			err := c.ResetCorrelationData(context.Background(), doc)
			utilerror.AssertErrorMessage(t, err, tt.wantErr)

			if err == nil {
				cd, err := c.GetClusterDeployment(context.Background(), doc)
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
				uuid.DefaultGenerator = uuidfake.NewGenerator(tc.nsNames)
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
			fakeClientBuilder := fake.NewClientBuilder()
			if tt.wantErr == "" {
				fakeClientBuilder = fakeClientBuilder.WithRuntimeObjects(cd)
			}
			c := clusterManager{
				hiveClientset: fakeClientBuilder.Build(),
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

func TestHandleProvisionFailed(t *testing.T) {
	fakeNamespace := "aro-00000000-0000-0000-0000-00000000000"
	genericErr := &api.CloudError{
		StatusCode: http.StatusInternalServerError,
		CloudErrorBody: &api.CloudErrorBody{
			Code:    api.CloudErrorCodeInternalServerError,
			Message: "Deployment failed.",
		},
	}

	for _, tt := range []struct {
		name       string
		reason     string
		installLog string
		wantErr    error
	}{
		{
			name:    "No Reason provided returns generic error",
			reason:  "",
			wantErr: genericErr,
		},
		{
			name:    "Known Reason not relevant to ARO returns generic error",
			reason:  "AWSInsufficientCapacity",
			wantErr: genericErr,
		},
		{
			name:    "Reason: UnknownError returns generic error",
			reason:  ProvisionFailedReasonUnknownError,
			wantErr: genericErr,
		},
		{
			name:   "Reason: InvalidTemplateDeployment extracts error from logs",
			reason: ProvisionFailedReasonInvalidTemplateDeployment,
			installLog: `level=info msg=running in local development mode
			level=info msg=creating development InstanceMetadata
			level=info msg=InstanceMetadata: running on AzurePublicCloud
			level=info msg=running step [Action github.com/Azure/ARO-RP/pkg/installer.(*manager).Manifests.func1]
			level=info msg=running step [Action github.com/Azure/ARO-RP/pkg/installer.(*manager).Manifests.func2]
			level=info msg=resolving graph
			level=info msg=running step [Action github.com/Azure/ARO-RP/pkg/installer.(*manager).Manifests.func3]
			level=info msg=checking if graph exists
			level=info msg=save graph
			Generates the Ignition Config asset
			
			level=info msg=running in local development mode
			level=info msg=creating development InstanceMetadata
			level=info msg=InstanceMetadata: running on AzurePublicCloud
			level=info msg=running step [AuthorizationRefreshingAction [Action github.com/Azure/ARO-RP/pkg/installer.(*manager).deployResourceTemplate-fm]]
			level=info msg=load persisted graph
			level=info msg=deploying resources template
			level=error msg=step [AuthorizationRefreshingAction [Action github.com/Azure/ARO-RP/pkg/installer.(*manager).deployResourceTemplate-fm]] encountered error: 400: DeploymentFailed: : Deployment failed. Details: : : {"code": "InvalidTemplateDeployment","message": "The template deployment failed with multiple errors. Please see details for more information.","target": null,"details": [{"code": "RequestDisallowedByPolicy","message": "Resource 'aro-test-aaaaa-bootstrap' was disallowed by policy.","target": "aro-test-aaaaa-bootstrap"},{"code": "RequestDisallowedByPolicy","message": "Resource 'aro-test-aaaaa-master-0' was disallowed by policy.","target": "aro-test-aaaaa-master-0"},{"code": "RequestDisallowedByPolicy","message": "Resource 'aro-test-aaaaa-master-1' was disallowed by policy.","target": "aro-test-aaaaa-master-1"},{"code": "RequestDisallowedByPolicy","message": "Resource 'aro-test-aaaaa-master-2' was disallowed by policy.","target": "aro-test-aaaaa-master-2"}]}
			level=error msg=400: DeploymentFailed: : Deployment failed. Details: : : {"code": "InvalidTemplateDeployment","message": "The template deployment failed with multiple errors. Please see details for more information.","target": null,"details": [{"code": "RequestDisallowedByPolicy","message": "Resource 'aro-test-aaaaa-bootstrap' was disallowed by policy.","target": "aro-test-aaaaa-bootstrap"},{"code": "RequestDisallowedByPolicy","message": "Resource 'aro-test-aaaaa-master-0' was disallowed by policy.","target": "aro-test-aaaaa-master-0"},{"code": "RequestDisallowedByPolicy","message": "Resource 'aro-test-aaaaa-master-1' was disallowed by policy.","target": "aro-test-aaaaa-master-1"},{"code": "RequestDisallowedByPolicy","message": "Resource 'aro-test-aaaaa-master-2' was disallowed by policy.","target": "aro-test-aaaaa-master-2"}]}`,
			wantErr: &api.CloudError{
				StatusCode: http.StatusBadRequest,
				CloudErrorBody: &api.CloudErrorBody{
					Code:    api.CloudErrorCodeDeploymentFailed,
					Message: "The deployment failed. Please see details for more information.",
					Details: []api.CloudErrorBody{
						{
							Code:    api.CloudErrorCodeRequestDisallowedByPolicy,
							Message: "Resource 'aro-test-aaaaa-bootstrap' was disallowed by policy.",
							Target:  "aro-test-aaaaa-bootstrap",
						},
						{
							Code:    api.CloudErrorCodeRequestDisallowedByPolicy,
							Message: "Resource 'aro-test-aaaaa-master-0' was disallowed by policy.",
							Target:  "aro-test-aaaaa-master-0",
						},
						{
							Code:    api.CloudErrorCodeRequestDisallowedByPolicy,
							Message: "Resource 'aro-test-aaaaa-master-1' was disallowed by policy.",
							Target:  "aro-test-aaaaa-master-1",
						},
						{
							Code:    api.CloudErrorCodeRequestDisallowedByPolicy,
							Message: "Resource 'aro-test-aaaaa-master-2' was disallowed by policy.",
							Target:  "aro-test-aaaaa-master-2",
						},
					},
				},
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			cond := hivev1.ClusterDeploymentCondition{
				Type:   hivev1.ProvisionFailedCondition,
				Status: corev1.ConditionTrue,
				Reason: tt.reason,
			}
			hcd := &hivev1.ClusterDeployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      ClusterDeploymentName,
					Namespace: fakeNamespace,
				},
				Status: hivev1.ClusterDeploymentStatus{
					Conditions: []hivev1.ClusterDeploymentCondition{cond},
				},
			}
			hcp := &hivev1.ClusterProvision{
				ObjectMeta: metav1.ObjectMeta{
					Name:      ClusterDeploymentName + "-0-bbbbb",
					Namespace: fakeNamespace,
					Labels: map[string]string{
						"hive.openshift.io/cluster-deployment-name": ClusterDeploymentName,
					},
				},
				Spec: hivev1.ClusterProvisionSpec{
					InstallLog: &tt.installLog,
				},
			}

			fakeClientBuilder := fake.NewClientBuilder().
				WithRuntimeObjects(hcd, hcp)

			c := clusterManager{
				hiveClientset: fakeClientBuilder.Build(),
				log:           logrus.NewEntry(logrus.StandardLogger()),
			}

			err := c.handleProvisionFailed(context.Background(), hcd, cond)

			if diff := cmp.Diff(tt.wantErr, err); diff != "" {
				t.Error(diff)
			}
		})
	}
}
