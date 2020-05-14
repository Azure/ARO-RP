package controllers

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"net/http"

	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	arov1alpha1 "github.com/Azure/ARO-RP/operator/api/v1alpha1"
)

// InternetChecker reconciles a Cluster object
type InternetChecker struct {
	client.Client
	Log    *logrus.Entry
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=aro.openshift.io,resources=clusters,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=aro.openshift.io,resources=clusters/status,verbs=get;update;patch

func (r *InternetChecker) Reconcile(request ctrl.Request) (ctrl.Result, error) {
	operatorNs, err := OperatorNamespace()
	if err != nil {
		r.Log.Error(err, "OperatorNamespace")
		return ReconcileResultError, err
	}

	if request.Name != arov1alpha1.SingletonClusterName || request.Namespace != operatorNs {
		return ReconcileResultIgnore, nil
	}
	r.Log.Info("Polling outgoing internet connection")

	// TODO https://github.com/Azure/OpenShift/issues/185

	req, err := http.NewRequest("GET", "https://management.azure.com", nil)
	if err != nil {
		r.Log.Error(err, "failed building request")
		return ReconcileResultError, err
	}
	req.Header.Set("Content-Type", "application/json")

	ctx := context.TODO()
	sr := NewStatusReporter(r.Client, request.Namespace, request.Name)
	client := &http.Client{}
	resp, err := client.Do(req)
	r.Log.Debugf("response code %s, err %s", resp.Status, err)
	if err != nil || resp.StatusCode != http.StatusOK {
		err = sr.SetNoInternetConnection(ctx, err)
	} else {
		err = sr.SetInternetConnected(ctx)
	}
	if err != nil {
		r.Log.Error(err, "StatusReporter")
		return ReconcileResultError, err
	}

	r.Log.Info("done, requeueing")
	return ReconcileResultRequeue, nil
}

func (r *InternetChecker) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&arov1alpha1.Cluster{}).
		Complete(r)
}
