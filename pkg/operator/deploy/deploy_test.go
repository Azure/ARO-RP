package deploy

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"path/filepath"
	"reflect"
	"testing"

	configv1 "github.com/openshift/api/config/v1"
	operatorv1 "github.com/openshift/api/operator/v1"
	operatorfake "github.com/openshift/client-go/operator/clientset/versioned/fake"
	"github.com/sirupsen/logrus"
	"go.uber.org/mock/gomock"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	ctrlfake "sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/yaml"

	"github.com/Azure/ARO-RP/pkg/api"
	pkgoperator "github.com/Azure/ARO-RP/pkg/operator"
	"github.com/Azure/ARO-RP/pkg/util/clienthelper"
	"github.com/Azure/ARO-RP/pkg/util/cmp"
	mock_env "github.com/Azure/ARO-RP/pkg/util/mocks/env"
	utilerror "github.com/Azure/ARO-RP/test/util/error"
)

func TestCheckIngressIP(t *testing.T) {
	type test struct {
		name       string
		oc         func() *api.OpenShiftClusterProperties
		want       string
		wantErrMsg string
	}

	for _, tt := range []*test{
		{
			name: "default IngressProfile",
			oc: func() *api.OpenShiftClusterProperties {
				return &api.OpenShiftClusterProperties{
					IngressProfiles: []api.IngressProfile{
						{
							Name: "default",
							IP:   "1.2.3.4",
						},
					},
				}
			},
			want: "1.2.3.4",
		},
		{
			name: "Multiple IngressProfiles, pick default",
			oc: func() *api.OpenShiftClusterProperties {
				return &api.OpenShiftClusterProperties{
					IngressProfiles: []api.IngressProfile{
						{
							Name: "custom-ingress",
							IP:   "1.1.1.1",
						},
						{
							Name: "default",
							IP:   "1.2.3.4",
						},
						{
							Name: "not-default",
							IP:   "1.1.2.2",
						},
					},
				}
			},
			want: "1.2.3.4",
		},
		{
			name: "Single Ingress Profile, No Default",
			oc: func() *api.OpenShiftClusterProperties {
				return &api.OpenShiftClusterProperties{
					IngressProfiles: []api.IngressProfile{
						{
							Name: "custom-ingress",
							IP:   "1.1.1.1",
						},
					},
				}
			},
			want: "1.1.1.1",
		},
		{
			name: "Multiple Ingress Profiles, No Default",
			oc: func() *api.OpenShiftClusterProperties {
				return &api.OpenShiftClusterProperties{
					IngressProfiles: []api.IngressProfile{
						{
							Name: "custom-ingress",
							IP:   "1.1.1.1",
						},
						{
							Name: "not-default",
							IP:   "1.1.2.2",
						},
					},
				}
			},
			want: "1.1.1.1",
		},
		{
			name: "No Ingresses in IngressProfiles, Error",
			oc: func() *api.OpenShiftClusterProperties {
				return &api.OpenShiftClusterProperties{
					IngressProfiles: []api.IngressProfile{},
				}
			},
			wantErrMsg: "no Ingress Profiles found",
		},
		{
			name: "Nil IngressProfiles, Error",
			oc: func() *api.OpenShiftClusterProperties {
				return &api.OpenShiftClusterProperties{}
			},
			wantErrMsg: "no Ingress Profiles found",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			oc := tt.oc()
			ingressIP, err := checkIngressIP(oc.IngressProfiles)

			utilerror.AssertErrorMessage(t, err, tt.wantErrMsg)

			if tt.want != ingressIP {
				t.Error(cmp.Diff(ingressIP, tt.want))
			}
		})
	}
}

