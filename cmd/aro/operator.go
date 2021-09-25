package main

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"flag"
	"fmt"

	configclient "github.com/openshift/client-go/config/clientset/versioned"
	consoleclient "github.com/openshift/client-go/console/clientset/versioned"
	securityclient "github.com/openshift/client-go/security/clientset/versioned"
	maoclient "github.com/openshift/machine-api-operator/pkg/generated/clientset/versioned"
	mcoclient "github.com/openshift/machine-config-operator/pkg/generated/clientset/versioned"
	"github.com/sirupsen/logrus"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"

	"github.com/Azure/ARO-RP/pkg/env"
	pkgoperator "github.com/Azure/ARO-RP/pkg/operator"
	aroclient "github.com/Azure/ARO-RP/pkg/operator/clientset/versioned"
	"github.com/Azure/ARO-RP/pkg/operator/controllers"
	"github.com/Azure/ARO-RP/pkg/operator/controllers/alertwebhook"
	"github.com/Azure/ARO-RP/pkg/operator/controllers/banner"
	"github.com/Azure/ARO-RP/pkg/operator/controllers/checker"
	"github.com/Azure/ARO-RP/pkg/operator/controllers/clusteroperatoraro"
	"github.com/Azure/ARO-RP/pkg/operator/controllers/dnsmasq"
	"github.com/Azure/ARO-RP/pkg/operator/controllers/genevalogging"
	"github.com/Azure/ARO-RP/pkg/operator/controllers/imageconfig"
	"github.com/Azure/ARO-RP/pkg/operator/controllers/machine"
	"github.com/Azure/ARO-RP/pkg/operator/controllers/machineset"
	"github.com/Azure/ARO-RP/pkg/operator/controllers/monitoring"
	"github.com/Azure/ARO-RP/pkg/operator/controllers/node"
	"github.com/Azure/ARO-RP/pkg/operator/controllers/pullsecret"
	"github.com/Azure/ARO-RP/pkg/operator/controllers/rbac"
	"github.com/Azure/ARO-RP/pkg/operator/controllers/routefix"
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

	arocli, err := aroclient.NewForConfig(restConfig)
	if err != nil {
		return err
	}
	configcli, err := configclient.NewForConfig(restConfig)
	if err != nil {
		return err
	}
	consolecli, err := consoleclient.NewForConfig(restConfig)
	if err != nil {
		return err
	}
	kubernetescli, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return err
	}
	maocli, err := maoclient.NewForConfig(restConfig)
	if err != nil {
		return err
	}
	mcocli, err := mcoclient.NewForConfig(restConfig)
	if err != nil {
		return err
	}
	securitycli, err := securityclient.NewForConfig(restConfig)
	if err != nil {
		return err
	}
	// TODO (NE): dh is sometimes passed, sometimes created later. Can we standardize?
	dh, err := dynamichelper.New(log, restConfig)
	if err != nil {
		return err
	}

	if role == pkgoperator.RoleMaster {
		if err = (genevalogging.NewReconciler(
			log.WithField("controller", controllers.GenevaLoggingControllerName),
			arocli, kubernetescli, securitycli,
			restConfig)).SetupWithManager(mgr); err != nil {
			return fmt.Errorf("unable to create controller Genevalogging: %v", err)
		}
		if err = (clusteroperatoraro.NewReconciler(
			log.WithField("controller", controllers.ClusterOperatorAROName),
			arocli, configcli)).SetupWithManager(mgr); err != nil {
			return fmt.Errorf("unable to create controller ClusterOperatorARO: %v", err)
		}
		if err = (pullsecret.NewReconciler(
			log.WithField("controller", controllers.PullSecretControllerName),
			arocli, kubernetescli)).SetupWithManager(mgr); err != nil {
			return fmt.Errorf("unable to create controller PullSecret: %v", err)
		}
		if err = (alertwebhook.NewReconciler(
			log.WithField("controller", controllers.AlertwebhookControllerName),
			arocli, kubernetescli)).SetupWithManager(mgr); err != nil {
			return fmt.Errorf("unable to create controller AlertWebhook: %v", err)
		}
		if err = (workaround.NewReconciler(
			log.WithField("controller", controllers.WorkaroundControllerName),
			arocli, configcli, kubernetescli, mcocli, restConfig)).SetupWithManager(mgr); err != nil {
			return fmt.Errorf("unable to create controller Workaround: %v", err)
		}
		if err = (routefix.NewReconciler(
			log.WithField("controller", controllers.RouteFixControllerName),
			arocli, configcli, kubernetescli, securitycli, restConfig)).SetupWithManager(mgr); err != nil {
			return fmt.Errorf("unable to create controller RouteFix: %v", err)
		}
		if err = (monitoring.NewReconciler(
			log.WithField("controller", controllers.MonitoringControllerName),
			arocli, kubernetescli)).SetupWithManager(mgr); err != nil {
			return fmt.Errorf("unable to create controller Monitoring: %v", err)
		}
		if err = (rbac.NewReconciler(
			log.WithField("controller", controllers.RBACControllerName),
			arocli, dh)).SetupWithManager(mgr); err != nil {
			return fmt.Errorf("unable to create controller RBAC: %v", err)
		}
		if err = (dnsmasq.NewClusterReconciler(
			log.WithField("controller", controllers.DnsmasqClusterControllerName),
			arocli, mcocli, dh)).SetupWithManager(mgr); err != nil {
			return fmt.Errorf("unable to create controller DnsmasqCluster: %v", err)
		}
		if err = (dnsmasq.NewMachineConfigReconciler(
			log.WithField("controller", controllers.DnsmasqMachineConfigControllerName),
			arocli, mcocli, dh)).SetupWithManager(mgr); err != nil {
			return fmt.Errorf("unable to create controller DnsmasqMachineConfig: %v", err)
		}
		if err = (dnsmasq.NewMachineConfigPoolReconciler(
			log.WithField("controller", controllers.DnsmasqMachineConfigPoolControllerName),
			arocli, mcocli, dh)).SetupWithManager(mgr); err != nil {
			return fmt.Errorf("unable to create controller DnsmasqMachineConfigPool: %v", err)
		}
		if err = (node.NewReconciler(
			log.WithField("controller", controllers.NodeControllerName),
			arocli, kubernetescli)).SetupWithManager(mgr); err != nil {
			return fmt.Errorf("unable to create controller Node: %v", err)
		}
		if err = (subnets.NewReconciler(
			log.WithField("controller", controllers.AzureSubnetsControllerName),
			arocli, kubernetescli, maocli)).SetupWithManager(mgr); err != nil {
			return fmt.Errorf("unable to create controller Subnets: %v", err)
		}
		if err = (machine.NewReconciler(
			log.WithField("controller", controllers.MachineControllerName),
			arocli, maocli, isLocalDevelopmentMode, role)).SetupWithManager(mgr); err != nil {
			return fmt.Errorf("unable to create controller Machine: %v", err)
		}
		if err = (banner.NewReconciler(
			log.WithField("controller", controllers.BannerControllerName),
			arocli, consolecli)).SetupWithManager(mgr); err != nil {
			return fmt.Errorf("unable to create controller Banner: %v", err)
		}
		if err = (machineset.NewReconciler(
			log.WithField("controller", controllers.MachineSetControllerName),
			arocli, maocli)).SetupWithManager(mgr); err != nil {
			return fmt.Errorf("unable to create controller MachineSet: %v", err)
		}
		if err = (imageconfig.NewReconciler(
			log.WithField("controller", controllers.ImageConfigControllerName),
			arocli, configcli)).SetupWithManager(mgr); err != nil {
			return fmt.Errorf("unable to create controller ImageConfig: %v", err)
		}
	}

	if err = (checker.NewReconciler(
		log.WithField("controller", controllers.CheckerControllerName),
		arocli, kubernetescli, maocli, role)).SetupWithManager(mgr); err != nil {
		return fmt.Errorf("unable to create controller Checker: %v", err)
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
