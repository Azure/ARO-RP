package controllers

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"io/ioutil"
	"net/http"

	"github.com/sirupsen/logrus"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	aro "github.com/Azure/ARO-RP/operator/apis/aro.openshift.io/v1alpha1"
	aroclient "github.com/Azure/ARO-RP/pkg/util/aro-operator-client/clientset/versioned/typed/aro.openshift.io/v1alpha1"
)

// InternetChecker reconciles a Cluster object
type InternetChecker struct {
	Kubernetescli kubernetes.Interface
	AROCli        aroclient.AroV1alpha1Interface
	Log           *logrus.Entry
	Scheme        *runtime.Scheme
	testurl       string
}

// +kubebuilder:rbac:groups=aro.openshift.io,resources=clusters,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=aro.openshift.io,resources=clusters/status,verbs=get;update;patch

// TODO https://github.com/Azure/OpenShift/issues/185
func (r *InternetChecker) Reconcile(request ctrl.Request) (ctrl.Result, error) {
	if request.Name != aro.SingletonClusterName {
		return reconcile.Result{}, nil
	}
	ctx := context.TODO()
	if r.testurl == "" {
		instance, err := r.AROCli.Clusters().Get(request.Name, v1.GetOptions{})
		if err != nil {
			return reconcile.Result{}, err
		}
		if instance.Spec.ResourceID == "" {
			return ReconcileResultRequeue, nil
		}
		r.testurl = "https://management.azure.com" + instance.Spec.ResourceID + "?api-version=2020-04-30"
	}
	r.Log.Info("Polling outgoing internet connection")

	req, err := http.NewRequest("GET", r.testurl, nil)
	if err != nil {
		r.Log.Error(err, "failed building request")
		return reconcile.Result{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{}
	resp, err := client.Do(req)

	sr := NewStatusReporter(r.Log, r.AROCli, request.Name)
	// this is not ideal, but we can at least see that the site is working
	// if it is returning Unauthorized.
	if err != nil || resp.StatusCode != http.StatusUnauthorized {
		defer resp.Body.Close()
		b, err := ioutil.ReadAll(resp.Body)
		r.Log.Warn(string(b))
		err = sr.SetNoInternetConnection(ctx, err)
	} else {
		err = sr.SetInternetConnected(ctx)
	}
	if err != nil {
		r.Log.Errorf("StatusReporter request:%v err:%v", request, err)
		return reconcile.Result{}, err
	}

	r.Log.Info("done, requeueing")
	return ReconcileResultRequeue, nil
}

func (r *InternetChecker) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&aro.Cluster{}).
		Complete(r)
}
