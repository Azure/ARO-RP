package clusterdata

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"reflect"
	"testing"

	operatorv1 "github.com/openshift/api/operator/v1"
	operatorclient "github.com/openshift/client-go/operator/clientset/versioned"
	fakeopclient "github.com/openshift/client-go/operator/clientset/versioned/fake"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/test/util/cmp"
)

func TestIngressProfilesEnricherTask(t *testing.T) {
	log := logrus.NewEntry(logrus.StandardLogger())

	oioNamespace := "openshift-ingress-operator"
	oiNamespace := "openshift-ingress"
	owningIngressLabel := "ingresscontroller.operator.openshift.io/owning-ingresscontroller"
	for _, tt := range []struct {
		name        string
		operatorcli operatorclient.Interface
		kubecli     kubernetes.Interface
		wantOc      *api.OpenShiftCluster
		wantErr     string
	}{
		{
			name: "default simplest case of ingress profile found",
			operatorcli: fakeopclient.NewSimpleClientset(
				&operatorv1.IngressController{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "default",
						Namespace: oioNamespace,
					},
				},
			),
			kubecli: fake.NewSimpleClientset(&corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "router-default",
					Namespace: oiNamespace,
					Labels: map[string]string{
						"app":              "router",
						owningIngressLabel: "default",
					},
				},
				Status: corev1.ServiceStatus{
					LoadBalancer: corev1.LoadBalancerStatus{
						Ingress: []corev1.LoadBalancerIngress{
							{
								IP: "x.x.x.x",
							},
						},
					},
				},
			}),
			wantOc: &api.OpenShiftCluster{
				Properties: api.OpenShiftClusterProperties{
					IngressProfiles: []api.IngressProfile{
						{
							Name:       "default",
							Visibility: api.VisibilityPublic,
							IP:         "x.x.x.x",
						},
					},
				},
			},
		},
		{
			name: "private ingress profile found",
			operatorcli: fakeopclient.NewSimpleClientset(
				&operatorv1.IngressController{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "private",
						Namespace: oioNamespace,
					},
					Spec: operatorv1.IngressControllerSpec{
						EndpointPublishingStrategy: &operatorv1.EndpointPublishingStrategy{
							LoadBalancer: &operatorv1.LoadBalancerStrategy{
								Scope: operatorv1.InternalLoadBalancer,
							},
						},
					},
				},
			),
			kubecli: fake.NewSimpleClientset(&corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "router-private",
					Namespace: oiNamespace,
					Labels: map[string]string{
						"app":              "router",
						owningIngressLabel: "private",
					},
				},
				Status: corev1.ServiceStatus{
					LoadBalancer: corev1.LoadBalancerStatus{
						Ingress: []corev1.LoadBalancerIngress{
							{
								IP: "x.x.x.x",
							},
						},
					},
				},
			}),
			wantOc: &api.OpenShiftCluster{
				Properties: api.OpenShiftClusterProperties{
					IngressProfiles: []api.IngressProfile{
						{
							Name:       "private",
							Visibility: api.VisibilityPrivate,
							IP:         "x.x.x.x",
						},
					},
				},
			},
		},
		{
			name: "public ingress profile found",
			operatorcli: fakeopclient.NewSimpleClientset(
				&operatorv1.IngressController{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "public",
						Namespace: oioNamespace,
					},
					Spec: operatorv1.IngressControllerSpec{
						EndpointPublishingStrategy: &operatorv1.EndpointPublishingStrategy{
							LoadBalancer: &operatorv1.LoadBalancerStrategy{
								Scope: operatorv1.ExternalLoadBalancer,
							},
						},
					},
				},
			),
			kubecli: fake.NewSimpleClientset(&corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "router-public",
					Namespace: oiNamespace,
					Labels: map[string]string{
						"app":              "router",
						owningIngressLabel: "public",
					},
				},
				Status: corev1.ServiceStatus{
					LoadBalancer: corev1.LoadBalancerStatus{
						Ingress: []corev1.LoadBalancerIngress{
							{
								IP: "x.x.x.x",
							},
						},
					},
				},
			}),
			wantOc: &api.OpenShiftCluster{
				Properties: api.OpenShiftClusterProperties{
					IngressProfiles: []api.IngressProfile{
						{
							Name:       "public",
							Visibility: api.VisibilityPublic,
							IP:         "x.x.x.x",
						},
					},
				},
			},
		},
		{
			name: "several ingress profiles found",
			operatorcli: fakeopclient.NewSimpleClientset(
				&operatorv1.IngressController{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "default",
						Namespace: oioNamespace,
					},
					Spec: operatorv1.IngressControllerSpec{
						EndpointPublishingStrategy: &operatorv1.EndpointPublishingStrategy{
							LoadBalancer: &operatorv1.LoadBalancerStrategy{
								Scope: operatorv1.ExternalLoadBalancer,
							},
						},
					},
				},
				&operatorv1.IngressController{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "extra",
						Namespace: oioNamespace,
					},
					Spec: operatorv1.IngressControllerSpec{
						EndpointPublishingStrategy: &operatorv1.EndpointPublishingStrategy{
							LoadBalancer: &operatorv1.LoadBalancerStrategy{
								Scope: operatorv1.InternalLoadBalancer,
							},
						},
					},
				},
			),
			kubecli: fake.NewSimpleClientset(
				&corev1.Service{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "router-default",
						Namespace: oiNamespace,
						Labels: map[string]string{
							"app":              "router",
							owningIngressLabel: "default",
						},
					},
					Status: corev1.ServiceStatus{
						LoadBalancer: corev1.LoadBalancerStatus{
							Ingress: []corev1.LoadBalancerIngress{
								{
									IP: "x.x.x.x",
								},
							},
						},
					},
				},
				&corev1.Service{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "router-extra",
						Namespace: oiNamespace,
						Labels: map[string]string{
							"app":              "router",
							owningIngressLabel: "extra",
						},
					},
					Status: corev1.ServiceStatus{
						LoadBalancer: corev1.LoadBalancerStatus{
							Ingress: []corev1.LoadBalancerIngress{
								{
									IP: "y.y.y.y",
								},
							},
						},
					},
				},
			),
			wantOc: &api.OpenShiftCluster{
				Properties: api.OpenShiftClusterProperties{
					IngressProfiles: []api.IngressProfile{
						{
							Name:       "default",
							Visibility: api.VisibilityPublic,
							IP:         "x.x.x.x",
						},
						{
							Name:       "extra",
							Visibility: api.VisibilityPrivate,
							IP:         "y.y.y.y",
						},
					},
				},
			},
		},
		{
			name: "no router service found",
			operatorcli: fakeopclient.NewSimpleClientset(
				&operatorv1.IngressController{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "private",
						Namespace: oioNamespace,
					},
					Spec: operatorv1.IngressControllerSpec{
						EndpointPublishingStrategy: &operatorv1.EndpointPublishingStrategy{
							LoadBalancer: &operatorv1.LoadBalancerStrategy{
								Scope: operatorv1.InternalLoadBalancer,
							},
						},
					},
				},
			),
			kubecli: fake.NewSimpleClientset(&corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "useless",
					Namespace: oiNamespace,
				},
				Status: corev1.ServiceStatus{
					LoadBalancer: corev1.LoadBalancerStatus{
						Ingress: []corev1.LoadBalancerIngress{
							{
								IP: "x.x.x.x",
							},
						},
					},
				},
			}),
			wantOc: &api.OpenShiftCluster{
				Properties: api.OpenShiftClusterProperties{
					IngressProfiles: []api.IngressProfile{
						{
							Name:       "private",
							Visibility: api.VisibilityPrivate,
						},
					},
				},
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			oc := &api.OpenShiftCluster{}
			e := &ingressProfileEnricherTask{
				log:         log,
				operatorcli: tt.operatorcli,
				kubecli:     tt.kubecli,
				oc:          oc,
			}
			e.SetDefaults()

			callbacks := make(chan func())
			errors := make(chan error)
			go e.FetchData(context.Background(), callbacks, errors)

			select {
			case f := <-callbacks:
				f()
				if !reflect.DeepEqual(oc, tt.wantOc) {
					t.Error(cmp.Diff(oc, tt.wantOc))
				}
			case err := <-errors:
				if tt.wantErr != err.Error() {
					t.Error(err)
				}
			}
		})
	}
}