func TestCreateDeploymentData(t *testing.T) {
	operatorImageTag := "v20071110"
	operatorImageUntagged := "arosvc.azurecr.io/aro"
	operatorImageWithTag := operatorImageUntagged + ":" + operatorImageTag

	for _, tt := range []struct {
		name                    string
		mock                    func(*mock_env.MockInterface, *api.OpenShiftCluster)
		operatorVersionOverride string
		clusterVersion          string
		expected                deploymentData
		wantErr                 string
	}{
		{
			name: "no image override, use default",
			mock: func(env *mock_env.MockInterface, oc *api.OpenShiftCluster) {
				env.EXPECT().
					AROOperatorImage().
					Return(operatorImageWithTag)
			},
			clusterVersion: "4.10.0",
			expected: deploymentData{
				Image:   operatorImageWithTag,
				Version: operatorImageTag,
			},
		},
		{
			name: "no image tag, use latest version",
			mock: func(env *mock_env.MockInterface, oc *api.OpenShiftCluster) {
				env.EXPECT().
					AROOperatorImage().
					Return(operatorImageUntagged)
			},
			clusterVersion: "4.10.0",
			expected: deploymentData{
				Image:   operatorImageUntagged,
				Version: "latest",
			},
		},
		{
			name: "OperatorVersion override set",
			mock: func(env *mock_env.MockInterface, oc *api.OpenShiftCluster) {
				env.EXPECT().
					AROOperatorImage().
					Return(operatorImageUntagged)
				env.EXPECT().
					ACRDomain().
					Return("docker.io")

				oc.Properties.OperatorVersion = "override"
			},
			clusterVersion: "4.10.0",
			expected: deploymentData{
				Image:   "docker.io/aro:override",
				Version: "override",
			},
		},
		{
			name: "version supports pod security admission",
			mock: func(env *mock_env.MockInterface, oc *api.OpenShiftCluster) {
				env.EXPECT().
					AROOperatorImage().
					Return(operatorImageWithTag)
			},
			clusterVersion: "4.11.0",
			expected: deploymentData{
				Image:                        operatorImageWithTag,
				Version:                      operatorImageTag,
				SupportsPodSecurityAdmission: true,
			},
		},
		{
			name: "workload identity detected",
			mock: func(env *mock_env.MockInterface, oc *api.OpenShiftCluster) {
				env.EXPECT().
					AROOperatorImage().
					Return(operatorImageWithTag)
				// set so that UsesWorkloadIdentity() returns true
				oc.Properties.PlatformWorkloadIdentityProfile = &api.PlatformWorkloadIdentityProfile{}
			},
			clusterVersion: "4.10.0",
			expected: deploymentData{
				Image:                  operatorImageWithTag,
				Version:                operatorImageTag,
				UsesWorkloadIdentity:   true,
				TokenVolumeMountPath:   filepath.Dir(pkgoperator.OperatorTokenFile),
				FederatedTokenFilePath: pkgoperator.OperatorTokenFile,
			},
		},
		{
			name: "service principal detected",
			mock: func(env *mock_env.MockInterface, oc *api.OpenShiftCluster) {
				env.EXPECT().
					AROOperatorImage().
					Return(operatorImageWithTag)
				// set so that UsesWorkloadIdentity() returns false
				oc.Properties.ServicePrincipalProfile = &api.ServicePrincipalProfile{}
			},
			clusterVersion: "4.10.0",
			expected: deploymentData{
				Image:                operatorImageWithTag,
				Version:              operatorImageTag,
				UsesWorkloadIdentity: false,
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			controller := gomock.NewController(t)
			defer controller.Finish()

			env := mock_env.NewMockInterface(controller)
			env.EXPECT().IsLocalDevelopmentMode().Return(tt.expected.IsLocalDevelopment).AnyTimes()

			oc := &api.OpenShiftCluster{Properties: api.OpenShiftClusterProperties{}}
			tt.mock(env, oc)

			cv := &configv1.ClusterVersion{
				ObjectMeta: metav1.ObjectMeta{
					Name: "version",
				},
				Spec: configv1.ClusterVersionSpec{},
				Status: configv1.ClusterVersionStatus{
					History: []configv1.UpdateHistory{
						{
							State:   configv1.CompletedUpdate,
							Version: tt.clusterVersion,
						},
					},
				},
			}

			o := operator{
				oc:     oc,
				env:    env,
				client: clienthelper.NewWithClient(logrus.NewEntry(logrus.StandardLogger()), ctrlfake.NewClientBuilder().WithObjects(cv).Build()),
			}

			deploymentData, err := o.createDeploymentData(ctx)
			utilerror.AssertErrorMessage(t, err, tt.wantErr)
			if !reflect.DeepEqual(deploymentData, tt.expected) {
				t.Errorf("actual deployment: %v, expected %v", deploymentData, tt.expected)
			}
		})
	}
}

