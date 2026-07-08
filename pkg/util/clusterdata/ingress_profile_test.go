package clusterdata

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"reflect"
	"testing"

	"github.com/sirupsen/logrus"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"

	operatorv1 "github.com/openshift/api/operator/v1"
	operatorclient "github.com/openshift/client-go/operator/clientset/versioned"
	operatorfake "github.com/openshift/client-go/operator/clientset/versioned/fake"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/util/cmp"
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
			name: "visibility remains unknown when endpoint publishing strategy is missing",
			operatorcli: operatorfake.NewSimpleClientset(
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
							Name: "default",
							IP:   "x.x.x.x",
						},
					},
				},
			},
		},
		{
			name: "private ingress profile found",
			operatorcli: operatorfake.NewSimpleClientset(
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
			operatorcli: operatorfake.NewSimpleClientset(
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
			operatorcli: operatorfake.NewSimpleClientset(
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
			operatorcli: operatorfake.NewSimpleClientset(
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
			e := ingressProfileEnricher{}

			clients := clients{
				k8s:      tt.kubecli,
				operator: tt.operatorcli,
			}
			err := e.Enrich(context.Background(), log, oc, clients.k8s, clients.config, clients.machine, clients.operator)
			if (err == nil && tt.wantErr != "") || (err != nil && err.Error() != tt.wantErr) {
				t.Errorf("wanted err to be %s but got %s", err, tt.wantErr)
			}
			if !reflect.DeepEqual(oc, tt.wantOc) {
				t.Error(cmp.Diff(oc, tt.wantOc))
			}
		})
	}
}

func TestVisibilityFromEndpointPublishingStrategy(t *testing.T) {
	for _, tt := range []struct {
		name           string
		strategy       *operatorv1.EndpointPublishingStrategy
		wantVisibility api.Visibility
		wantOK         bool
	}{
		{
			name:           "internal load balancer resolves to private",
			strategy:       endpointPublishingStrategy(operatorv1.InternalLoadBalancer),
			wantVisibility: api.VisibilityPrivate,
			wantOK:         true,
		},
		{
			name:           "external load balancer resolves to public",
			strategy:       endpointPublishingStrategy(operatorv1.ExternalLoadBalancer),
			wantVisibility: api.VisibilityPublic,
			wantOK:         true,
		},
		{
			name:     "nil strategy remains unknown",
			strategy: nil,
		},
		{
			name:     "missing load balancer remains unknown",
			strategy: &operatorv1.EndpointPublishingStrategy{},
		},
		{
			name:     "unexpected scope remains unknown",
			strategy: endpointPublishingStrategy(operatorv1.LoadBalancerScope("Unexpected")),
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			gotVisibility, gotOK := visibilityFromEndpointPublishingStrategy(tt.strategy)
			if gotVisibility != tt.wantVisibility {
				t.Fatalf("got visibility %q, want %q", gotVisibility, tt.wantVisibility)
			}
			if gotOK != tt.wantOK {
				t.Fatalf("got ok %t, want %t", gotOK, tt.wantOK)
			}
		})
	}
}

func TestIngressProfileVisibility(t *testing.T) {
	log := logrus.NewEntry(logrus.StandardLogger())

	for _, tt := range []struct {
		name              string
		ingressController operatorv1.IngressController
		wantVisibility    api.Visibility
	}{
		{
			name: "status internal load balancer resolves to private when spec is nil",
			ingressController: operatorv1.IngressController{
				Status: operatorv1.IngressControllerStatus{
					EndpointPublishingStrategy: endpointPublishingStrategy(operatorv1.InternalLoadBalancer),
				},
			},
			wantVisibility: api.VisibilityPrivate,
		},
		{
			name: "status external load balancer resolves to public when spec is nil",
			ingressController: operatorv1.IngressController{
				Status: operatorv1.IngressControllerStatus{
					EndpointPublishingStrategy: endpointPublishingStrategy(operatorv1.ExternalLoadBalancer),
				},
			},
			wantVisibility: api.VisibilityPublic,
		},
		{
			name: "spec internal load balancer resolves to private when status is nil",
			ingressController: operatorv1.IngressController{
				Spec: operatorv1.IngressControllerSpec{
					EndpointPublishingStrategy: endpointPublishingStrategy(operatorv1.InternalLoadBalancer),
				},
			},
			wantVisibility: api.VisibilityPrivate,
		},
		{
			name: "spec external load balancer resolves to public when status is nil",
			ingressController: operatorv1.IngressController{
				Spec: operatorv1.IngressControllerSpec{
					EndpointPublishingStrategy: endpointPublishingStrategy(operatorv1.ExternalLoadBalancer),
				},
			},
			wantVisibility: api.VisibilityPublic,
		},
		{
			name:              "visibility remains unknown when status and spec are nil",
			ingressController: operatorv1.IngressController{},
		},
		{
			name: "status takes precedence over spec when both resolve",
			ingressController: operatorv1.IngressController{
				Spec: operatorv1.IngressControllerSpec{
					EndpointPublishingStrategy: endpointPublishingStrategy(operatorv1.ExternalLoadBalancer),
				},
				Status: operatorv1.IngressControllerStatus{
					EndpointPublishingStrategy: endpointPublishingStrategy(operatorv1.InternalLoadBalancer),
				},
			},
			wantVisibility: api.VisibilityPrivate,
		},
		{
			name: "status without load balancer falls back to spec",
			ingressController: operatorv1.IngressController{
				Spec: operatorv1.IngressControllerSpec{
					EndpointPublishingStrategy: endpointPublishingStrategy(operatorv1.ExternalLoadBalancer),
				},
				Status: operatorv1.IngressControllerStatus{
					EndpointPublishingStrategy: &operatorv1.EndpointPublishingStrategy{},
				},
			},
			wantVisibility: api.VisibilityPublic,
		},
		{
			name: "visibility remains unknown when both status and spec are unusable",
			ingressController: operatorv1.IngressController{
				Spec: operatorv1.IngressControllerSpec{
					EndpointPublishingStrategy: &operatorv1.EndpointPublishingStrategy{},
				},
				Status: operatorv1.IngressControllerStatus{
					EndpointPublishingStrategy: endpointPublishingStrategy(operatorv1.LoadBalancerScope("Unexpected")),
				},
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			gotVisibility := ingressProfileVisibility(log, "test", tt.ingressController)
			if gotVisibility != tt.wantVisibility {
				t.Fatalf("got visibility %q, want %q", gotVisibility, tt.wantVisibility)
			}
		})
	}
}

func endpointPublishingStrategy(scope operatorv1.LoadBalancerScope) *operatorv1.EndpointPublishingStrategy {
	return &operatorv1.EndpointPublishingStrategy{
		LoadBalancer: &operatorv1.LoadBalancerStrategy{
			Scope: scope,
		},
	}
}
