package controllers

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	aro "github.com/Azure/ARO-RP/operator/apis/aro.openshift.io/v1alpha1"
	aroclient "github.com/Azure/ARO-RP/pkg/util/aro-operator-client/clientset/versioned/typed/aro.openshift.io/v1alpha1"
)

var sites = []string{
	"https://registry.redhat.io",
	"https://quay.io",
	"https://sso.redhat.com",
	"https://mirror.openshift.com",
	"https://api.openshift.com",
}

// InternetChecker reconciles a Cluster object
type InternetChecker struct {
	Kubernetescli kubernetes.Interface
	AROCli        aroclient.AroV1alpha1Interface
	Log           *logrus.Entry
	Scheme        *runtime.Scheme
	testurls      []string
	sr            *StatusReporter
}

// SimpleHTTPClient to aid in mocking
type SimpleHTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// This is the permissions that this controller needs to work.
// "make generate" will run kubebuilder and cause operator/deploy/staticresources/role.yaml to be updated
// from the annotation below.
// +kubebuilder:rbac:groups=aro.openshift.io,resources=clusters,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=aro.openshift.io,resources=clusters/status,verbs=get;update;patch

// Reconcile will keep checking that the cluster can connect to essential services.
func (r *InternetChecker) Reconcile(request ctrl.Request) (ctrl.Result, error) {
	if request.Name != aro.SingletonClusterName {
		return reconcile.Result{}, nil
	}
	ctx := context.TODO()
	if r.testurls == nil {
		instance, err := r.AROCli.Clusters().Get(request.Name, v1.GetOptions{})
		if err != nil {
			return reconcile.Result{}, err
		}
		if instance.Spec.ResourceID == "" {
			return ReconcileResultRequeueShort, nil
		}
		r.testurls = sites
		r.testurls = append(r.testurls, "https://management.azure.com"+instance.Spec.ResourceID+"?api-version=2020-04-30")
	}
	r.Log.Info("Polling outgoing internet connection")

	sitesNotAvailable := map[string]string{}
	for _, testurl := range r.testurls {
		checkErr := r.check(&http.Client{}, testurl)
		if checkErr != nil {
			sitesNotAvailable[testurl] = checkErr.Error()
		}
	}

	var err error
	if len(sitesNotAvailable) > 0 {
		err = r.sr.SetNoInternetConnection(ctx, sitesNotAvailable)
	} else {
		err = r.sr.SetInternetConnected(ctx)
	}
	if err != nil {
		r.Log.Errorf("StatusReporter request:%v err:%v", request, err)
		return reconcile.Result{}, err
	}

	r.Log.Info("done, requeueing")
	return ReconcileResultRequeueShort, nil
}

func (r *InternetChecker) check(client SimpleHTTPClient, testurl string) error {
	ctx := context.TODO()
	timeoutCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	req, err := http.NewRequest("GET", testurl, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req.WithContext(timeoutCtx))
	if err != nil {
		return err
	}
	if resp.StatusCode >= http.StatusInternalServerError {
		defer resp.Body.Close()
		b, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		r.Log.Warnf("check failed (%s) status:%s body:%s", testurl, resp.Status, string(b))
		return fmt.Errorf("check failed %s bad status:%s", testurl, resp.Status)
	}
	return nil
}

// SetupWithManager setup our mananger
func (r *InternetChecker) SetupWithManager(mgr ctrl.Manager) error {
	r.sr = NewStatusReporter(r.Log, r.AROCli, aro.SingletonClusterName)

	return ctrl.NewControllerManagedBy(mgr).
		For(&aro.Cluster{}).
		Complete(r)
}
