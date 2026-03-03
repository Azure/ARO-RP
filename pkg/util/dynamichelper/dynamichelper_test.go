package dynamichelper

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"net/http"
	"reflect"
	"testing"

	"github.com/sirupsen/logrus"

	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apiresource "k8s.io/apimachinery/pkg/api/resource"
	kruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/cli-runtime/pkg/resource"
	"k8s.io/client-go/rest/fake"

	"github.com/Azure/ARO-RP/pkg/util/cmp"
	"github.com/Azure/ARO-RP/pkg/util/pointerutils"
)

type mockGVRResolver struct{}

func (gvr mockGVRResolver) Refresh() error {
	return nil
}

func (gvr mockGVRResolver) Resolve(groupKind, optionalVersion string) (*schema.GroupVersionResource, error) {
	return &schema.GroupVersionResource{Group: "metal3.io", Version: "v1alpha1", Resource: "configmap"}, nil
}

func TestEsureDeleted(t *testing.T) {
	ctx := context.Background()

	mockGVRResolver := mockGVRResolver{}

	mockRestCLI := &fake.RESTClient{
		GroupVersion:         schema.GroupVersion{Group: "testgroup", Version: "v1"},
		NegotiatedSerializer: resource.UnstructuredPlusDefaultContentConfig().NegotiatedSerializer,
		Client: fake.CreateHTTPClient(func(req *http.Request) (*http.Response, error) {
			switch req.Method {
			case http.MethodDelete:
				switch req.URL.Path {
				case "/apis/metal3.io/v1alpha1/namespaces/test-ns-1/configmap/test-name-1":
					return &http.Response{StatusCode: http.StatusNotFound}, nil
				case "/apis/metal3.io/v1alpha1/namespaces/test-ns-2/configmap/test-name-2":
					return &http.Response{StatusCode: http.StatusInternalServerError}, nil
				case "/apis/metal3.io/v1alpha1/namespaces/test-ns-3/configmap/test-name-3":
					return &http.Response{StatusCode: http.StatusOK}, nil
				default:
					t.Fatalf("unexpected path: %#v\n%#v", req.URL, req)
					return nil, nil
				}
			default:
				t.Fatalf("unexpected request: %s %#v\n%#v", req.Method, req.URL, req)
				return nil, nil
			}
		}),
	}

	dh := &dynamicHelper{
		GVRResolver: mockGVRResolver,
		restcli:     mockRestCLI,
		log:         logrus.NewEntry(logrus.StandardLogger()),
	}

	err := dh.EnsureDeleted(ctx, "configmap", "test-ns-1", "test-name-1")
	if err != nil {
		t.Errorf("no error should be bounced for status not found, but got: %v", err)
	}

	err = dh.EnsureDeleted(ctx, "configmap", "test-ns-2", "test-name-2")
	if err == nil {
		t.Errorf("function should handle failure response (non-404) correctly")
	}

	err = dh.EnsureDeleted(ctx, "configmap", "test-ns-3", "test-name-3")
	if err != nil {
		t.Errorf("function should handle success response correctly")
	}
}

func TestMakeURLSegments(t *testing.T) {
	for _, tt := range []struct {
		gvr         *schema.GroupVersionResource
		namespace   string
		uname, name string
		url         []string
		want        []string
	}{
		{
			uname: "Group is empty",
			gvr: &schema.GroupVersionResource{
				Group:    "",
				Version:  "4.10",
				Resource: "test-resource",
			},
			namespace: "openshift",
			name:      "test-name-1",
			want:      []string{"api", "4.10", "namespaces", "openshift", "test-resource", "test-name-1"},
		},
		{
			uname: "Group is not empty",
			gvr: &schema.GroupVersionResource{
				Group:    "test-group",
				Version:  "4.10",
				Resource: "test-resource",
			},
			namespace: "openshift-apiserver",
			name:      "test-name-2",
			want:      []string{"apis", "test-group", "4.10", "namespaces", "openshift-apiserver", "test-resource", "test-name-2"},
		},
		{
			uname: "Namespace is empty",
			gvr: &schema.GroupVersionResource{
				Group:    "test-group",
				Version:  "4.10",
				Resource: "test-resource",
			},
			namespace: "",
			name:      "test-name-3",
			want:      []string{"apis", "test-group", "4.10", "test-resource", "test-name-3"},
		},
		{
			uname: "Namespace is not empty",
			gvr: &schema.GroupVersionResource{
				Group:    "test-group",
				Version:  "4.10",
				Resource: "test-resource",
			},
			namespace: "openshift-sdn",
			name:      "test-name-3",
			want:      []string{"apis", "test-group", "4.10", "namespaces", "openshift-sdn", "test-resource", "test-name-3"},
		},
		{
			uname: "Name is empty",
			gvr: &schema.GroupVersionResource{
				Group:    "test-group",
				Version:  "4.10",
				Resource: "test-resource",
			},
			namespace: "openshift-ns",
			name:      "",
			want:      []string{"apis", "test-group", "4.10", "namespaces", "openshift-ns", "test-resource"},
		},
	} {
		t.Run(tt.uname, func(t *testing.T) {
			got := makeURLSegments(tt.gvr, tt.namespace, tt.name)
			if !reflect.DeepEqual(got, tt.want) {
				t.Error(cmp.Diff(got, tt.want))
			}
		})
	}
}

