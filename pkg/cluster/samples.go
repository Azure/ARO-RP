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
func (i *manager) disableSamples(ctx context.Context) error {
	if i.env.DeploymentMode() != deployment.Development &&
		i.doc.OpenShiftCluster.Properties.ClusterProfile.PullSecret != "" {
		return nil
	}

	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		c, err := i.samplescli.SamplesV1().Configs().Get("cluster", metav1.GetOptions{})
		if err != nil {
			return err
		}

		c.Spec.ManagementState = operatorv1.Removed

		_, err = i.samplescli.SamplesV1().Configs().Update(c)
		return err
	})
}

// disableOperatorHubSources disables operator hub sources if there's no
// appropriate pull secret
func (i *manager) disableOperatorHubSources(ctx context.Context) error {
	if i.env.DeploymentMode() != deployment.Development &&
		i.doc.OpenShiftCluster.Properties.ClusterProfile.PullSecret != "" {
		return nil
	}

	// https://bugzilla.redhat.com/show_bug.cgi?id=1815649
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		c := &configv1.OperatorHub{}
		err := i.configcli.ConfigV1().RESTClient().Get().
			Resource("operatorhubs").
			Name("cluster").
			VersionedParams(&metav1.GetOptions{}, configscheme.ParameterCodec).
			Do().
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

		err = i.configcli.ConfigV1().RESTClient().Put().
			Resource("operatorhubs").
			Name("cluster").
			Body(c).
			Do().
			Into(c)
		return err
	})
}
