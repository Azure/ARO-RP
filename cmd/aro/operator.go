package main

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"flag"
	"fmt"

	configclient "github.com/openshift/client-go/config/clientset/versioned"
	consoleclient "github.com/openshift/client-go/console/clientset/versioned"
	imageregistryclient "github.com/openshift/client-go/imageregistry/clientset/versioned"
	machineclient "github.com/openshift/client-go/machine/clientset/versioned"
	operatorclient "github.com/openshift/client-go/operator/clientset/versioned"
	securityclient "github.com/openshift/client-go/security/clientset/versioned"
	mcoclient "github.com/openshift/machine-config-operator/pkg/generated/clientset/versioned"
	"github.com/sirupsen/logrus"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"

	"github.com/Azure/ARO-RP/pkg/env"
	pkgoperator "github.com/Azure/ARO-RP/pkg/operator"
	aroclient "github.com/Azure/ARO-RP/pkg/operator/clientset/versioned"
	"github.com/Azure/ARO-RP/pkg/operator/controllers/alertwebhook"
	"github.com/Azure/ARO-RP/pkg/operator/controllers/autosizednodes"
	"github.com/Azure/ARO-RP/pkg/operator/controllers/banner"
	"github.com/Azure/ARO-RP/pkg/operator/controllers/checkers/clusterdnschecker"
	"github.com/Azure/ARO-RP/pkg/operator/controllers/checkers/ingresscertificatechecker"
	"github.com/Azure/ARO-RP/pkg/operator/controllers/checkers/internetchecker"
	"github.com/Azure/ARO-RP/pkg/operator/controllers/checkers/serviceprincipalchecker"
	"github.com/Azure/ARO-RP/pkg/operator/controllers/clusteroperatoraro"
	"github.com/Azure/ARO-RP/pkg/operator/controllers/dnsmasq"
	"github.com/Azure/ARO-RP/pkg/operator/controllers/genevalogging"
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
	maocli, err := machineclient.NewForConfig(restConfig)
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
	imageregistrycli, err := imageregistryclient.NewForConfig(restConfig)
	if err != nil {
		return err
	}
	operatorcli, err := operatorclient.NewForConfig(restConfig)
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
			log.WithField("controller", genevalogging.ControllerName),
			kubernetescli, securitycli, restConfig)).SetupWithManager(mgr); err != nil {
			return fmt.Errorf("unable to create controller %s: %v", genevalogging.ControllerName, err)
		}
		if err = (clusteroperatoraro.NewReconciler(
			log.WithField("controller", clusteroperatoraro.ControllerName),
			configcli)).SetupWithManager(mgr); err != nil {
			return fmt.Errorf("unable to create controller %s: %v", clusteroperatoraro.ControllerName, err)
		}
		if err = (pullsecret.NewReconciler(
			log.WithField("controller", pullsecret.ControllerName),
			kubernetescli)).SetupWithManager(mgr); err != nil {
			return fmt.Errorf("unable to create controller %s: %v", pullsecret.ControllerName, err)
		}
		if err = (alertwebhook.NewReconciler(
			log.WithField("controller", alertwebhook.ControllerName), kubernetescli)).SetupWithManager(mgr); err != nil {
			return fmt.Errorf("unable to create controller %s: %v", alertwebhook.ControllerName, err)
		}
		if err = (workaround.NewReconciler(
			log.WithField("controller", workaround.ControllerName),
			configcli, kubernetescli, mcocli, restConfig)).SetupWithManager(mgr); err != nil {
			return fmt.Errorf("unable to create controller %s: %v", workaround.ControllerName, err)
		}
		if err = (routefix.NewReconciler(
			log.WithField("controller", routefix.ControllerName),
			configcli, kubernetescli, securitycli, restConfig)).SetupWithManager(mgr); err != nil {
			return fmt.Errorf("unable to create controller %s: %v", routefix.ControllerName, err)
		}
		if err = (monitoring.NewReconciler(
			log.WithField("controller", monitoring.ControllerName),
			kubernetescli)).SetupWithManager(mgr); err != nil {
			return fmt.Errorf("unable to create controller %s: %v", monitoring.ControllerName, err)
		}
		if err = (rbac.NewReconciler(
			log.WithField("controller", rbac.ControllerName), dh)).SetupWithManager(mgr); err != nil {
			return fmt.Errorf("unable to create controller %s: %v", rbac.ControllerName, err)
		}
		if err = (dnsmasq.NewClusterReconciler(
			log.WithField("controller", dnsmasq.ClusterControllerName),
			mcocli, dh)).SetupWithManager(mgr); err != nil {
			return fmt.Errorf("unable to create controller %s: %v", dnsmasq.ClusterControllerName, err)
		}
		if err = (dnsmasq.NewMachineConfigReconciler(
			log.WithField("controller", dnsmasq.MachineConfigControllerName),
			mcocli, dh)).SetupWithManager(mgr); err != nil {
			return fmt.Errorf("unable to create controller %s: %v", dnsmasq.MachineConfigControllerName, err)
		}
		if err = (dnsmasq.NewMachineConfigPoolReconciler(
			log.WithField("controller", dnsmasq.MachineConfigPoolControllerName),
			mcocli, dh)).SetupWithManager(mgr); err != nil {
			return fmt.Errorf("unable to create controller %s: %v", dnsmasq.MachineConfigPoolControllerName, err)
		}
		if err = (node.NewReconciler(
			log.WithField("controller", node.ControllerName), kubernetescli)).SetupWithManager(mgr); err != nil {
			return fmt.Errorf("unable to create controller %s: %v", node.ControllerName, err)
		}
		if err = (subnets.NewReconciler(
			log.WithField("controller", subnets.ControllerName),
			kubernetescli, maocli)).SetupWithManager(mgr); err != nil {
			return fmt.Errorf("unable to create controller %s: %v", subnets.ControllerName, err)
		}
		if err = (machine.NewReconciler(
			log.WithField("controller", machine.ControllerName),
			arocli, maocli, isLocalDevelopmentMode, role)).SetupWithManager(mgr); err != nil {
			return fmt.Errorf("unable to create controller %s: %v", machine.ControllerName, err)
		}
		if err = (banner.NewReconciler(
			log.WithField("controller", banner.ControllerName), consolecli)).SetupWithManager(mgr); err != nil {
			return fmt.Errorf("unable to create controller %s: %v", banner.ControllerName, err)
		}
		if err = (machineset.NewReconciler(
			log.WithField("controller", machineset.ControllerName),
			maocli)).SetupWithManager(mgr); err != nil {
			return fmt.Errorf("unable to create controller %s: %v", machineset.ControllerName, err)
		}
		if err = (imageconfig.NewReconciler(
			log.WithField("controller", imageconfig.ControllerName),
			configcli)).SetupWithManager(mgr); err != nil {
			return fmt.Errorf("unable to create controller %s: %v", imageconfig.ControllerName, err)
		}
		if err = (previewfeature.NewReconciler(
			log.WithField("controller", previewfeature.ControllerName),
			kubernetescli, maocli)).SetupWithManager(mgr); err != nil {
			return fmt.Errorf("unable to create controller %s: %v", previewfeature.ControllerName, err)
		}
		if err = (storageaccounts.NewReconciler(
			log.WithField("controller", storageaccounts.ControllerName),
			maocli, kubernetescli, imageregistrycli)).SetupWithManager(mgr); err != nil {
			return fmt.Errorf("unable to create controller %s: %v", storageaccounts.ControllerName, err)
		}
		if err = (muo.NewReconciler(
			log.WithField("controller", muo.ControllerName),
			kubernetescli, dh)).SetupWithManager(mgr); err != nil {
			return fmt.Errorf("unable to create controller %s: %v", muo.ControllerName, err)
		}
		if err = (autosizednodes.NewReconciler(
			log.WithField("controller", autosizednodes.ControllerName),
			mgr)).SetupWithManager(mgr); err != nil {
			return fmt.Errorf("unable to create controller %s: %v", autosizednodes.ControllerName, err)
		}
		if err = (machinehealthcheck.NewReconciler(
			log.WithField("controller", machinehealthcheck.ControllerName),
			dh)).SetupWithManager(mgr); err != nil {
			return fmt.Errorf("unable to create controller %s: %v", machinehealthcheck.ControllerName, err)
		}
		if err = (ingress.NewReconciler(
			log.WithField("controller", ingress.ControllerName),
			operatorcli)).SetupWithManager(mgr); err != nil {
			return fmt.Errorf("unable to create controller %s: %v", ingress.ControllerName, err)
		}
		if err = (serviceprincipalchecker.NewReconciler(
			log.WithField("controller", serviceprincipalchecker.ControllerName),
			arocli, kubernetescli, role)).SetupWithManager(mgr); err != nil {
			return fmt.Errorf("unable to create controller %s: %v", serviceprincipalchecker.ControllerName, err)
		}
		if err = (clusterdnschecker.NewReconciler(
			log.WithField("controller", clusterdnschecker.ControllerName),
			arocli, operatorcli, role)).SetupWithManager(mgr); err != nil {
			return fmt.Errorf("unable to create controller %s: %v", clusterdnschecker.ControllerName, err)
		}
		if err = (ingresscertificatechecker.NewReconciler(
			log.WithField("controller", ingresscertificatechecker.ControllerName),
			arocli, operatorcli, configcli, role)).SetupWithManager(mgr); err != nil {
			return fmt.Errorf("unable to create controller %s: %v", ingresscertificatechecker.ControllerName, err)
		}
	}

	if err = (internetchecker.NewReconciler(
		log.WithField("controller", internetchecker.ControllerName),
		arocli, role)).SetupWithManager(mgr); err != nil {
		return fmt.Errorf("unable to create controller %s: %v", internetchecker.ControllerName, err)
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
