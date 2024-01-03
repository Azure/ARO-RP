package genevalogging

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/go-test/deep"
	"github.com/golang/mock/gomock"
	operatorv1 "github.com/openshift/api/operator/v1"
	securityv1 "github.com/openshift/api/security/v1"
	"github.com/sirupsen/logrus"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrlfake "sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/Azure/ARO-RP/pkg/operator"
	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	mock_dynamichelper "github.com/Azure/ARO-RP/pkg/util/mocks/dynamichelper"
	_ "github.com/Azure/ARO-RP/pkg/util/scheme"
	"github.com/Azure/ARO-RP/pkg/util/version"
	testdatabase "github.com/Azure/ARO-RP/test/database"
	utilconditions "github.com/Azure/ARO-RP/test/util/conditions"
	utilerror "github.com/Azure/ARO-RP/test/util/error"
)

func getContainer(d *appsv1.DaemonSet, containerName string) (corev1.Container, error) {
	for _, container := range d.Spec.Template.Spec.Containers {
		if container.Name == containerName {
			return container, nil
		}
	}
	return corev1.Container{}, errors.New("not found")
}

func TestGenevaLoggingDaemonset(t *testing.T) {
	nominalMocks := func(mockDh *mock_dynamichelper.MockInterface) {
		mockDh.EXPECT().Ensure(
			gomock.Any(),
			gomock.AssignableToTypeOf(&securityv1.SecurityContextConstraints{}),
			gomock.AssignableToTypeOf(&corev1.Namespace{}),
			gomock.AssignableToTypeOf(&corev1.ConfigMap{}),
			gomock.AssignableToTypeOf(&corev1.Secret{}),
			gomock.AssignableToTypeOf(&corev1.ServiceAccount{}),
			gomock.AssignableToTypeOf(&appsv1.DaemonSet{}),
		).Times(1)
	}

	defaultConditions := []operatorv1.OperatorCondition{
		utilconditions.ControllerDefaultAvailable(controllerName),
		utilconditions.ControllerDefaultProgressing(controllerName),
		utilconditions.ControllerDefaultDegraded(controllerName),
	}

	tests := []struct {
		name              string
		request           ctrl.Request
		operatorFlags     arov1alpha1.OperatorFlags
		validateDaemonset func(*appsv1.DaemonSet) []error
		mocks             func(mockDh *mock_dynamichelper.MockInterface)
		wantErrMsg        string
		wantConditions    []operatorv1.OperatorCondition
	}{
		{
			name: "no flags given",
			operatorFlags: arov1alpha1.OperatorFlags{
				controllerEnabled: "true",
			},
			validateDaemonset: func(d *appsv1.DaemonSet) (errs []error) {
				if len(d.Spec.Template.Spec.Containers) != 2 {
					errs = append(errs, fmt.Errorf("expected 2 containers, got %d", len(d.Spec.Template.Spec.Containers)))
				}

				// we want the default fluentbit image
				fluentbit, err := getContainer(d, "fluentbit")
				if err != nil {
					errs = append(errs, err)
					return
				}
				for _, err := range deep.Equal(fluentbit.Image, version.FluentbitImage("acrDomain")) {
					errs = append(errs, errors.New(err))
				}

				// we want the default mdsd image
				mdsd, err := getContainer(d, "mdsd")
				if err != nil {
					errs = append(errs, err)
					return
				}
				for _, err := range deep.Equal(mdsd.Image, version.MdsdImage("acrDomain")) {
					errs = append(errs, errors.New(err))
				}

				return
			},
			mocks:          nominalMocks,
			wantErrMsg:     "",
			wantConditions: defaultConditions,
		},
		{
			name: "fluentbit changed",
			operatorFlags: arov1alpha1.OperatorFlags{
				controllerEnabled:           "true",
				controllerFluentbitPullSpec: "otherurl/fluentbit",
			},
			validateDaemonset: func(d *appsv1.DaemonSet) (errs []error) {
				if len(d.Spec.Template.Spec.Containers) != 2 {
					errs = append(errs, fmt.Errorf("expected 2 containers, got %d", len(d.Spec.Template.Spec.Containers)))
				}

				// we want our fluentbit image
				fluentbit, err := getContainer(d, "fluentbit")
				if err != nil {
					errs = append(errs, err)
					return
				}
				for _, err := range deep.Equal(fluentbit.Image, "otherurl/fluentbit") {
					errs = append(errs, errors.New(err))
				}

				// we want the default mdsd image
				mdsd, err := getContainer(d, "mdsd")
				if err != nil {
					errs = append(errs, err)
					return
				}
				for _, err := range deep.Equal(mdsd.Image, version.MdsdImage("acrDomain")) {
					errs = append(errs, errors.New(err))
				}

				return
			},
			mocks:          nominalMocks,
			wantErrMsg:     "",
			wantConditions: defaultConditions,
		},
		{
			name: "mdsd changed",
			operatorFlags: arov1alpha1.OperatorFlags{
				controllerEnabled:      "true",
				controllerMDSDPullSpec: "otherurl/mdsd",
			},
			validateDaemonset: func(d *appsv1.DaemonSet) (errs []error) {
				if len(d.Spec.Template.Spec.Containers) != 2 {
					errs = append(errs, fmt.Errorf("expected 2 containers, got %d", len(d.Spec.Template.Spec.Containers)))
				}

				// we want the default fluentbit image
				fluentbit, err := getContainer(d, "fluentbit")
				if err != nil {
					errs = append(errs, err)
					return
				}
				for _, err := range deep.Equal(fluentbit.Image, version.FluentbitImage("acrDomain")) {
					errs = append(errs, errors.New(err))
				}

				// we want the default mdsd image
				mdsd, err := getContainer(d, "mdsd")
				if err != nil {
					errs = append(errs, err)
					return
				}
				for _, err := range deep.Equal(mdsd.Image, "otherurl/mdsd") {
					errs = append(errs, errors.New(err))
				}

				return
			},
			mocks:      nominalMocks,
			wantErrMsg: "",
			wantConditions: []operatorv1.OperatorCondition{
				utilconditions.ControllerDefaultAvailable(controllerName),
				utilconditions.ControllerDefaultProgressing(controllerName),
				utilconditions.ControllerDefaultDegraded(controllerName),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			instance := &arov1alpha1.Cluster{
				ObjectMeta: metav1.ObjectMeta{Name: "cluster"},
				Status:     arov1alpha1.ClusterStatus{Conditions: defaultConditions},
				Spec: arov1alpha1.ClusterSpec{
					ResourceID:    testdatabase.GetResourcePath("00000000-0000-0000-0000-000000000000", "testcluster"),
					OperatorFlags: tt.operatorFlags,
					ACRDomain:     "acrDomain",
				},
			}

			resources := []client.Object{
				instance,
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: operator.Namespace,
						Name:      operator.SecretName,
					},
					Data: map[string][]byte{
						GenevaCertName: {},
						GenevaKeyName:  {},
					},
				},
				&securityv1.SecurityContextConstraints{
					ObjectMeta: metav1.ObjectMeta{
						Name: "privileged",
					},
				},
			}

			log := logrus.NewEntry(logrus.StandardLogger())
			client := ctrlfake.NewClientBuilder().WithObjects(resources...).Build()
			mockDh := mock_dynamichelper.NewMockInterface(controller)

			r := NewReconciler(log, client, mockDh)

			daemonset, err := r.daemonset(instance)
			if err != nil {
				t.Fatal(err)
			}

			errs := tt.validateDaemonset(daemonset)
			for _, err := range errs {
				t.Error(err)
			}

			tt.mocks(mockDh)
			ctx := context.Background()
			_, err = r.Reconcile(ctx, tt.request)

			utilerror.AssertErrorMessage(t, err, tt.wantErrMsg)
			utilconditions.AssertControllerConditions(t, ctx, r.Client, tt.wantConditions)
		})
	}
}

