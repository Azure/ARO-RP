package genevalogging

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/go-test/deep"
	"github.com/sirupsen/logrus"
	"go.uber.org/mock/gomock"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrlfake "sigs.k8s.io/controller-runtime/pkg/client/fake"

	configv1 "github.com/openshift/api/config/v1"
	operatorv1 "github.com/openshift/api/operator/v1"
	securityv1 "github.com/openshift/api/security/v1"

	"github.com/Azure/ARO-RP/pkg/operator"
	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	"github.com/Azure/ARO-RP/pkg/operator/controllers/base"
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

func clusterVersion(version string) configv1.ClusterVersion {
	return configv1.ClusterVersion{
		ObjectMeta: metav1.ObjectMeta{
			Name: "version",
		},
		Spec: configv1.ClusterVersionSpec{},
		Status: configv1.ClusterVersionStatus{
			History: []configv1.UpdateHistory{
				{
					State:   configv1.CompletedUpdate,
					Version: version,
				},
			},
		},
	}
}

func TestGenevaLoggingNamespaceLabels(t *testing.T) {
	tests := []struct {
		name       string
		cv         configv1.ClusterVersion
		wantLabels map[string]string
		wantErr    string
	}{
		{
			name:       "cluster < 4.11, no labels",
			cv:         clusterVersion("4.10.99"),
			wantLabels: map[string]string{},
		},
		{
			name:       "cluster >= 4.11, use pod security labels",
			cv:         clusterVersion("4.11.0"),
			wantLabels: privilegedNamespaceLabels,
		},
		{
			name:    "cluster version doesn't exist",
			cv:      configv1.ClusterVersion{},
			wantErr: `clusterversions.config.openshift.io "version" not found`,
		},
		{
			name:    "invalid version",
			cv:      clusterVersion("abcd"),
			wantErr: `could not parse version "abcd"`,
		},
	}
	for _, tt := range tests {
		ctx := context.Background()

		controller := gomock.NewController(t)
		defer controller.Finish()

		mockDh := mock_dynamichelper.NewMockInterface(controller)

		r := &Reconciler{
			AROController: base.AROController{
				Log:    logrus.NewEntry(logrus.StandardLogger()),
				Client: ctrlfake.NewClientBuilder().WithObjects(&tt.cv).Build(),
				Name:   ControllerName,
			},
			dh: mockDh,
		}

		labels, err := r.namespaceLabels(ctx)
		utilerror.AssertErrorMessage(t, err, tt.wantErr)

		if !reflect.DeepEqual(labels, tt.wantLabels) {
			t.Errorf("got: %v\nwanted:%v\n", labels, tt.wantLabels)
		}
	}
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
		utilconditions.ControllerDefaultAvailable(ControllerName),
		utilconditions.ControllerDefaultProgressing(ControllerName),
		utilconditions.ControllerDefaultDegraded(ControllerName),
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
				operator.GenevaLoggingEnabled: operator.FlagTrue,
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
			name: "fluentbit/mdsd specs provided as empty strings",
			operatorFlags: arov1alpha1.OperatorFlags{
				operator.GenevaLoggingEnabled: operator.FlagTrue,
				controllerFluentbitPullSpec:   "",
				controllerMDSDPullSpec:        "",
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
				operator.GenevaLoggingEnabled: operator.FlagTrue,
				controllerFluentbitPullSpec:   "otherurl/fluentbit",
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
				operator.GenevaLoggingEnabled: operator.FlagTrue,
				controllerMDSDPullSpec:        "otherurl/mdsd",
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
				utilconditions.ControllerDefaultAvailable(ControllerName),
				utilconditions.ControllerDefaultProgressing(ControllerName),
				utilconditions.ControllerDefaultDegraded(ControllerName),
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

			cv := clusterVersion("4.11.0")
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
				&cv,
			}

			mockDh := mock_dynamichelper.NewMockInterface(controller)

			r := &Reconciler{
				AROController: base.AROController{
					Log:    logrus.NewEntry(logrus.StandardLogger()),
					Client: ctrlfake.NewClientBuilder().WithObjects(resources...).Build(),
					Name:   ControllerName,
				},
				dh: mockDh,
			}

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
				operator.GenevaLoggingEnabled: operator.FlagTrue,
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

			cv := clusterVersion("4.11.0")
			r := &Reconciler{
				AROController: base.AROController{
					Log:    logrus.NewEntry(logrus.StandardLogger()),
					Client: ctrlfake.NewClientBuilder().WithObjects(instance, scc, &cv).Build(),
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
