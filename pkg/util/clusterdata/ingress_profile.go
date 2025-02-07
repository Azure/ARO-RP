package clusterdata

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
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

type ingressProfileEnricher struct {
}

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
	//     IngressController.spec.endpointPublishingStrategy.loadBalancer.scope -> IngressProfile.Visibility
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
		matchingICName, ok := service.ObjectMeta.Labels["ingresscontroller.operator.openshift.io/owning-ingresscontroller"]
		if !ok {
			// Un-expected case where a router service has no owning ingress controller
			continue
		}
		routerIPs[matchingICName] = service.Status.LoadBalancer.Ingress[0].IP
	}

	// Reconcile IngressController and corresponding router service
	ingressProfiles := make([]api.IngressProfile, len(ingressControllers.Items))
	for i, ingressController := range ingressControllers.Items {
		ingressProfiles[i] = api.IngressProfile{
			Name: ingressController.ObjectMeta.Name,
			IP:   routerIPs[ingressController.ObjectMeta.Name],
		}

		var visibility api.Visibility
		switch {
		case ingressController.Spec.EndpointPublishingStrategy == nil:
			// Default case on Azure, LoadBalancerStrategy with External scope
			// See https://docs.openshift.com/container-platform/4.6/networking/ingress-operator.html#nw-ingress-controller-configuration-parameters_configuring-ingress
			visibility = api.VisibilityPublic
		case ingressController.Spec.EndpointPublishingStrategy.LoadBalancer == nil:
			log.Infof("Cannot determine Visibility for IngressProfile %q. IngressController has EndpointPublishingStrategy but LoadBalancer is nil", ingressProfiles[i].Name)
		case ingressController.Spec.EndpointPublishingStrategy.LoadBalancer.Scope == operatorv1.InternalLoadBalancer:
			visibility = api.VisibilityPrivate
		case ingressController.Spec.EndpointPublishingStrategy.LoadBalancer.Scope == operatorv1.ExternalLoadBalancer:
			visibility = api.VisibilityPublic
		default:
			log.Infof("Cannot determine Visibility for IngressProfile %q. IngressController EndpointPublishingStrategy.LoadBalancer has unexpected Scope value %q",
				ingressProfiles[i].Name,
				ingressController.Spec.EndpointPublishingStrategy.LoadBalancer.Scope)
		}

		ingressProfiles[i].Visibility = visibility
	}

	sort.Slice(ingressProfiles, func(i, j int) bool { return ingressProfiles[i].Name < ingressProfiles[j].Name })

	oc.Lock.Lock()
	defer oc.Lock.Unlock()

	oc.Properties.IngressProfiles = ingressProfiles

	return nil
}

func (ip ingressProfileEnricher) SetDefaults(oc *api.OpenShiftCluster) {
	oc.Properties.IngressProfiles = nil
}
