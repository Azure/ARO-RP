package banner

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"

	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	consolev1 "github.com/openshift/api/console/v1"

	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
)

func (r *Reconciler) reconcileBanner(ctx context.Context, instance *arov1alpha1.Cluster) error {
	var text string

	switch instance.Spec.Banner.Content {
	case arov1alpha1.BannerDisabled:
		banner := &consolev1.ConsoleNotification{
			ObjectMeta: metav1.ObjectMeta{
				Name: BannerName,
			},
		}
		err := r.client.Delete(ctx, banner)
		if err != nil && kerrors.IsNotFound(err) {
			// we don't care if the object doesn't exist
			return nil
		}
		return err
	case arov1alpha1.BannerContactSupport:
		text = fmt.Sprintf(TextContactSupport, instance.Spec.ResourceID)
	default:
		return fmt.Errorf("wrong banner setting '%s'", instance.Spec.Banner.Content)
	}

	return r.createOrUpdate(ctx, text)
}

func (r *Reconciler) createOrUpdate(ctx context.Context, text string) error {
	oldBanner := &consolev1.ConsoleNotification{}
	err := r.client.Get(ctx, types.NamespacedName{Name: BannerName}, oldBanner)
	if err != nil && !kerrors.IsNotFound(err) {
		return err
	}

	// if the object doesn't exist Create
	if err != nil && kerrors.IsNotFound(err) {
		return r.client.Create(ctx, r.newBanner(text))
	}

	// if there's no errors, object found then update
	oldBanner.Spec.Text = text
	return r.client.Update(ctx, oldBanner)
}

func (r *Reconciler) newBanner(text string) *consolev1.ConsoleNotification {
	return &consolev1.ConsoleNotification{
		ObjectMeta: metav1.ObjectMeta{
			Name: BannerName,
		},
		Spec: consolev1.ConsoleNotificationSpec{
			Text:            text,
			Location:        consolev1.BannerTop,
			Color:           "#000",
			BackgroundColor: "#ff0",
		},
	}
}
