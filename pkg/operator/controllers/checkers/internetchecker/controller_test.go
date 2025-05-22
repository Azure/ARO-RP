package internetchecker

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
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
	"github.com/Azure/ARO-RP/pkg/util/cmp"
	utillog "github.com/Azure/ARO-RP/pkg/util/log"
	_ "github.com/Azure/ARO-RP/pkg/util/scheme"
	utilerror "github.com/Azure/ARO-RP/test/util/error"
)

type fakeChecker func(URLs []string) error

func (fc fakeChecker) Check(URLs []string) error {
	return fc(URLs)
}

func TestReconcile(t *testing.T) {
	ctx := context.Background()
	urlsToCheck := []string{"https://fake-url-for-test-only.xyz"}

	tests := []struct {
		name               string
		controllerDisabled bool
		checkerReturnErr   error
		wantCondition      operatorv1.ConditionStatus
		wantErr            string
		wantResult         reconcile.Result
	}{
		{
			name:          "no errors",
			wantCondition: operatorv1.ConditionTrue,
			wantResult:    reconcile.Result{RequeueAfter: time.Hour},
		},
		{
			name:             "error making a request",
			checkerReturnErr: errors.New("fake error from checker"),
			wantCondition:    operatorv1.ConditionFalse,
			wantErr:          "fake error from checker",
			wantResult:       reconcile.Result{RequeueAfter: time.Hour},
		},
		{
			name:               "controller disabled",
			controllerDisabled: true,
			wantCondition:      operatorv1.ConditionUnknown,
			wantResult:         reconcile.Result{},
		},
	}

	roleToConditionTypeMap := map[string]string{
		"master":         arov1alpha1.InternetReachableFromMaster,
		"worker":         arov1alpha1.InternetReachableFromWorker,
		"incorrect-role": arov1alpha1.InternetReachableFromWorker,
	}
	for _, testRole := range []string{operator.RoleMaster, operator.RoleWorker, "incorrect-role"} {
		t.Run(testRole, func(t *testing.T) {
			for _, tt := range tests {
				t.Run(tt.name, func(t *testing.T) {
					instance := &arov1alpha1.Cluster{
						ObjectMeta: metav1.ObjectMeta{
							Name: arov1alpha1.SingletonClusterName,
						},
						Spec: arov1alpha1.ClusterSpec{
							InternetChecker: arov1alpha1.InternetCheckerSpec{
								URLs: urlsToCheck,
							},
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
						log:  utillog.GetLogger(),
						role: testRole,
						checker: fakeChecker(func(URLs []string) error {
							if !reflect.DeepEqual(urlsToCheck, URLs) {
								t.Error(cmp.Diff(urlsToCheck, URLs))
							}

							return tt.checkerReturnErr
						}),
						client: clientFake,
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
						if instance.Status.Conditions[i].Type == roleToConditionTypeMap[testRole] {
							condition = &instance.Status.Conditions[i]
						}
					}
					if condition == nil {
						t.Fatal("no condition found")
					} else if condition.Status != tt.wantCondition {
						t.Error(condition.Status)
					}
				})
			}
		})
	}
}