func TestOperatorVersion(t *testing.T) {
	type test struct {
		name           string
		clusterVersion string
		oc             func() *api.OpenShiftClusterProperties
		wantVersion    string
		wantPullspec   string
	}

	for _, tt := range []*test{
		{
			name:           "default",
			clusterVersion: "4.10.0",
			oc: func() *api.OpenShiftClusterProperties {
				return &api.OpenShiftClusterProperties{}
			},
			wantVersion:  "latest",
			wantPullspec: "defaultaroimagefromenv",
		},
		{
			name:           "overridden",
			clusterVersion: "4.10.0",
			oc: func() *api.OpenShiftClusterProperties {
				return &api.OpenShiftClusterProperties{
					OperatorVersion: "v20220101.0",
				}
			},
			wantVersion:  "v20220101.0",
			wantPullspec: "intsvcdomain/aro:v20220101.0",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			oc := tt.oc()

			controller := gomock.NewController(t)
			defer controller.Finish()

			_env := mock_env.NewMockInterface(controller)
			_env.EXPECT().ACRDomain().AnyTimes().Return("intsvcdomain")
			_env.EXPECT().AROOperatorImage().AnyTimes().Return("defaultaroimagefromenv")
			_env.EXPECT().IsLocalDevelopmentMode().AnyTimes().Return(false)

			cv := &configv1.ClusterVersion{
				ObjectMeta: metav1.ObjectMeta{
					Name: "version",
				},
				Spec: configv1.ClusterVersionSpec{},
				Status: configv1.ClusterVersionStatus{
					History: []configv1.UpdateHistory{
						{
							State:   configv1.CompletedUpdate,
							Version: tt.clusterVersion,
						},
					},
				},
			}

			o := &operator{
				oc:     &api.OpenShiftCluster{Properties: *oc},
				env:    _env,
				client: clienthelper.NewWithClient(logrus.NewEntry(logrus.StandardLogger()), ctrlfake.NewClientBuilder().WithObjects(cv).Build()),
			}

			staticResources, err := o.createObjects(ctx)
			if err != nil {
				t.Error(err)
			}

			var deployments []*appsv1.Deployment
			for _, i := range staticResources {
				if d, ok := i.(*appsv1.Deployment); ok {
					deployments = append(deployments, d)
				}
			}

			if len(deployments) != 2 {
				t.Errorf("found %d deployments, not 2", len(deployments))
			}

			for _, d := range deployments {
				if d.Labels["version"] != tt.wantVersion {
					t.Errorf("Got %q, not %q for label \"version\"", d.Labels["version"], tt.wantVersion)
				}

				if len(d.Spec.Template.Spec.Containers) != 1 {
					t.Errorf("found %d containers, not 1", len(d.Spec.Template.Spec.Containers))
				}

				image := d.Spec.Template.Spec.Containers[0].Image
				if image != tt.wantPullspec {
					t.Errorf("Got %q, not %q for the image", image, tt.wantPullspec)
				}
			}
		})
	}
}

func TestCheckOperatorDeploymentVersion(t *testing.T) {
	ctx := context.Background()
	for _, tt := range []struct {
		name           string
		deployment     *appsv1.Deployment
		desiredVersion string
		want           bool
		wantErrMsg     string
	}{
		{
			name: "arooperator deployment has correct version",
			deployment: &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "arooperator-deploy",
					Namespace: "openshift-azure-operator",
					Labels: map[string]string{
						"version": "abcde",
					},
				},
			},
			desiredVersion: "abcde",
			want:           true,
		},
		{
			name: "arooperator deployment has incorrect version",
			deployment: &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "arooperator-deploy",
					Namespace: "openshift-azure-operator",
					Labels: map[string]string{
						"version": "unknown",
					},
				},
			},
			desiredVersion: "abcde",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			clientset := fake.NewSimpleClientset()
			_, err := clientset.AppsV1().Deployments("openshift-azure-operator").Create(ctx, tt.deployment, metav1.CreateOptions{})
			if err != nil {
				t.Fatalf("error creating deployment: %v", err)
			}

			got, err := checkOperatorDeploymentVersion(ctx, clientset.AppsV1().Deployments("openshift-azure-operator"), tt.deployment.Name, tt.desiredVersion)
			utilerror.AssertErrorMessage(t, err, tt.wantErrMsg)

			if tt.want != got {
				t.Fatalf("error with CheckOperatorDeploymentVersion test %s: got %v wanted %v", tt.name, got, tt.want)
			}
		})
	}
}

