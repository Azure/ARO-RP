package main

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"flag"
	"fmt"

	machinev1 "github.com/openshift/api/machine/v1"
	"github.com/sirupsen/logrus"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"

	"github.com/Azure/ARO-RP/pkg/env"
	pkgoperator "github.com/Azure/ARO-RP/pkg/operator"
	"github.com/Azure/ARO-RP/pkg/operator/controllers/alertwebhook"
	"github.com/Azure/ARO-RP/pkg/operator/controllers/autosizednodes"
	"github.com/Azure/ARO-RP/pkg/operator/controllers/banner"
	"github.com/Azure/ARO-RP/pkg/operator/controllers/base"
	"github.com/Azure/ARO-RP/pkg/operator/controllers/checkers/clusterdnschecker"
	"github.com/Azure/ARO-RP/pkg/operator/controllers/checkers/ingresscertificatechecker"
	"github.com/Azure/ARO-RP/pkg/operator/controllers/checkers/internetchecker"
	"github.com/Azure/ARO-RP/pkg/operator/controllers/checkers/serviceprincipalchecker"
	"github.com/Azure/ARO-RP/pkg/operator/controllers/cloudproviderconfig"
	"github.com/Azure/ARO-RP/pkg/operator/controllers/clusteroperatoraro"
	"github.com/Azure/ARO-RP/pkg/operator/controllers/cpms"
	"github.com/Azure/ARO-RP/pkg/operator/controllers/dnsmasq"
	"github.com/Azure/ARO-RP/pkg/operator/controllers/genevalogging"
	"github.com/Azure/ARO-RP/pkg/operator/controllers/guardrails"
	"github.com/Azure/ARO-RP/pkg/operator/controllers/imageconfig"
	"github.com/Azure/ARO-RP/pkg/operator/controllers/ingress"
	"github.com/Azure/ARO-RP/pkg/operator/controllers/machine"
	"github.com/Azure/ARO-RP/pkg/operator/controllers/machinehealthcheck"
	"github.com/Azure/ARO-RP/pkg/operator/controllers/machineset"
	"github.com/Azure/ARO-RP/pkg/operator/controllers/monitoring"
	"github.com/Azure/ARO-RP/pkg/operator/controllers/muo"
	"github.com/Azure/ARO-RP/pkg/operator/controllers/node"
	"github.com/Azure/ARO-RP/pkg/operator/controllers/previewfeature"
	"github.com/Azure/ARO-RP/pkg/operator/controllers/pullsecret"
	"github.com/Azure/ARO-RP/pkg/operator/controllers/rbac"
	"github.com/Azure/ARO-RP/pkg/operator/controllers/routefix"
	"github.com/Azure/ARO-RP/pkg/operator/controllers/storageaccounts"
	"github.com/Azure/ARO-RP/pkg/operator/controllers/subnets"
	"github.com/Azure/ARO-RP/pkg/operator/controllers/workaround"
	"github.com/Azure/ARO-RP/pkg/util/dynamichelper"
	utillog "github.com/Azure/ARO-RP/pkg/util/log"
	// +kubebuilder:scaffold:imports
)

