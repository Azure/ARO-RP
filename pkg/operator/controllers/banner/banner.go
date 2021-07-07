package banner

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	consolev1 "github.com/openshift/api/console/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
)

const (
	BannerName         = "openshift-aro-sre"
	TextContactSupport = "Please ask your cluster administrator to contact Azure or Red Hat support."
)

func (r *BannerReconciler) reconcileBanner(ctx context.Context, instance *arov1alpha1.Cluster) error {
	var err error
	switch instance.Spec.Banner.Content {
	case arov1alpha1.BannerEmpty:
		_ = r.consolecli.ConsoleV1().ConsoleNotifications().Delete(ctx, BannerName, metav1.DeleteOptions{})
	case arov1alpha1.BannerContactSupport:
		_, err = r.consolecli.ConsoleV1().ConsoleNotifications().Create(ctx, r.newBanner(TextContactSupport, instance.Spec.ResourceID), metav1.CreateOptions{})
	}
	return err
}

func (r *BannerReconciler) newBanner(text string, resourceID string) *consolev1.ConsoleNotification {
	return &consolev1.ConsoleNotification{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ConsoleNotification",
			APIVersion: "console.openshift.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: BannerName,
		},
		Spec: consolev1.ConsoleNotificationSpec{
			Text:            TextContactSupport + " Your cluster's resourceid: " + resourceID,
			Location:        consolev1.BannerTop,
			Color:           "#000",
			BackgroundColor: "#ff0",
		},
	}
}
