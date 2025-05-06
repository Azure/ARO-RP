package ingresscertificatechecker

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"reflect"
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	operatorv1 "github.com/openshift/api/operator/v1"

	"github.com/Azure/ARO-RP/pkg/operator"
	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	"github.com/Azure/ARO-RP/pkg/util/azureclient"
	"github.com/Azure/ARO-RP/pkg/util/cmp"
	utillog "github.com/Azure/ARO-RP/pkg/util/log"
	_ "github.com/Azure/ARO-RP/pkg/util/scheme"
	utilerror "github.com/Azure/ARO-RP/test/util/error"
)

type fakeChecker struct {
	expectedResult error
}

func (fc fakeChecker) Check(ctx context.Context) error {
	return fc.expectedResult
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
			wantConditionMessage: "Default ingress certificate is in use",
			wantResult:           reconcile.Result{RequeueAfter: time.Hour},
		},
		{
			name:                 "reconciler handles correctly any error returned from the reconciler.checker",
			wantConditionStatus:  operatorv1.ConditionFalse,
			wantConditionMessage: errNoCertificateAndCustomDomain.Error(),
			checkerReturnErr:     errNoCertificateAndCustomDomain,
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
						operator.CheckerEnabled: operator.FlagTrue,
					},
				},
			}
			if tt.controllerDisabled {
				instance.Spec.OperatorFlags[operator.CheckerEnabled] = operator.FlagFalse
			}

			clientFake := fake.NewClientBuilder().WithObjects(instance).WithStatusSubresource(instance).Build()

			r := &Reconciler{
				log:     utillog.GetLogger(),
				role:    "master",
				checker: fakeChecker{expectedResult: tt.checkerReturnErr},
				client:  clientFake,
			}

			result, err := r.Reconcile(ctx, ctrl.Request{})
			utilerror.AssertErrorMessage(t, err, tt.wantErr)

			if !reflect.DeepEqual(tt.wantResult, result) {
				t.Error(cmp.Diff(tt.wantResult, result))
			}

			err = r.client.Get(ctx, types.NamespacedName{Name: arov1alpha1.SingletonClusterName}, instance)
			if err != nil {
				t.Fatal(err)
			}

			var condition *operatorv1.OperatorCondition
			for i := range instance.Status.Conditions {
				if instance.Status.Conditions[i].Type == arov1alpha1.DefaultIngressCertificate {
					condition = &instance.Status.Conditions[i]
				}
			}
			if condition == nil {
				t.Fatal("no condition found")
			} else if condition.Status != tt.wantConditionStatus {
				t.Error(string(condition.Status))
			}

			if condition.Message != tt.wantConditionMessage {
				t.Error(condition.Message)
			}
		})
	}
}