func TestGenevaConfigMapResources(t *testing.T) {
	tests := []struct {
		name          string
		request       ctrl.Request
		operatorFlags arov1alpha1.OperatorFlags
		validate      func([]runtime.Object) []error
	}{
		{
			name: "enabled",
			operatorFlags: arov1alpha1.OperatorFlags{
				controllerEnabled: "true",
			},
			validate: func(r []runtime.Object) (errs []error) {
				maps := make(map[string]*corev1.ConfigMap)
				for _, i := range r {
					if d, ok := i.(*corev1.ConfigMap); ok {
						maps[d.Name] = d
					}
				}

				c, ok := maps["fluent-config"]
				if !ok {
					errs = append(errs, errors.New("missing fluent-config"))
				} else {
					fConf := c.Data["fluent.conf"]
					pConf := c.Data["parsers.conf"]

					if !strings.Contains(fConf, "[INPUT]") {
						errs = append(errs, errors.New("incorrect fluent-config fluent.conf"))
					}

					if !strings.Contains(pConf, "[PARSER]") {
						errs = append(errs, errors.New("incorrect fluent-config parser.conf"))
					}
				}

				return
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			instance := &arov1alpha1.Cluster{
				ObjectMeta: metav1.ObjectMeta{Name: "cluster"},
				Status:     arov1alpha1.ClusterStatus{Conditions: []operatorv1.OperatorCondition{}},
				Spec: arov1alpha1.ClusterSpec{
					ResourceID:    testdatabase.GetResourcePath("00000000-0000-0000-0000-000000000000", "testcluster"),
					OperatorFlags: tt.operatorFlags,
					ACRDomain:     "acrDomain",
				},
			}

			scc := &securityv1.SecurityContextConstraints{
				ObjectMeta: metav1.ObjectMeta{Name: "privileged"},
			}

			r := &Reconciler{
				AROController: base.AROController{
					Log:    logrus.NewEntry(logrus.StandardLogger()),
					Client: ctrlfake.NewClientBuilder().WithObjects(instance, scc).Build(),
					Name:   ControllerName,
				},
			}

			out, err := r.resources(context.Background(), instance, []byte{}, []byte{})
			if err != nil {
				t.Fatal(err)
			}

			errs := tt.validate(out)
			for _, err := range errs {
				t.Error(err)
			}
		})
	}
}
