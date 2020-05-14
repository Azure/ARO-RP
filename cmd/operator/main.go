package main

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"flag"
	"os"

	securityclient "github.com/openshift/client-go/security/clientset/versioned"
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
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithValues("controller", "setup")
)

func init() {
	_ = clientgoscheme.AddToScheme(scheme)

	_ = aro.AddToScheme(scheme)
	// +kubebuilder:scaffold:scheme
}

func main() {
	var metricsAddr string
	var enableLeaderElection bool
	flag.StringVar(&metricsAddr, "metrics-addr", ":8080", "The address the metric endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "enable-leader-election", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	flag.Parse()

	log := utillog.GetLogger()

	ctrl.SetLogger(utillog.GetRLogger(log))

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:             scheme,
		MetricsBindAddress: metricsAddr,
		Port:               9443,
		LeaderElection:     enableLeaderElection,
		LeaderElectionID:   "965fa11c.openshift.io",
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	kubernetescli, err := kubernetes.NewForConfig(mgr.GetConfig())
	if err != nil {
		setupLog.Error(err, "unable to create clients")
		os.Exit(1)
	}
	securitycli, err := securityclient.NewForConfig(mgr.GetConfig())
	if err != nil {
		setupLog.Error(err, "unable to create clients")
		os.Exit(1)
	}
	arocli, err := aroclient.NewForConfig(mgr.GetConfig())
	if err != nil {
		setupLog.Error(err, "unable to create clients")
		os.Exit(1)
	}

	if err = (&controllers.GenevaloggingReconciler{
		Kubernetescli: kubernetescli,
		Securitycli:   securitycli,
		AROCli:        arocli,
		Log:           log.WithField("controller", "Genevalogging"),
		Scheme:        mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Genevalogging")
		os.Exit(1)
	}
	if err = (&controllers.PullsecretReconciler{
		Kubernetescli: kubernetescli,
		AROCli:        arocli,
		Log:           log.WithField("controller", "PullSecret"),
		Scheme:        mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Pullsecret")
		os.Exit(1)
	}
	if err = (&controllers.InternetChecker{
		Kubernetescli: kubernetescli,
		AROCli:        arocli,
		Log:           log.WithField("controller", "InternetChecker"),
		Scheme:        mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "InternetChecker")
		os.Exit(1)
	}
	// +kubebuilder:scaffold:builder

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}
