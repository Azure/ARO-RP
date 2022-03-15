package imageregistry

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"reflect"
	"strconv"
	"testing"

	"github.com/sirupsen/logrus"

	imageregistryv1 "github.com/openshift/api/imageregistry/v1"
	imageregistryclient "github.com/openshift/client-go/imageregistry/clientset/versioned"
	imageregistryfake "github.com/openshift/client-go/imageregistry/clientset/versioned/fake"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	arofake "github.com/Azure/ARO-RP/pkg/operator/clientset/versioned/fake"
)

func TestReconciler(t *testing.T) {
	// Common API objects
	clusterSpecFeatureEnabled := arov1alpha1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name: arov1alpha1.SingletonClusterName,
		},
		Spec: arov1alpha1.ClusterSpec{
			InfraID: "aro-fake",
			OperatorFlags: arov1alpha1.OperatorFlags{
				controllerEnabled: strconv.FormatBool(true),
			},
		},
	}

	t.Run("should ignore requests when controller is disabled", func(t *testing.T) {
		// Given
		clusterSpecFeatureDisabled := arov1alpha1.Cluster{
			ObjectMeta: metav1.ObjectMeta{
				Name: arov1alpha1.SingletonClusterName,
			},
			Spec: arov1alpha1.ClusterSpec{
				InfraID: "aro-fake",
				OperatorFlags: arov1alpha1.OperatorFlags{
					controllerEnabled: strconv.FormatBool(false),
				},
			},
		}
		imageregistryConfig := imageregistryv1.Config{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "cluster",
				Namespace: "",
			},
			Spec: imageregistryv1.ImageRegistrySpec{
				DisableRedirect: false,
			},
		}
		reconciler := getReconciler(clusterSpecFeatureDisabled, imageregistryConfig)
		request := getReconcileRequestForImageRegistryConfig(imageregistryConfig)
		wantResult := reconcile.Result{}

		// When
		gotResult, err := reconciler.Reconcile(context.Background(), request)

		// Then
		assertNoError(t, err)
		assertControllerResultsEqual(t, gotResult, wantResult)
		assertCurrentImageregistryConfigDisableRedirectEquals(t, reconciler.imageregistrycli, imageregistryConfig, false)
	})

	t.Run("should ignore requests for configs not named 'cluster'", func(t *testing.T) {
		// Given
		imageregistryConfig := imageregistryv1.Config{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "foo",
				Namespace: "",
			},
			Spec: imageregistryv1.ImageRegistrySpec{
				DisableRedirect: false,
			},
		}
		reconciler := getReconciler(clusterSpecFeatureEnabled, imageregistryConfig)
		request := getReconcileRequestForImageRegistryConfig(imageregistryConfig)
		wantResult := reconcile.Result{}

		// When
		gotResult, err := reconciler.Reconcile(context.Background(), request)

		// Then
		assertNoError(t, err)
		assertControllerResultsEqual(t, gotResult, wantResult)
		assertCurrentImageregistryConfigDisableRedirectEquals(t, reconciler.imageregistrycli, imageregistryConfig, false)
	})

	t.Run("should modify 'cluster' config to disable redirect", func(t *testing.T) {
		// Given
		imageregistryConfig := imageregistryv1.Config{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "cluster",
				Namespace: "",
			},
			Spec: imageregistryv1.ImageRegistrySpec{
				DisableRedirect: false,
			},
		}
		reconciler := getReconciler(clusterSpecFeatureEnabled, imageregistryConfig)
		request := getReconcileRequestForImageRegistryConfig(imageregistryConfig)
		wantResult := reconcile.Result{}

		// When
		gotResult, err := reconciler.Reconcile(context.Background(), request)

		// Then
		assertNoError(t, err)
		assertControllerResultsEqual(t, gotResult, wantResult)
		assertCurrentImageregistryConfigDisableRedirectEquals(t, reconciler.imageregistrycli, imageregistryConfig, true)
	})

	t.Run("should no-op 'cluster' config if already disabled", func(t *testing.T) {
		// Given
		imageregistryConfig := imageregistryv1.Config{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "cluster",
				Namespace: "",
			},
			Spec: imageregistryv1.ImageRegistrySpec{
				DisableRedirect: true,
			},
		}
		reconciler := getReconciler(clusterSpecFeatureEnabled, imageregistryConfig)
		request := getReconcileRequestForImageRegistryConfig(imageregistryConfig)
		wantResult := reconcile.Result{}

		// When
		gotResult, err := reconciler.Reconcile(context.Background(), request)

		// Then
		assertNoError(t, err)
		assertControllerResultsEqual(t, gotResult, wantResult)
		assertCurrentImageregistryConfigDisableRedirectEquals(t, reconciler.imageregistrycli, imageregistryConfig, true)
	})
}

func getReconcileRequestForImageRegistryConfig(imageregistryConfig imageregistryv1.Config) reconcile.Request {
	return reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      imageregistryConfig.Name,
			Namespace: "",
		},
	}
}

func getReconciler(cluster arov1alpha1.Cluster, imageregistryConfig imageregistryv1.Config) *Reconciler {
	return NewReconciler(
		logrus.NewEntry(logrus.StandardLogger()),
		arofake.NewSimpleClientset(&cluster),
		imageregistryfake.NewSimpleClientset(&imageregistryConfig),
	)
}

func assertNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Errorf("Got unexpected error %v", err)
	}
}

func assertControllerResultsEqual(t *testing.T, got, want reconcile.Result) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Controller results were not equal: got %v wanted %v", got, want)
	}
}

func assertCurrentImageregistryConfigDisableRedirectEquals(t *testing.T, imageregistryCli imageregistryclient.Interface, imageregistryConfig imageregistryv1.Config, want bool) {
	t.Helper()
	// fetch the updated object (assume we already called Reconcile...)
	upToDateImageregistryConfig, err := imageregistryCli.ImageregistryV1().Configs().Get(context.Background(), imageregistryConfig.Name, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("Got unexpected error while looking up image registry config DisableRegistry from fake client... check your test code!")
	}
	if upToDateImageregistryConfig.Spec.DisableRedirect != want {
		t.Errorf("Image registry config had incorrect DisableRegistry setting %t", imageregistryConfig.Spec.DisableRedirect)
	}
}
