package banner

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"

	consolev1 "github.com/openshift/api/console/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
)

func (r *Reconciler) reconcileBanner(ctx context.Context, instance *arov1alpha1.Cluster) (err error) {
	switch instance.Spec.Banner.Content {
	case arov1alpha1.BannerDisabled:
		err = r.consolecli.ConsoleV1().ConsoleNotifications().Delete(ctx, BannerName, metav1.DeleteOptions{})
		if err != nil && kerrors.IsNotFound(err) {
			// we don't care if the object doesn't exist
			return nil
		}
		return err
	case arov1alpha1.BannerContactSupport:
		banner := r.newBanner(TextContactSupport, instance.Spec.ResourceID)
		_, err := r.consolecli.ConsoleV1().ConsoleNotifications().Get(ctx, BannerName, metav1.GetOptions{})
		if err != nil && kerrors.IsNotFound(err) {
			// if the object doesn't exist Create
			_, err = r.consolecli.ConsoleV1().ConsoleNotifications().Create(ctx, banner, metav1.CreateOptions{})
			return err
		}
		if err != nil {
			// if there's a different error - surface it
			return err
		}
		// if there's no errors, object found then update
		_, err = r.consolecli.ConsoleV1().ConsoleNotifications().Update(ctx, banner, metav1.UpdateOptions{})
		return err
	}
	return fmt.Errorf("wrong banner setting '%s'", instance.Spec.Banner.Content)
}

func (r *Reconciler) newBanner(text string, resourceID string) *consolev1.ConsoleNotification {
	return &consolev1.ConsoleNotification{
		ObjectMeta: metav1.ObjectMeta{
			Name: BannerName,
		},
		Spec: consolev1.ConsoleNotificationSpec{
			Text:            fmt.Sprintf(TextContactSupport, resourceID),
			Location:        consolev1.BannerTop,
			Color:           "#000",
			BackgroundColor: "#ff0",
		},
	}
}