func TestCheckPodImageVersion(t *testing.T) {
	ctx := context.Background()
	for _, tt := range []struct {
		name           string
		pod            *corev1.Pod
		desiredVersion string
		want           bool
		wantErrMsg     string
	}{
		{
			name: "arooperator pod has correct image version",
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "arooperator-pod",
					Namespace: "openshift-azure-operator",
					Labels: map[string]string{
						"app": "arooperator-pod",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Image: "random-image:abcde",
						},
					},
				},
			},
			desiredVersion: "abcde",
			want:           true,
		},
		{
			name: "arooperator pod has incorrect image version",
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "arooperator-pod",
					Namespace: "openshift-azure-operator",
					Labels: map[string]string{
						"app": "arooperator-pod",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Image: "random-image:unknown",
						},
					},
				},
			},
			desiredVersion: "abcde",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			clientset := fake.NewSimpleClientset()
			_, err := clientset.CoreV1().Pods("openshift-azure-operator").Create(ctx, tt.pod, metav1.CreateOptions{})
			if err != nil {
				t.Fatalf("error creating pod: %v", err)
			}

			got, err := checkPodImageVersion(ctx, clientset.CoreV1().Pods("openshift-azure-operator"), tt.pod.Name, tt.desiredVersion)
			utilerror.AssertErrorMessage(t, err, tt.wantErrMsg)

			if tt.want != got {
				t.Fatalf("error with CheckPodImageVersion test %s: got %v wanted %v", tt.name, got, tt.want)
			}
		})
	}
}

func TestTestEnsureUpgradeAnnotation(t *testing.T) {
	UpgradeableTo1 := api.UpgradeableTo("4.14.59")

	for _, tt := range []struct {
		name                 string
		cluster              api.OpenShiftClusterProperties
		annotation           map[string]string
		wantAnnotation       map[string]string
		wantErr              string
		cloudCredentialsName string
	}{
		{
			name: "nil PlatformWorkloadIdentityProfile, no version persisted in cluster document",
		},
		{
			name: "non-nil ServicePrincipalProfile, no version persisted in cluster document",
			cluster: api.OpenShiftClusterProperties{
				ServicePrincipalProfile: &api.ServicePrincipalProfile{
					ClientID:     "",
					ClientSecret: "",
				},
				PlatformWorkloadIdentityProfile: &api.PlatformWorkloadIdentityProfile{},
			},
		},
		{
			name: "no version persisted in cluster document, persist it",
			cluster: api.OpenShiftClusterProperties{
				PlatformWorkloadIdentityProfile: &api.PlatformWorkloadIdentityProfile{
					UpgradeableTo: &UpgradeableTo1,
				},
			},
			annotation: nil,
			wantAnnotation: map[string]string{
				"cloudcredential.openshift.io/upgradeable-to": "4.14.59",
			},
		},
		{
			name: "cloud credential 'cluster' doesn't exist",
			cluster: api.OpenShiftClusterProperties{
				PlatformWorkloadIdentityProfile: &api.PlatformWorkloadIdentityProfile{
					UpgradeableTo: &UpgradeableTo1,
				},
			},
			cloudCredentialsName: "oh_no",
			annotation:           nil,
			wantAnnotation:       nil,
			wantErr:              `cloudcredentials.operator.openshift.io "cluster" not found`,
		},
		{
			name: "version persisted in cluster document, replace it",
			cluster: api.OpenShiftClusterProperties{
				PlatformWorkloadIdentityProfile: &api.PlatformWorkloadIdentityProfile{
					UpgradeableTo: &UpgradeableTo1,
				},
			},
			annotation: map[string]string{
				"cloudcredential.openshift.io/upgradeable-to": "4.14.02",
			},
			wantAnnotation: map[string]string{
				"cloudcredential.openshift.io/upgradeable-to": "4.14.59",
			},
		},
		{
			name: "annotations exist, append the upgradeable annotation",
			cluster: api.OpenShiftClusterProperties{
				PlatformWorkloadIdentityProfile: &api.PlatformWorkloadIdentityProfile{
					UpgradeableTo: &UpgradeableTo1,
				},
			},
			annotation: map[string]string{
				"foo": "bar",
			},
			wantAnnotation: map[string]string{
				"foo": "bar",
				"cloudcredential.openshift.io/upgradeable-to": "4.14.59",
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			controller := gomock.NewController(t)
			defer controller.Finish()

			env := mock_env.NewMockInterface(controller)

			oc := &api.OpenShiftCluster{
				Properties: tt.cluster,
			}

			if tt.cloudCredentialsName == "" {
				tt.cloudCredentialsName = "cluster"
			}

			cloudcredentialobject := &operatorv1.CloudCredential{
				ObjectMeta: metav1.ObjectMeta{
					Name:        tt.cloudCredentialsName,
					Annotations: tt.annotation,
				},
			}

			o := operator{
				oc:          oc,
				env:         env,
				operatorcli: operatorfake.NewSimpleClientset(cloudcredentialobject),
			}

			err := o.EnsureUpgradeAnnotation(ctx)
			utilerror.AssertErrorMessage(t, err, tt.wantErr)
			result, _ := o.operatorcli.OperatorV1().CloudCredentials().List(ctx, metav1.ListOptions{})
			for _, v := range result.Items {
				actualAnnotations := v.ObjectMeta.Annotations
				if !reflect.DeepEqual(actualAnnotations, tt.wantAnnotation) {
					t.Errorf("actual annotation: %v, wanted %v", tt.annotation, tt.wantAnnotation)
				}
			}
		})
	}
}

