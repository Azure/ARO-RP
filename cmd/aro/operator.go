package main

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"flag"
	"os"
	"regexp"

	securityclient "github.com/openshift/client-go/security/clientset/versioned"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"

	aro "github.com/Azure/ARO-RP/operator/apis/aro.openshift.io/v1alpha1"
	"github.com/Azure/ARO-RP/pkg/controllers"
	aroclient "github.com/Azure/ARO-RP/pkg/util/aro-operator-client/clientset/versioned/typed/aro.openshift.io/v1alpha1"
	utillog "github.com/Azure/ARO-RP/pkg/util/log"
	// +kubebuilder:scaffold:imports
)

var (
	scheme = runtime.NewScheme()
)

func init() {
	_ = clientgoscheme.AddToScheme(scheme)

	_ = aro.AddToScheme(scheme)
	// +kubebuilder:scaffold:scheme
}

func operator(ctx context.Context, log *logrus.Entry) error {
	var metricsAddr string
	var enableLeaderElection bool
	flag.StringVar(&metricsAddr, "metrics-addr", ":8080", "The address the metric endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "enable-leader-election", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	flag.Parse()

	ctrl.SetLogger(utillog.GetRLogger(log))

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:             scheme,
		MetricsBindAddress: metricsAddr,
		Port:               9443,
		LeaderElection:     enableLeaderElection,
		LeaderElectionID:   "965fa11c.openshift.io",
	})
	if err != nil {
		log.Errorf("unable to start manager %v", err)
		return err
	}
	restConfig, err := ctrl.GetConfig()
	if err != nil {
		log.Errorf("unable to get rest config %v", err)
		return err
	}

	kubernetescli, err := kubernetes.NewForConfig(mgr.GetConfig())
	if err != nil {
		log.Errorf("unable to create clients %v", err)
		return err
	}
	securitycli, err := securityclient.NewForConfig(mgr.GetConfig())
	if err != nil {
		log.Errorf("unable to create clients %v", err)
		return err
	}
	arocli, err := aroclient.NewForConfig(mgr.GetConfig())
	if err != nil {
		log.Errorf("unable to create clients %v", err)
		return err
	}

	placement, err := getPlacement(log)
	if placement == "master" {
		if err = (&controllers.GenevaloggingReconciler{
			Kubernetescli: kubernetescli,
			Securitycli:   securitycli,
			AROCli:        arocli,
			RestConfig:    restConfig,
			Log:           log.WithField("controller", controllers.GenevaLoggingControllerName),
			Scheme:        mgr.GetScheme(),
		}).SetupWithManager(mgr); err != nil {
			log.Errorf("unable to create controller: Genevalogging %v", err)
			return err
		}
		if err = (&controllers.PullsecretReconciler{
			Kubernetescli: kubernetescli,
			AROCli:        arocli,
			Log:           log.WithField("controller", controllers.PullSecretControllerName),
			Scheme:        mgr.GetScheme(),
		}).SetupWithManager(mgr); err != nil {
			log.Errorf("unable to create controller: PullSecret %v", err)
			return err
		}
		if err = (&controllers.CPValidator{
			Kubernetescli: kubernetescli,
			AROCli:        arocli,
			Log:           log.WithField("controller", controllers.CPValidatorControllerName),
			Scheme:        mgr.GetScheme(),
		}).SetupWithManager(mgr); err != nil {
			log.Errorf("unable to create controller: CPValidator %v", err)
			return err
		}
		if err = (&controllers.AlertWebhookReconciler{
			Kubernetescli: kubernetescli,
			Log:           log.WithField("controller", controllers.AlertwebhookControllerName),
			Scheme:        mgr.GetScheme(),
		}).SetupWithManager(mgr); err != nil {
			log.Errorf("unable to create controller: AlertWebhook %v", err)
			return err
		}
	}

	if err = (&controllers.InternetChecker{
		Kubernetescli: kubernetescli,
		AROCli:        arocli,
		Log:           log.WithField("controller", controllers.InternetCheckerControllerName),
		Scheme:        mgr.GetScheme(),
		Placement:     placement,
	}).SetupWithManager(mgr); err != nil {
		log.Errorf("unable to create controller: InternetChecker %v", err)
		return err
	}
	// +kubebuilder:scaffold:builder

	log.Info("starting manager")
	return mgr.Start(ctrl.SetupSignalHandler())
}

// Parses `NODE_NAME` environment variable to retrieve pod's placement - master or worker
func getPlacement(log *logrus.Entry) (string, error) {
	nodeName, found := os.LookupEnv("NODE_NAME")
	if !found {
		if os.Getenv("RP_MODE") == "development" {
			return "master", nil
		}
		log.Errorf("environment variable NODE_NAME is unset, exiting")
		os.Exit(2)
	}

	isMaster, err := regexp.MatchString(".*-master-[0-2]{1}$", nodeName)
	if err != nil {
		log.Errorf("unable to parse node name %v", err)
		return "", err
	}
	if isMaster {
		return "master", nil
	}

	isWorker, err := regexp.MatchString(".*-worker-[a-z0-9]+-[a-z0-9]+$", nodeName)
	if err != nil || !isWorker {
		log.Errorf("unable to parse node name or unexpected node name value %v", err)
		return "", err
	}
	return "worker", nil
}