func operator(ctx context.Context, log *logrus.Entry) error {
	role := flag.Arg(1)
	switch role {
	case pkgoperator.RoleMaster, pkgoperator.RoleWorker:
	default:
		return fmt.Errorf("invalid role %s", role)
	}
	isLocalDevelopmentMode := env.IsLocalDevelopmentMode()
	if isLocalDevelopmentMode {
		log.Info("running in local development mode")
	}

	ctrl.SetLogger(utillog.LogrWrapper(log))

	restConfig, err := ctrl.GetConfig()
	if err != nil {
		return err
	}

	mgr, err := ctrl.NewManager(restConfig, ctrl.Options{
		HealthProbeBindAddress: ":8080",
		MetricsBindAddress:     "0", // disabled
		Port:                   8443,
	})
	if err != nil {
		return err
	}

	client := mgr.GetClient()

	kubernetescli, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return err
	}
	discoverycli, err := discovery.NewDiscoveryClientForConfig(restConfig)
	if err != nil {
		return err
	}
	dh, err := dynamichelper.New(log, restConfig)
	if err != nil {
		return err
	}

	var AROReconcilers []base.AROReconciler

	if role == pkgoperator.RoleMaster {
		AROReconcilers = append(AROReconcilers,
			cloudproviderconfig.NewReconciler(log, client),
			dnsmasq.NewClusterReconciler(log, client, dh),
			dnsmasq.NewMachineConfigPoolReconciler(log, client, dh),
			dnsmasq.NewMachineConfigReconciler(log, client, dh),
			genevalogging.NewReconciler(log, client, dh),
			imageconfig.NewReconciler(log, client),
			ingress.NewReconciler(log, client),
			machinehealthcheck.NewReconciler(log, client, dh),
			machineset.NewReconciler(log, client),
			monitoring.NewReconciler(log, client),
			node.NewReconciler(log, client, kubernetescli),
		)

		// FIXME Add controllers to the slice above as they're converted to AROController.

		if err = (clusteroperatoraro.NewReconciler(
			log.WithField("controller", clusteroperatoraro.ControllerName),
			client)).SetupWithManager(mgr); err != nil {
			return fmt.Errorf("unable to create controller %s: %v", clusteroperatoraro.ControllerName, err)
		}
		if err = (pullsecret.NewReconciler(
			log.WithField("controller", pullsecret.ControllerName),
			client)).SetupWithManager(mgr); err != nil {
			return fmt.Errorf("unable to create controller %s: %v", pullsecret.ControllerName, err)
		}
		if err = (alertwebhook.NewReconciler(
			log.WithField("controller", alertwebhook.ControllerName),
			client)).SetupWithManager(mgr); err != nil {
			return fmt.Errorf("unable to create controller %s: %v", alertwebhook.ControllerName, err)
		}
		if err = (workaround.NewReconciler(
			log.WithField("controller", workaround.ControllerName),
			client)).SetupWithManager(mgr); err != nil {
			return fmt.Errorf("unable to create controller %s: %v", workaround.ControllerName, err)
		}
		if err = (routefix.NewReconciler(
			log.WithField("controller", routefix.ControllerName),
			client, dh)).SetupWithManager(mgr); err != nil {
			return fmt.Errorf("unable to create controller %s: %v", routefix.ControllerName, err)
		}
		if err = (rbac.NewReconciler(
			log.WithField("controller", rbac.ControllerName),
			client, dh)).SetupWithManager(mgr); err != nil {
			return fmt.Errorf("unable to create controller %s: %v", rbac.ControllerName, err)
		}
		if err = (subnets.NewReconciler(
			log.WithField("controller", subnets.ControllerName),
			client)).SetupWithManager(mgr); err != nil {
			return fmt.Errorf("unable to create controller %s: %v", subnets.ControllerName, err)
		}
		if err = (machine.NewReconciler(
			log.WithField("controller", machine.ControllerName),
			client, isLocalDevelopmentMode, role)).SetupWithManager(mgr); err != nil {
			return fmt.Errorf("unable to create controller %s: %v", machine.ControllerName, err)
		}
		if err = (banner.NewReconciler(
			log.WithField("controller", banner.ControllerName),
			client)).SetupWithManager(mgr); err != nil {
			return fmt.Errorf("unable to create controller %s: %v", banner.ControllerName, err)
		}
		if err = (previewfeature.NewReconciler(
			log.WithField("controller", previewfeature.ControllerName),
			client)).SetupWithManager(mgr); err != nil {
			return fmt.Errorf("unable to create controller %s: %v", previewfeature.ControllerName, err)
		}
		if err = (storageaccounts.NewReconciler(
			log.WithField("controller", storageaccounts.ControllerName),
			client)).SetupWithManager(mgr); err != nil {
			return fmt.Errorf("unable to create controller %s: %v", storageaccounts.ControllerName, err)
		}
		if err = (muo.NewReconciler(
			log.WithField("controller", muo.ControllerName),
			client, dh)).SetupWithManager(mgr); err != nil {
			return fmt.Errorf("unable to create controller %s: %v", muo.ControllerName, err)
		}
		if err = (autosizednodes.NewReconciler(
			log.WithField("controller", autosizednodes.ControllerName),
			client)).SetupWithManager(mgr); err != nil {
			return fmt.Errorf("unable to create controller %s: %v", autosizednodes.ControllerName, err)
		}
		if err = (serviceprincipalchecker.NewReconciler(
			log.WithField("controller", serviceprincipalchecker.ControllerName),
			client, role)).SetupWithManager(mgr); err != nil {
			return fmt.Errorf("unable to create controller %s: %v", serviceprincipalchecker.ControllerName, err)
		}
		if err = (clusterdnschecker.NewReconciler(
			log.WithField("controller", clusterdnschecker.ControllerName),
			client, role)).SetupWithManager(mgr); err != nil {
			return fmt.Errorf("unable to create controller %s: %v", clusterdnschecker.ControllerName, err)
		}
		if err = (ingresscertificatechecker.NewReconciler(
			log.WithField("controller", ingresscertificatechecker.ControllerName),
			client, role)).SetupWithManager(mgr); err != nil {
			return fmt.Errorf("unable to create controller %s: %v", ingresscertificatechecker.ControllerName, err)
		}
		if err = (guardrails.NewReconciler(
			log.WithField("controller", guardrails.ControllerName),
			client, dh, kubernetescli)).SetupWithManager(mgr); err != nil {
			return fmt.Errorf("unable to create controller %s: %v", guardrails.ControllerName, err)
		}

		// only register CPMS controller on clusters that support the CRD
		if err := discovery.ServerSupportsVersion(discoverycli, machinev1.GroupVersion); err == nil {
			if err = (cpms.NewReconciler(
				log.WithField("controller", cpms.ControllerName),
				client)).SetupWithManager(mgr); err != nil {
				return fmt.Errorf("unable to create controller %s: %v", cpms.ControllerName, err)
			}
		} else {
			log.Infof("Cluster does not support machinev1 API, will not set up CPMS controller: %v", err)
		}
	}

	if err = (internetchecker.NewReconciler(
		log.WithField("controller", internetchecker.ControllerName),
		client, role)).SetupWithManager(mgr); err != nil {
		return fmt.Errorf("unable to create controller %s: %v", internetchecker.ControllerName, err)
	}

	for _, reconciler := range AROReconcilers {
		if err = reconciler.SetupWithManager(mgr); err != nil {
			return fmt.Errorf("unable to create controller %s: %v", reconciler.GetName(), err)
		}
	}

	// +kubebuilder:scaffold:builder

	log.Info("starting manager")

	if err := mgr.AddHealthzCheck("ready", healthz.Ping); err != nil {
		log.Error(err, "unable to set up health check")
		return err
	}
	if err := mgr.AddReadyzCheck("ready", healthz.Ping); err != nil {
		log.Error(err, "unable to set up ready check")
		return err
	}

	return mgr.Start(ctrl.SetupSignalHandler())
}