func TestGenerateOperatorIdentitySecret(t *testing.T) {
	tests := []struct {
		Name           string
		Operator       *operator
		ExpectedSecret *corev1.Secret
	}{
		{
			Name: "valid cluster operator",
			Operator: &operator{
				oc: &api.OpenShiftCluster{
					Location: "eastus1",
					Properties: api.OpenShiftClusterProperties{
						PlatformWorkloadIdentityProfile: &api.PlatformWorkloadIdentityProfile{
							PlatformWorkloadIdentities: map[string]api.PlatformWorkloadIdentity{
								pkgoperator.OperatorIdentityName: {
									ClientID: "11111111-1111-1111-1111-111111111111",
								},
							},
						},
					},
				},
				subscriptiondoc: &api.SubscriptionDocument{
					ID: "00000000-0000-0000-0000-000000000000",
					Subscription: &api.Subscription{
						Properties: &api.SubscriptionProperties{
							TenantID: "22222222-2222-2222-2222-222222222222",
						},
					},
				},
			},
			ExpectedSecret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      pkgoperator.OperatorIdentitySecretName,
					Namespace: pkgoperator.Namespace,
				},
				// StringData is only converted to Data in live kubernetes
				StringData: map[string]string{
					"azure_client_id":            "11111111-1111-1111-1111-111111111111",
					"azure_tenant_id":            "22222222-2222-2222-2222-222222222222",
					"azure_region":               "eastus1",
					"azure_subscription_id":      "00000000-0000-0000-0000-000000000000",
					"azure_federated_token_file": pkgoperator.OperatorTokenFile,
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			actualSecret, err := test.Operator.generateOperatorIdentitySecret()
			if err != nil {
				t.Errorf("generateOperatorIdentitySecret() %s: unexpected error: %s\n", test.Name, err)
			}

			if !reflect.DeepEqual(actualSecret, test.ExpectedSecret) {
				t.Errorf("generateOperatorIdentitySecret() %s:\nexpected:\n%+v\n\ngot:\n%+v\n", test.Name, test.ExpectedSecret, actualSecret)
			}
		})
	}
}

func TestTemplateManifests(t *testing.T) {
	tests := []struct {
		Name           string
		DeploymentData deploymentData
	}{
		{
			Name: "service principal data",
			DeploymentData: deploymentData{
				Image:                        "someImage",
				Version:                      "someVersion",
				IsLocalDevelopment:           false,
				SupportsPodSecurityAdmission: false,
				UsesWorkloadIdentity:         false,
			},
		},
		{
			Name: "workload identity data",
			DeploymentData: deploymentData{
				Image:                        "someImage",
				Version:                      "someVersion",
				IsLocalDevelopment:           false,
				SupportsPodSecurityAdmission: false,
				UsesWorkloadIdentity:         true,
			},
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			actualBytes, err := templateManifests(test.DeploymentData)
			if err != nil {
				t.Errorf("templateManifests() %s: unexpected error: %s\n", test.Name, err)
			}

			for _, fileBytes := range actualBytes {
				var resource *kruntime.Object
				err := yaml.Unmarshal(fileBytes, resource)

				if err != nil {
					t.Errorf("templateManifests() %s: unexpected error: %s\n", test.Name, err)
				}
			}
		})
	}
}
