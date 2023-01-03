package deploy

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"reflect"
	"testing"

	"github.com/golang/mock/gomock"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/util/cmp"
	mock_env "github.com/Azure/ARO-RP/pkg/util/mocks/env"
	"github.com/Azure/ARO-RP/test/util/matcher"
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

			matcher.AssertErrHasWantMsg(t, err, tt.wantErrMsg)

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
		expected                deploymentData
	}{
		{
			name: "no image override, use default",
			mock: func(env *mock_env.MockInterface, oc *api.OpenShiftCluster) {
				env.EXPECT().
					AROOperatorImage().
					Return(operatorImageWithTag)
			},
			expected: deploymentData{
				Image:   operatorImageWithTag,
				Version: operatorImageTag},
		},
		{
			name: "no image tag, use latest version",
			mock: func(env *mock_env.MockInterface, oc *api.OpenShiftCluster) {
				env.EXPECT().
					AROOperatorImage().
					Return(operatorImageUntagged)
			},
			expected: deploymentData{
				Image:   operatorImageUntagged,
				Version: "latest"},
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
			expected: deploymentData{
				Image:   "docker.io/aro:override",
				Version: "override"},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			env := mock_env.NewMockInterface(controller)
			env.EXPECT().IsLocalDevelopmentMode().Return(tt.expected.IsLocalDevelopment).AnyTimes()

			oc := &api.OpenShiftCluster{Properties: api.OpenShiftClusterProperties{}}
			tt.mock(env, oc)

			o := operator{
				oc:  oc,
				env: env,
			}

			deploymentData := o.createDeploymentData()
			if !reflect.DeepEqual(deploymentData, tt.expected) {
				t.Errorf("actual deployment: %v, expected %v", deploymentData, tt.expected)
			}
		})
	}
}

func TestOperatorVersion(t *testing.T) {
	type test struct {
		name         string
		oc           func() *api.OpenShiftClusterProperties
		wantVersion  string
		wantPullspec string
	}

	for _, tt := range []*test{
		{
			name: "default",
			oc: func() *api.OpenShiftClusterProperties {
				return &api.OpenShiftClusterProperties{}
			},
			wantVersion:  "latest",
			wantPullspec: "defaultaroimagefromenv",
		},
		{
			name: "overridden",
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
			oc := tt.oc()

			controller := gomock.NewController(t)
			defer controller.Finish()

			_env := mock_env.NewMockInterface(controller)
			_env.EXPECT().ACRDomain().AnyTimes().Return("intsvcdomain")
			_env.EXPECT().AROOperatorImage().AnyTimes().Return("defaultaroimagefromenv")
			_env.EXPECT().IsLocalDevelopmentMode().AnyTimes().Return(false)

			o := &operator{
				oc:  &api.OpenShiftCluster{Properties: *oc},
				env: _env,
			}

			staticResources, err := o.createObjects()
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
			matcher.AssertErrHasWantMsg(t, err, tt.wantErrMsg)

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
			matcher.AssertErrHasWantMsg(t, err, tt.wantErrMsg)

			if tt.want != got {
				t.Fatalf("error with CheckPodImageVersion test %s: got %v wanted %v", tt.name, got, tt.want)
			}
		})
	}
}
