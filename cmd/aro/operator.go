package main

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"flag"
	"fmt"

	securityclient "github.com/openshift/client-go/security/clientset/versioned"
	"github.com/sirupsen/logrus"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"

	pkgoperator "github.com/Azure/ARO-RP/pkg/operator"
	aroclient "github.com/Azure/ARO-RP/pkg/operator/clientset/versioned/typed/aro.openshift.io/v1alpha1"
	"github.com/Azure/ARO-RP/pkg/operator/controllers"
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

	ctrl.SetLogger(utillog.LogrWrapper(log))

	restConfig, err := ctrl.GetConfig()
	if err != nil {
		return err
	}

	mgr, err := ctrl.NewManager(restConfig, ctrl.Options{
		MetricsBindAddress: "0", // disabled
		Port:               8443,
	})
	if err != nil {
		return err
	}

	kubernetescli, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return err
	}
	securitycli, err := securityclient.NewForConfig(restConfig)
	if err != nil {
		return err
	}
	arocli, err := aroclient.NewForConfig(restConfig)
	if err != nil {
		return err
	}

	if role == pkgoperator.RoleMaster {
		if err = (controllers.NewGenevaloggingReconciler(
			log.WithField("controller", controllers.GenevaLoggingControllerName),
			kubernetescli, securitycli, arocli,
			restConfig)).SetupWithManager(mgr); err != nil {
			return fmt.Errorf("unable to create controller Genevalogging: %v", err)
		}
		if err = (controllers.NewPullSecretReconciler(
			log.WithField("controller", controllers.PullSecretControllerName),
			kubernetescli, arocli)).SetupWithManager(mgr); err != nil {
			return fmt.Errorf("unable to create controller PullSecret: %v", err)
		}
		if err = (controllers.NewAlertWebhookReconciler(
			log.WithField("controller", controllers.AlertwebhookControllerName),
			kubernetescli)).SetupWithManager(mgr); err != nil {
			return fmt.Errorf("unable to create controller AlertWebhook: %v", err)
		}
	}

	if err = (controllers.NewInternetChecker(
		log.WithField("controller", controllers.InternetCheckerControllerName),
		kubernetescli, arocli,
		role,
	)).SetupWithManager(mgr); err != nil {
		return fmt.Errorf("unable to create controller InternetChecker: %v", err)
	}
	// +kubebuilder:scaffold:builder

	log.Info("starting manager")
	return mgr.Start(ctrl.SetupSignalHandler())
}
