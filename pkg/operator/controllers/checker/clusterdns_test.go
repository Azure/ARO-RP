package checker

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"testing"
	"time"

	"github.com/go-test/deep"
	operatorv1 "github.com/openshift/api/operator/v1"
	operatorfake "github.com/openshift/client-go/operator/clientset/versioned/fake"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	arofake "github.com/Azure/ARO-RP/pkg/operator/clientset/versioned/fake"
)

func TestDefaultClusterDNS(t *testing.T) {
	ctx := context.Background()

	for _, tt := range []struct {
		name          string
		aroCluster    *arov1alpha1.Cluster
		dns           *operatorv1.DNS
		expectedState operatorv1.OperatorCondition
	}{
		{
			name: "run: has default DNS",
			aroCluster: &arov1alpha1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: arov1alpha1.SingletonClusterName,
				},
			},
			dns: &operatorv1.DNS{
				ObjectMeta: metav1.ObjectMeta{
					Name: "default",
				},
				Spec: operatorv1.DNSSpec{},
			},
			expectedState: operatorv1.OperatorCondition{
				Type:    arov1alpha1.DefaultClusterDNS,
				Status:  operatorv1.ConditionTrue,
				Reason:  "CheckDone",
				Message: "No in-cluster upstream DNS servers",
			},
		},
		{
			name: "run: has changed DNS",
			aroCluster: &arov1alpha1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: arov1alpha1.SingletonClusterName,
				},
			},
			dns: &operatorv1.DNS{
				ObjectMeta: metav1.ObjectMeta{
					Name: "default",
				},
				Spec: operatorv1.DNSSpec{
					Servers: []operatorv1.Server{
						{
							Name:  "test-server",
							Zones: []string{"example.com"},
							ForwardPlugin: operatorv1.ForwardPlugin{
								Upstreams: []string{
									"1.2.3.4", "5.6.7.8",
								},
							},
						},
					},
				},
			},
			expectedState: operatorv1.OperatorCondition{
				Type:    arov1alpha1.DefaultClusterDNS,
				Status:  operatorv1.ConditionFalse,
				Reason:  "CheckDone",
				Message: "Custom upstream DNS servers in use: 1.2.3.4, 5.6.7.8",
			},
		},
		{
			name: "run: malformed, servers but no forwardplugin",
			aroCluster: &arov1alpha1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: arov1alpha1.SingletonClusterName,
				},
			},
			dns: &operatorv1.DNS{
				ObjectMeta: metav1.ObjectMeta{
					Name: "default",
				},
				Spec: operatorv1.DNSSpec{
					Servers: []operatorv1.Server{
						{
							Name:  "test-server",
							Zones: []string{"example.com"},
						},
					},
				},
			},
			expectedState: operatorv1.OperatorCondition{
				Type:    arov1alpha1.DefaultClusterDNS,
				Status:  operatorv1.ConditionTrue,
				Reason:  "CheckDone",
				Message: "No in-cluster upstream DNS servers",
			},
		},
		{
			name: "fail: malformed, using . for zones",
			aroCluster: &arov1alpha1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: arov1alpha1.SingletonClusterName,
				},
			},
			dns: &operatorv1.DNS{
				ObjectMeta: metav1.ObjectMeta{
					Name: "default",
				},
				Spec: operatorv1.DNSSpec{
					Servers: []operatorv1.Server{
						{
							Name:  "test-server",
							Zones: []string{"."},
							ForwardPlugin: operatorv1.ForwardPlugin{
								Upstreams: []string{
									"1.2.3.4", "5.6.7.8",
								},
							},
						},
					},
				},
			},
			expectedState: operatorv1.OperatorCondition{
				Type:    arov1alpha1.DefaultClusterDNS,
				Status:  operatorv1.ConditionFalse,
				Reason:  "CheckDone",
				Message: `Malformed config: "." in zones`,
			},
		},
		{
			name: "fail: no DNS",
			aroCluster: &arov1alpha1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: arov1alpha1.SingletonClusterName,
				},
			},
			expectedState: operatorv1.OperatorCondition{
				Type:    arov1alpha1.DefaultClusterDNS,
				Status:  operatorv1.ConditionUnknown,
				Reason:  "CheckFailed",
				Message: `dnses.operator.openshift.io "default" not found`,
			},
		},
	} {
		arocli := arofake.NewSimpleClientset()
		operatorcli := operatorfake.NewSimpleClientset()

		if tt.aroCluster != nil {
			arocli = arofake.NewSimpleClientset(tt.aroCluster)
		}
		if tt.dns != nil {
			operatorcli = operatorfake.NewSimpleClientset(tt.dns)
		}

		sp := NewClusterDNSChecker(nil, arocli, operatorcli, "")

		t.Run(tt.name, func(t *testing.T) {
			err := sp.Check(ctx)

			if err != nil {
				t.Error(err)
			}

			cluster, err := arocli.AroV1alpha1().Clusters().Get(ctx, arov1alpha1.SingletonClusterName, metav1.GetOptions{})
			if err != nil {
				t.Error(err)
			}

			conds := []operatorv1.OperatorCondition{}

			// nil out time
			for _, c := range cluster.Status.Conditions {
				c.LastTransitionTime = metav1.NewTime(time.Time{})
				conds = append(conds, c)
			}

			errs := deep.Equal(conds, []operatorv1.OperatorCondition{tt.expectedState})
			for _, err := range errs {
				t.Error(err)
			}
		})
	}
}
