package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	configv1 "github.com/openshift/api/config/v1"
	operatorv1 "github.com/openshift/api/operator/v1"
	samplesv1 "github.com/openshift/api/samples/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"
)

// disableSamples disables the samples if there's no appropriate pull secret
func (m *manager) disableSamples(ctx context.Context) error {
	if !m.shouldDisableSamples() {
		return nil
	}

	return retry.OnError(
		retry.DefaultRetry,
		func(err error) bool {
			return errors.IsConflict(err) || errors.IsNotFound(err)
		},
		func() error {
			c, err := m.samplescli.SamplesV1().Configs().Get(ctx, "cluster", metav1.GetOptions{})

			// if config cr is missing, we create it with managementState: Removed to
			// immediately disable the samples operator without waiting for it to
			// bootstrap and create the resource
			//
			// https://docs.openshift.com/container-platform/4.17/openshift_images/configuring-samples-operator.html#samples-operator-restricted-network-install-with-access
			if errors.IsNotFound(err) {
				_, err := m.samplescli.SamplesV1().Configs().Create(ctx, bootstrapDisabledSamplesConfig(), metav1.CreateOptions{})

				return err
			} else if err != nil {
				return err
			}

			c.Spec.ManagementState = operatorv1.Removed

			_, err = m.samplescli.SamplesV1().Configs().Update(ctx, c, metav1.UpdateOptions{})
			return err
		})
}

func (m *manager) shouldDisableSamples() bool {
	return m.env.IsLocalDevelopmentMode() || m.doc.OpenShiftCluster.Properties.ClusterProfile.PullSecret == ""
}

func bootstrapDisabledSamplesConfig() *samplesv1.Config {
	return &samplesv1.Config{
		TypeMeta: metav1.TypeMeta{
			APIVersion: samplesv1.SchemeGroupVersion.Identifier(),
			Kind:       "Config",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "cluster",
		},
		Spec: samplesv1.ConfigSpec{
			ManagementState: operatorv1.Removed,
		},
	}
}

// disableOperatorHubSources disables operator hub sources if there's no
// appropriate pull secret
func (m *manager) disableOperatorHubSources(ctx context.Context) error {
	if !m.env.IsLocalDevelopmentMode() &&
		m.doc.OpenShiftCluster.Properties.ClusterProfile.PullSecret != "" {
		return nil
	}

	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		c, err := m.configcli.ConfigV1().OperatorHubs().Get(ctx, "cluster", metav1.GetOptions{})
		if err != nil {
			return err
		}

		sources := []configv1.HubSource{
			{
				Name:     "certified-operators",
				Disabled: true,
			},
			{
				Name:     "community-operators",
				Disabled: true,
			},
			{
				Name:     "redhat-marketplace",
				Disabled: true,
			},
			{
				Name:     "redhat-operators",
				Disabled: true,
			},
		}
		for _, s := range c.Spec.Sources {
			switch s.Name {
			case "certified-operators", "community-operators",
				"redhat-marketplace", "redhat-operators":
			default:
				sources = append(sources, s)
			}
		}
		c.Spec.Sources = sources

		err = m.configcli.ConfigV1().RESTClient().Put().
			Resource("operatorhubs").
			Name("cluster").
			Body(c).
			Do(ctx).
			Into(c)
		return err
	})
}
