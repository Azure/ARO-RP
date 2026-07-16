package clusterdata

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"sort"

	"github.com/sirupsen/logrus"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	operatorv1 "github.com/openshift/api/operator/v1"
	configclient "github.com/openshift/client-go/config/clientset/versioned"
	machineclient "github.com/openshift/client-go/machine/clientset/versioned"
	operatorclient "github.com/openshift/client-go/operator/clientset/versioned"

	"github.com/Azure/ARO-RP/pkg/api"
)

type ingressProfileEnricher struct{}

func (ip ingressProfileEnricher) Enrich(
	ctx context.Context,
	log *logrus.Entry,
	oc *api.OpenShiftCluster,
	k8scli kubernetes.Interface,
	configcli configclient.Interface,
	machinecli machineclient.Interface,
	operatorcli operatorclient.Interface,
) error {
	// List IngressControllers from  openshift-ingress-operator namespace
	// Each IngressController will be the basis for an IngressProfile with the below mapping:
	//     IngressController.Name -> IngressProfile.Name
	//     IngressController.status.endpointPublishingStrategy.loadBalancer.scope -> IngressProfile.Visibility
	//             or fall back to spec when status is not usable
	//             Internal -> Private
	//             External -> Public
	ingressControllers, err := operatorcli.OperatorV1().IngressControllers("openshift-ingress-operator").List(ctx, metav1.ListOptions{})
	if err != nil {
		return err
	}

	// List Services from openshift-ingress namespace
	// Among those Services, look for the ones of type LoadBalancer, with label "app: router". The matching will be done with
	// IngressController based on ingresscontroller.operator.openshift.io/owning-ingresscontroller label.
	// Service IP will be taken from the candidate and added to the corresponding IngressProfile
	services, err := k8scli.CoreV1().Services("openshift-ingress").List(ctx, metav1.ListOptions{
		LabelSelector: "app=router",
	})
	if err != nil {
		return err
	}

	routerIPs := make(map[string]string)
	for _, service := range services.Items {
		if len(service.Status.LoadBalancer.Ingress) == 0 {
			continue
		}
		matchingICName, ok := service.Labels["ingresscontroller.operator.openshift.io/owning-ingresscontroller"]
		if !ok {
			// Un-expected case where a router service has no owning ingress controller
			continue
		}
		routerIPs[matchingICName] = service.Status.LoadBalancer.Ingress[0].IP
	}

	existingIngressProfileVisibility := map[string]api.Visibility{}
	oc.Lock.Lock()
	for _, ingressProfile := range oc.Properties.IngressProfiles {
		existingIngressProfileVisibility[ingressProfile.Name] = ingressProfile.Visibility
	}
	oc.Lock.Unlock()

	// Reconcile IngressController and corresponding router service
	ingressProfiles := make([]api.IngressProfile, len(ingressControllers.Items))
	for i, ingressController := range ingressControllers.Items {
		ingressProfiles[i] = api.IngressProfile{
			Name: ingressController.Name,
			IP:   routerIPs[ingressController.Name],
		}

		visibility := ingressProfileVisibility(log, ingressProfiles[i].Name, ingressController)
		if visibility == "" {
			visibility = existingIngressProfileVisibility[ingressProfiles[i].Name]
		}
		ingressProfiles[i].Visibility = visibility
	}

	sort.Slice(ingressProfiles, func(i, j int) bool { return ingressProfiles[i].Name < ingressProfiles[j].Name })

	oc.Lock.Lock()
	defer oc.Lock.Unlock()

	oc.Properties.IngressProfiles = ingressProfiles

	return nil
}

func ingressProfileVisibility(log *logrus.Entry, ingressProfileName string, ingressController operatorv1.IngressController) api.Visibility {
	if visibility, ok := visibilityFromEndpointPublishingStrategy(ingressController.Status.EndpointPublishingStrategy); ok {
		return visibility
	}

	if visibility, ok := visibilityFromEndpointPublishingStrategy(ingressController.Spec.EndpointPublishingStrategy); ok {
		return visibility
	}

	log.Infof(
		"Cannot determine Visibility for IngressProfile %q. IngressController status endpointPublishingStrategy is %s and spec endpointPublishingStrategy is %s",
		ingressProfileName,
		endpointPublishingStrategyReason(ingressController.Status.EndpointPublishingStrategy),
		endpointPublishingStrategyReason(ingressController.Spec.EndpointPublishingStrategy),
	)

	return ""
}

func visibilityFromEndpointPublishingStrategy(strategy *operatorv1.EndpointPublishingStrategy) (api.Visibility, bool) {
	if strategy == nil || strategy.LoadBalancer == nil {
		return "", false
	}

	switch strategy.LoadBalancer.Scope {
	case operatorv1.InternalLoadBalancer:
		return api.VisibilityPrivate, true
	case operatorv1.ExternalLoadBalancer:
		return api.VisibilityPublic, true
	default:
		return "", false
	}
}

func endpointPublishingStrategyReason(strategy *operatorv1.EndpointPublishingStrategy) string {
	switch {
	case strategy == nil:
		return "missing"
	case strategy.LoadBalancer == nil:
		return "present but loadBalancer is nil"
	case strategy.LoadBalancer.Scope == operatorv1.InternalLoadBalancer || strategy.LoadBalancer.Scope == operatorv1.ExternalLoadBalancer:
		return fmt.Sprintf("resolvable with loadBalancer scope %q", strategy.LoadBalancer.Scope)
	default:
		return fmt.Sprintf("present but loadBalancer scope %q is unsupported", strategy.LoadBalancer.Scope)
	}
}
