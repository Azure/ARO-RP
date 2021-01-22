package clusterdata

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	operatorv1 "github.com/openshift/api/operator/v1"
	operatorclient "github.com/openshift/client-go/operator/clientset/versioned"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/Azure/ARO-RP/pkg/api"
)

func newIngressProfileEnricherTask(log *logrus.Entry, restConfig *rest.Config, oc *api.OpenShiftCluster) (enricherTask, error) {
	operatorcli, err := operatorclient.NewForConfig(restConfig)
	if err != nil {
		return nil, err
	}

	kubecli, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, err
	}

	return &ingressProfileEnricherTask{
		log:         log,
		operatorcli: operatorcli,
		kubecli:     kubecli,
		oc:          oc,
	}, nil
}

type ingressProfileEnricherTask struct {
	log         *logrus.Entry
	operatorcli operatorclient.Interface
	kubecli     kubernetes.Interface
	oc          *api.OpenShiftCluster
}

func (ef *ingressProfileEnricherTask) FetchData(ctx context.Context, callbacks chan<- func(), errs chan<- error) {
	// List IngressControllers from  openshift-ingress-operator namespace
	// Each IngressController will be the basis for an IngressProfile with the below mapping:
	//     IngressController.Name -> IngressProfile.Name
	//     IngressController.spec.endpointPublishingStrategy.loadBalancer.scope -> IngressProfile.Visibility
	//             Internal -> Private
	//             External -> Public
	ingressControllers, err := ef.operatorcli.OperatorV1().IngressControllers("openshift-ingress-operator").List(ctx, metav1.ListOptions{})
	if err != nil {
		ef.log.Error(err)
		errs <- err
		return
	}

	// List Services from openshift-ingress namespace
	// Among those Services, look for the ones of type LoadBalancer, with label "app: router". The matching will be done with
	// IngressController based on ingresscontroller.operator.openshift.io/owning-ingresscontroller label.
	// Service IP will be taken from the candidate and added to the corresponding IngressProfile
	services, err := ef.kubecli.CoreV1().Services("openshift-ingress").List(ctx, metav1.ListOptions{
		LabelSelector: "app=router",
	})
	if err != nil {
		ef.log.Error(err)
		errs <- err
		return
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
			ef.log.Infof("Cannot determine Visibility for IngressProfile %q. IngressController has EndpointPublishingStrategy but LoadBalancer is nil", ingressProfiles[i].Name)
		case ingressController.Spec.EndpointPublishingStrategy.LoadBalancer.Scope == operatorv1.InternalLoadBalancer:
			visibility = api.VisibilityPrivate
		case ingressController.Spec.EndpointPublishingStrategy.LoadBalancer.Scope == operatorv1.ExternalLoadBalancer:
			visibility = api.VisibilityPublic
		default:
			ef.log.Infof("Cannot determine Visibility for IngressProfile %q. IngressController EndpointPublishingStrategy.LoadBalancer has unexpected Scope value %q",
				ingressProfiles[i].Name,
				ingressController.Spec.EndpointPublishingStrategy.LoadBalancer.Scope)
		}

		ingressProfiles[i].Visibility = visibility
	}

	callbacks <- func() {
		ef.oc.Properties.IngressProfiles = ingressProfiles
	}
}

func (ef *ingressProfileEnricherTask) SetDefaults() {
	ef.oc.Properties.IngressProfiles = nil
}
