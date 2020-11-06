package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	configv1 "github.com/openshift/api/config/v1"
	operatorv1 "github.com/openshift/api/operator/v1"
	configscheme "github.com/openshift/client-go/config/clientset/versioned/scheme"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"

	"github.com/Azure/ARO-RP/pkg/util/deployment"
)

// disableSamples disables the samples if there's no appropriate pull secret
func (m *manager) disableSamples(ctx context.Context) error {
	if m.env.DeploymentMode() != deployment.Development &&
		m.doc.OpenShiftCluster.Properties.ClusterProfile.PullSecret != "" {
		return nil
	}

	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		c, err := m.samplescli.SamplesV1().Configs().Get(ctx, "cluster", metav1.GetOptions{})
		if err != nil {
			return err
		}

		c.Spec.ManagementState = operatorv1.Removed

		_, err = m.samplescli.SamplesV1().Configs().Update(ctx, c, metav1.UpdateOptions{})
		return err
	})
}

// disableOperatorHubSources disables operator hub sources if there's no
// appropriate pull secret
func (m *manager) disableOperatorHubSources(ctx context.Context) error {
	if m.env.DeploymentMode() != deployment.Development &&
		m.doc.OpenShiftCluster.Properties.ClusterProfile.PullSecret != "" {
		return nil
	}

	// https://bugzilla.redhat.com/show_bug.cgi?id=1815649
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		c := &configv1.OperatorHub{}
		err := m.configcli.ConfigV1().RESTClient().Get().
			Resource("operatorhubs").
			Name("cluster").
			VersionedParams(&metav1.GetOptions{}, configscheme.ParameterCodec).
			Do(ctx).
			Into(c)
		if err != nil {
			return err
		}

		sources := []configv1.HubSource{
			{
				Name:     "certified-operators",
				Disabled: true,
			},
			{
				Name:     "redhat-operators",
				Disabled: true,
			},
		}
		for _, s := range c.Spec.Sources {
			switch s.Name {
			case "certified-operators", "redhat-operators":
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
