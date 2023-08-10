package machinehealthcheck

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/go-test/deep"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrlfake "sigs.k8s.io/controller-runtime/pkg/client/fake"

	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	"github.com/Azure/ARO-RP/pkg/util/dynamichelper"
	_ "github.com/Azure/ARO-RP/pkg/util/scheme"
	testdh "github.com/Azure/ARO-RP/test/util/dynamichelper"
	utilerror "github.com/Azure/ARO-RP/test/util/error"
	testlog "github.com/Azure/ARO-RP/test/util/log"
)

// Test reconcile function
func TestReconciler(t *testing.T) {
	type test struct {
		name             string
		instance         *arov1alpha1.Cluster
		causeFailureOn   []string
		wantErr          string
		wantRequeueAfter time.Duration
		wantCreated      map[string]int
		wantDeleted      map[string]int
	}

	for _, tt := range []*test{
		{
			name:        "Failure to get instance",
			wantErr:     `clusters.aro.openshift.io "cluster" not found`,
			wantCreated: map[string]int{},
			wantDeleted: map[string]int{},
		},
		{
			name: "Enabled Feature Flag is false",
			instance: &arov1alpha1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: arov1alpha1.SingletonClusterName,
				},
				Spec: arov1alpha1.ClusterSpec{
					OperatorFlags: arov1alpha1.OperatorFlags{
						enabled: strconv.FormatBool(false),
					},
				},
			},
			wantErr:     "",
			wantCreated: map[string]int{},
			wantDeleted: map[string]int{},
		},
		{
			name: "Managed Feature Flag is false: ensure mhc and its alert are deleted",
			instance: &arov1alpha1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: arov1alpha1.SingletonClusterName,
				},
				Spec: arov1alpha1.ClusterSpec{
					OperatorFlags: arov1alpha1.OperatorFlags{
						enabled: strconv.FormatBool(true),
						managed: strconv.FormatBool(false),
					},
				},
			},
			wantErr:     "",
			wantCreated: map[string]int{},
			wantDeleted: map[string]int{
				"MachineHealthCheck/openshift-machine-api/aro-machinehealthcheck": 1,
				"PrometheusRule/openshift-machine-api/mhc-remediation-alert":      1,
			},
		},
		{
			name: "Managed Feature Flag is false: mhc fails to delete, an error is returned",
			instance: &arov1alpha1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: arov1alpha1.SingletonClusterName,
				},
				Spec: arov1alpha1.ClusterSpec{
					OperatorFlags: arov1alpha1.OperatorFlags{
						enabled: strconv.FormatBool(true),
						managed: strconv.FormatBool(false),
					},
				},
			},
			causeFailureOn: []string{
				"MachineHealthCheck/openshift-machine-api/aro-machinehealthcheck",
			},
			wantErr:          "triggered failure deleting MachineHealthCheck/openshift-machine-api/aro-machinehealthcheck",
			wantRequeueAfter: time.Hour,
			wantCreated:      map[string]int{},
			wantDeleted: map[string]int{
				"PrometheusRule/openshift-machine-api/mhc-remediation-alert": 1,
			},
		},
		{
			name: "Managed Feature Flag is false: mhc deletes but mhc alert fails to delete, an error is returned",
			instance: &arov1alpha1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: arov1alpha1.SingletonClusterName,
				},
				Spec: arov1alpha1.ClusterSpec{
					OperatorFlags: arov1alpha1.OperatorFlags{
						enabled: strconv.FormatBool(true),
						managed: strconv.FormatBool(false),
					},
				},
			},
			causeFailureOn: []string{
				"PrometheusRule/openshift-machine-api/mhc-remediation-alert",
			},
			wantErr:          "triggered failure deleting PrometheusRule/openshift-machine-api/mhc-remediation-alert",
			wantRequeueAfter: time.Hour,
			wantCreated:      map[string]int{},
			wantDeleted: map[string]int{
				"MachineHealthCheck/openshift-machine-api/aro-machinehealthcheck": 1,
			},
		},
		{
			name: "Managed Feature Flag is true: dynamic helper ensures resources",
			instance: &arov1alpha1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: arov1alpha1.SingletonClusterName,
				},
				Spec: arov1alpha1.ClusterSpec{
					OperatorFlags: arov1alpha1.OperatorFlags{
						enabled: strconv.FormatBool(true),
						managed: strconv.FormatBool(true),
					},
				},
			},
			wantErr: "",
			wantCreated: map[string]int{
				"MachineHealthCheck/openshift-machine-api/aro-machinehealthcheck": 1,
				"PrometheusRule/openshift-machine-api/mhc-remediation-alert":      1,
			},
			wantDeleted: map[string]int{},
		},
		{
			name: "When ensuring resources fails, an error is returned",
			instance: &arov1alpha1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: arov1alpha1.SingletonClusterName,
				},
				Spec: arov1alpha1.ClusterSpec{
					OperatorFlags: arov1alpha1.OperatorFlags{
						enabled: strconv.FormatBool(true),
						managed: strconv.FormatBool(true),
					},
				},
			},
			causeFailureOn: []string{
				"MachineHealthCheck/openshift-machine-api/aro-machinehealthcheck",
			},
			wantErr:     "triggered failure creating MachineHealthCheck/openshift-machine-api/aro-machinehealthcheck",
			wantCreated: map[string]int{},
			wantDeleted: map[string]int{},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			_, log := testlog.New()
			clientBuilder := ctrlfake.NewClientBuilder()
			if tt.instance != nil {
				clientBuilder = clientBuilder.WithObjects(tt.instance)
			}

			deployedObjects := map[string]int{}
			deletedObjects := map[string]int{}
			wrappedClient := testdh.NewRedirectingClient(clientBuilder.Build()).
				WithDeleteHook(func(obj client.Object) error {
					for _, v := range tt.causeFailureOn {
						if obj.GetObjectKind().GroupVersionKind().Kind+"/"+obj.GetNamespace()+"/"+obj.GetName() == v {
							return fmt.Errorf("triggered failure deleting %s", v)
						}
					}
					testdh.TallyCountsAndKey(deletedObjects)(obj)
					return nil
				}).
				WithCreateHook(func(obj client.Object) error {
					for _, v := range tt.causeFailureOn {
						if obj.GetObjectKind().GroupVersionKind().Kind+"/"+obj.GetNamespace()+"/"+obj.GetName() == v {
							return fmt.Errorf("triggered failure creating %s", v)
						}
					}
					testdh.TallyCountsAndKey(deployedObjects)(obj)
					return nil
				})

			dh := dynamichelper.NewWithClient(log, wrappedClient)

			ctx := context.Background()
			r := &Reconciler{
				log:    logrus.NewEntry(logrus.StandardLogger()),
				dh:     dh,
				client: clientBuilder.Build(),
			}
			request := ctrl.Request{}
			request.Name = "cluster"

			result, err := r.Reconcile(ctx, request)

			if tt.wantRequeueAfter != result.RequeueAfter {
				t.Errorf("Wanted to requeue after %v but was set to %v", tt.wantRequeueAfter, result.RequeueAfter)
			}

			for _, v := range deep.Equal(deployedObjects, tt.wantCreated) {
				t.Errorf("created does not match: %s", v)
			}
			for _, v := range deep.Equal(deletedObjects, tt.wantDeleted) {
				t.Errorf("deleted does not match: %s", v)
			}

			utilerror.AssertErrorMessage(t, err, tt.wantErr)
		})
	}
}
