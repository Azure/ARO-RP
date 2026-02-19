package dns

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/sirupsen/logrus"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

var infrastructureGVK = schema.GroupVersionKind{
	Group:   "config.openshift.io",
	Version: "v1",
	Kind:    "Infrastructure",
}

const infrastructureName = "cluster"

// cloudLoadBalancerIPs holds the load balancer IPs that must be set on the
// Infrastructure CR for CustomDNS (ClusterHostedDNS) to function.
// These IPs are used by MCO to render the CoreDNS static pod Corefile.
type cloudLoadBalancerIPs struct {
	APIIntLoadBalancerIPs  []string `json:"apiIntLoadBalancerIPs"`
	APILoadBalancerIPs     []string `json:"apiLoadBalancerIPs"`
	IngressLoadBalancerIPs []string `json:"ingressLoadBalancerIPs,omitempty"`
}

// cloudLoadBalancerConfig represents the CloudLoadBalancerConfig section of
// the Infrastructure CR's status.platformStatus.azure field.
type cloudLoadBalancerConfig struct {
	DNSType       string                `json:"dnsType"`
	ClusterHosted *cloudLoadBalancerIPs `json:"clusterHosted,omitempty"`
}

// reconcileInfrastructureCR ensures the Infrastructure CR's cloudLoadBalancerConfig
// is set with the correct LB IPs for CustomDNS operation.
// It reads the current state, compares with the desired state derived from the
// ARO Cluster spec, and patches if they differ.
func reconcileInfrastructureCR(ctx context.Context, c client.Client, log *logrus.Entry, apiIntIP, ingressIP string) error {
	infra, err := getInfrastructureCR(ctx, c)
	if err != nil {
		return fmt.Errorf("failed to get Infrastructure CR: %w", err)
	}

	desired := buildDesiredCloudLBConfig(apiIntIP, ingressIP)
	current := getCurrentCloudLBConfig(infra)

	if reflect.DeepEqual(current, desired) {
		log.Debug("Infrastructure CR cloudLoadBalancerConfig is up to date")
		return nil
	}

	log.Infof("Infrastructure CR cloudLoadBalancerConfig needs update: current=%+v desired=%+v", current, desired)

	if err := patchInfrastructureStatus(ctx, c, infra, desired); err != nil {
		return fmt.Errorf("failed to patch Infrastructure CR status: %w", err)
	}

	log.Info("Infrastructure CR cloudLoadBalancerConfig updated successfully")
	return nil
}

// getInfrastructureCR retrieves the Infrastructure CR as an unstructured object.
// Using unstructured because the vendored openshift/api does not include
// CloudLoadBalancerConfig on AzurePlatformStatus.
func getInfrastructureCR(ctx context.Context, c client.Client) (*unstructured.Unstructured, error) {
	infra := &unstructured.Unstructured{}
	infra.SetGroupVersionKind(infrastructureGVK)
	err := c.Get(ctx, client.ObjectKey{Name: infrastructureName}, infra)
	return infra, err
}

// buildDesiredCloudLBConfig constructs the desired CloudLoadBalancerConfig
// from the ARO Cluster spec's APIIntIP and IngressIP.
// Per the TDR, apiLoadBalancerIPs is set to the same value as apiIntLoadBalancerIPs.
func buildDesiredCloudLBConfig(apiIntIP, ingressIP string) *cloudLoadBalancerConfig {
	cfg := &cloudLoadBalancerConfig{
		DNSType: "ClusterHosted",
		ClusterHosted: &cloudLoadBalancerIPs{
			APIIntLoadBalancerIPs: []string{apiIntIP},
			APILoadBalancerIPs:    []string{apiIntIP},
		},
	}
	if ingressIP != "" {
		cfg.ClusterHosted.IngressLoadBalancerIPs = []string{ingressIP}
	}
	return cfg
}

// getCurrentCloudLBConfig reads the current cloudLoadBalancerConfig from
// the Infrastructure CR's status.platformStatus.azure.
func getCurrentCloudLBConfig(infra *unstructured.Unstructured) *cloudLoadBalancerConfig {
	dnsType, found, err := unstructured.NestedString(infra.Object,
		"status", "platformStatus", "azure", "cloudLoadBalancerConfig", "dnsType")
	if !found || err != nil {
		return nil
	}

	apiIntIPs, _, _ := unstructured.NestedStringSlice(infra.Object,
		"status", "platformStatus", "azure", "cloudLoadBalancerConfig", "clusterHosted", "apiIntLoadBalancerIPs")
	apiIPs, _, _ := unstructured.NestedStringSlice(infra.Object,
		"status", "platformStatus", "azure", "cloudLoadBalancerConfig", "clusterHosted", "apiLoadBalancerIPs")
	ingressIPs, _, _ := unstructured.NestedStringSlice(infra.Object,
		"status", "platformStatus", "azure", "cloudLoadBalancerConfig", "clusterHosted", "ingressLoadBalancerIPs")

	return &cloudLoadBalancerConfig{
		DNSType: dnsType,
		ClusterHosted: &cloudLoadBalancerIPs{
			APIIntLoadBalancerIPs:  apiIntIPs,
			APILoadBalancerIPs:     apiIPs,
			IngressLoadBalancerIPs: ingressIPs,
		},
	}
}

// patchInfrastructureStatus applies a merge patch to the Infrastructure CR's
// status subresource to set the cloudLoadBalancerConfig.
func patchInfrastructureStatus(ctx context.Context, c client.Client, infra *unstructured.Unstructured, desired *cloudLoadBalancerConfig) error {
	patch := map[string]any{
		"status": map[string]any{
			"platformStatus": map[string]any{
				"azure": map[string]any{
					"cloudLoadBalancerConfig": map[string]any{
						"dnsType": desired.DNSType,
						"clusterHosted": map[string]any{
							"apiIntLoadBalancerIPs":  toInterfaceSlice(desired.ClusterHosted.APIIntLoadBalancerIPs),
							"apiLoadBalancerIPs":     toInterfaceSlice(desired.ClusterHosted.APILoadBalancerIPs),
							"ingressLoadBalancerIPs": toInterfaceSlice(desired.ClusterHosted.IngressLoadBalancerIPs),
						},
					},
				},
			},
		},
	}

	patchBytes, err := json.Marshal(patch)
	if err != nil {
		return err
	}

	return c.Status().Patch(ctx, infra, client.RawPatch(types.MergePatchType, patchBytes))
}

// toInterfaceSlice converts a []string to []interface{} for use in
// unstructured map building.
func toInterfaceSlice(s []string) []any {
	result := make([]any, len(s))
	for i, v := range s {
		result[i] = v
	}
	return result
}