func TestMergeGK(t *testing.T) {
	for _, tt := range []struct {
		name          string
		old           kruntime.Object
		new           kruntime.Object
		want          kruntime.Object
		wantChanged   bool
		wantEmptyDiff bool
	}{
		{
			name: "Deployment changes",
			old: &appsv1.Deployment{
				Spec: appsv1.DeploymentSpec{
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Image: "nginx:1.0.1",
									Resources: corev1.ResourceRequirements{
										Limits: corev1.ResourceList{
											"cpu":    apiresource.MustParse("1"),
											"memory": apiresource.MustParse("12Mi"),
										},
										Requests: corev1.ResourceList{
											"cpu":    apiresource.MustParse("2"),
											"memory": apiresource.MustParse("10Mi"),
										},
									},
								},
							},
						},
					},
				},
			},
			new: &appsv1.Deployment{
				Spec: appsv1.DeploymentSpec{
					Replicas: pointerutils.ToPtr(int32(1)),
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Image: "nginx:2.0.1",
									Resources: corev1.ResourceRequirements{
										Limits: corev1.ResourceList{
											"cpu":    apiresource.MustParse("1.5"),
											"memory": apiresource.MustParse("512Mi"),
										},
										Requests: corev1.ResourceList{
											"cpu":    apiresource.MustParse("0.5"),
											"memory": apiresource.MustParse("100Mi"),
										},
									},
								},
							},
						},
					},
				},
			},
			want: &appsv1.Deployment{
				Spec: appsv1.DeploymentSpec{
					Replicas: pointerutils.ToPtr(int32(1)),
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							RestartPolicy:                 "Always",
							TerminationGracePeriodSeconds: pointerutils.ToPtr(int64(corev1.DefaultTerminationGracePeriodSeconds)),
							DNSPolicy:                     "ClusterFirst",
							SecurityContext:               &corev1.PodSecurityContext{},
							SchedulerName:                 "default-scheduler",
							Containers: []corev1.Container{
								{
									Image: "nginx:2.0.1",
									Resources: corev1.ResourceRequirements{
										Limits: corev1.ResourceList{
											"cpu":    apiresource.MustParse("1.5"),
											"memory": apiresource.MustParse("512Mi"),
										},
										Requests: corev1.ResourceList{
											"cpu":    apiresource.MustParse("0.5"),
											"memory": apiresource.MustParse("100Mi"),
										},
									},
									TerminationMessagePath:   "/dev/termination-log",
									TerminationMessagePolicy: "File",
									ImagePullPolicy:          "IfNotPresent",
								},
							},
						},
					},
					Strategy: appsv1.DeploymentStrategy{
						Type: appsv1.RollingUpdateDeploymentStrategyType,
						RollingUpdate: &appsv1.RollingUpdateDeployment{
							MaxUnavailable: &intstr.IntOrString{
								Type:   1,
								StrVal: "25%",
							},
							MaxSurge: &intstr.IntOrString{
								Type:   1,
								StrVal: "25%",
							},
						},
					},
					RevisionHistoryLimit:    pointerutils.ToPtr(int32(10)),
					ProgressDeadlineSeconds: pointerutils.ToPtr(int32(600)),
				},
			},
			wantChanged: true,
		},
		{
			name: "ValidatingWebhookConfiguration changes",
			old: &admissionregistrationv1.ValidatingWebhookConfiguration{
				Webhooks: []admissionregistrationv1.ValidatingWebhook{
					{
						FailurePolicy: getFailurePolicyType("Fail"),
					},
				},
			},
			new: &admissionregistrationv1.ValidatingWebhookConfiguration{
				Webhooks: []admissionregistrationv1.ValidatingWebhook{
					{
						FailurePolicy: getFailurePolicyType("Ignore"),
					},
				},
			},
			want: &admissionregistrationv1.ValidatingWebhookConfiguration{
				Webhooks: []admissionregistrationv1.ValidatingWebhook{
					{
						FailurePolicy: getFailurePolicyType("Ignore"),
					},
				},
			},
			wantChanged: true,
		},
		{
			name: "ValidatingWebhookConfiguration no changes",
			old: &admissionregistrationv1.ValidatingWebhookConfiguration{
				Webhooks: []admissionregistrationv1.ValidatingWebhook{
					{
						FailurePolicy: getFailurePolicyType("Ignore"),
					},
				},
			},
			new: &admissionregistrationv1.ValidatingWebhookConfiguration{
				Webhooks: []admissionregistrationv1.ValidatingWebhook{
					{
						FailurePolicy: getFailurePolicyType("Ignore"),
					},
				},
			},
			want: &admissionregistrationv1.ValidatingWebhookConfiguration{
				Webhooks: []admissionregistrationv1.ValidatingWebhook{
					{
						FailurePolicy: getFailurePolicyType("Ignore"),
					},
				},
			},
			wantChanged:   false,
			wantEmptyDiff: true,
		},
		{
			name: "Secret changes, not logged",
			old: &corev1.Secret{
				Data: map[string][]byte{
					"secret": []byte("old"),
				},
			},
			new: &corev1.Secret{
				Data: map[string][]byte{
					"secret": []byte("new"),
				},
			},
			want: &corev1.Secret{
				Data: map[string][]byte{
					"secret": []byte("old"),
				},
				Type: corev1.SecretTypeOpaque,
			},
			wantChanged:   false,
			wantEmptyDiff: true,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got, changed, diff, err := mergeGK(tt.old, tt.new)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Error(cmp.Diff(got, tt.want))
			}
			if changed != tt.wantChanged {
				t.Error(changed)
			}
			if diff == "" != tt.wantEmptyDiff {
				t.Error(diff)
			}
		})
	}
}

func getFailurePolicyType(in string) *admissionregistrationv1.FailurePolicyType {
	fail := admissionregistrationv1.FailurePolicyType(in)
	return &fail
}
