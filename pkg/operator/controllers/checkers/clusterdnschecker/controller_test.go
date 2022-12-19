package clusterdnschecker

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	operatorv1 "github.com/openshift/api/operator/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	arofake "github.com/Azure/ARO-RP/pkg/operator/clientset/versioned/fake"
	checkercommon "github.com/Azure/ARO-RP/pkg/operator/controllers/checkers/common"
	"github.com/Azure/ARO-RP/pkg/util/azureclient"
	"github.com/Azure/ARO-RP/pkg/util/cmp"
	utillog "github.com/Azure/ARO-RP/pkg/util/log"
	_ "github.com/Azure/ARO-RP/pkg/util/scheme"
)

type fakeChecker func(ctx context.Context) error

func (fc fakeChecker) Check(ctx context.Context) error {
	return fc(ctx)
}

func TestReconcile(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name                 string
		controllerDisabled   bool
		checkerReturnErr     error
		wantConditionStatus  operatorv1.ConditionStatus
		wantConditionMessage string
		wantErr              string
		wantResult           reconcile.Result
	}{
		{
			name:                 "no errors",
			wantConditionStatus:  operatorv1.ConditionTrue,
			wantConditionMessage: "No in-cluster upstream DNS servers",
			wantResult:           reconcile.Result{RequeueAfter: time.Hour},
		},
		{
			name:                 "check failed with an error",
			wantConditionStatus:  operatorv1.ConditionFalse,
			wantConditionMessage: "fake basic error",
			checkerReturnErr:     errors.New("fake basic error"),
			wantErr:              "fake basic error",
			wantResult:           reconcile.Result{RequeueAfter: time.Hour},
		},
		{
			name:                "controller disabled",
			controllerDisabled:  true,
			wantConditionStatus: operatorv1.ConditionUnknown,
			wantResult:          reconcile.Result{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			instance := &arov1alpha1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: arov1alpha1.SingletonClusterName,
				},
				Spec: arov1alpha1.ClusterSpec{
					AZEnvironment: azureclient.PublicCloud.Environment.Name,
					OperatorFlags: arov1alpha1.OperatorFlags{
						checkercommon.ControllerEnabled: "true",
					},
				},
			}
			if tt.controllerDisabled {
				instance.Spec.OperatorFlags[checkercommon.ControllerEnabled] = "false"
			}

			clientFake := fake.NewClientBuilder().WithObjects(instance).Build()
			arocliFake := arofake.NewSimpleClientset(instance)

			r := &Reconciler{
				log:  utillog.GetLogger(),
				role: "master",
				checker: fakeChecker(func(ctx context.Context) error {
					return tt.checkerReturnErr
				}),
				arocli: arocliFake,
				client: clientFake,
			}

			result, err := r.Reconcile(ctx, ctrl.Request{})
			if err != nil && err.Error() != tt.wantErr ||
				err == nil && tt.wantErr != "" {
				t.Error(err)
			}

			if !reflect.DeepEqual(tt.wantResult, result) {
				t.Error(cmp.Diff(tt.wantResult, result))
			}

			instance, err = arocliFake.AroV1alpha1().Clusters().Get(ctx, arov1alpha1.SingletonClusterName, metav1.GetOptions{})
			if err != nil {
				t.Fatal(err)
			}

			var condition *operatorv1.OperatorCondition
			for i := range instance.Status.Conditions {
				if instance.Status.Conditions[i].Type == arov1alpha1.DefaultClusterDNS {
					condition = &instance.Status.Conditions[i]
				}
			}
			if condition == nil {
				t.Fatal("no condition found")
			}

			if condition.Status != tt.wantConditionStatus {
				t.Errorf(string(condition.Status))
			}

			if condition.Message != tt.wantConditionMessage {
				t.Errorf(condition.Message)
			}
		})
	}
}
