package genevalogging

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"errors"
	"fmt"
	"testing"

	"github.com/go-test/deep"
	operatorv1 "github.com/openshift/api/operator/v1"
	"github.com/sirupsen/logrus"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"

	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	arofake "github.com/Azure/ARO-RP/pkg/operator/clientset/versioned/fake"
	"github.com/Azure/ARO-RP/pkg/util/version"
	testdatabase "github.com/Azure/ARO-RP/test/database"
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
	tests := []struct {
		name              string
		request           ctrl.Request
		arocli            *arofake.Clientset
		operatorFlags     arov1alpha1.OperatorFlags
		validateDaemonset func(*appsv1.DaemonSet) []error
	}{
		{
			name: "no flags given",
			operatorFlags: arov1alpha1.OperatorFlags{
				ENABLED: "true",
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
		},
		{
			name: "fluentbit changed",
			operatorFlags: arov1alpha1.OperatorFlags{
				ENABLED:            "true",
				FLUENTBIT_PULLSPEC: "otherurl/fluentbit",
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
		},
		{
			name: "mdsd changed",
			operatorFlags: arov1alpha1.OperatorFlags{
				ENABLED:       "true",
				MDSD_PULLSPEC: "otherurl/mdsd",
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
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			cluster := &arov1alpha1.Cluster{
				ObjectMeta: metav1.ObjectMeta{Name: "cluster"},
				Status:     arov1alpha1.ClusterStatus{Conditions: []operatorv1.OperatorCondition{}},
				Spec: arov1alpha1.ClusterSpec{
					ResourceID:    testdatabase.GetResourcePath("00000000-0000-0000-0000-000000000000", "testcluster"),
					OperatorFlags: tt.operatorFlags,
					ACRDomain:     "acrDomain",
				},
			}

			r := &Reconciler{
				log:    logrus.NewEntry(logrus.StandardLogger()),
				arocli: arofake.NewSimpleClientset(cluster),
			}

			daemonset, err := r.daemonset(cluster)
			if err != nil {
				t.Fatal(err)
			}

			errs := tt.validateDaemonset(daemonset)
			for _, err := range errs {
				t.Error(err)
			}

		})
	}
}
