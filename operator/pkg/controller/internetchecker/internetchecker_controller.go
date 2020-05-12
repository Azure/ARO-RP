package internetchecker

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"net/http"

	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	aro "github.com/Azure/ARO-RP/operator/pkg/apis/aro/v1alpha1"
	"github.com/Azure/ARO-RP/operator/pkg/controller/consts"
	"github.com/Azure/ARO-RP/operator/pkg/controller/deploy"
	"github.com/Azure/ARO-RP/operator/pkg/controller/statusreporter"
)

var (
	log = logf.Log.WithName("controller_internetchecker")
)

// Add creates a new Cluster Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &internetChecker{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	c, err := controller.New("internetchecker-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource Cluster
	return c.Watch(&source.Kind{Type: &aro.Cluster{}}, &handler.EnqueueRequestForObject{})
}

// blank assignment to verify that internetChecker implements reconcile.Reconciler
var _ reconcile.Reconciler = &internetChecker{}

// internetChecker reconciles a Secret object
type internetChecker struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a Cluster object and makes changes based on the state read
// and what is in the Cluster.Spec
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *internetChecker) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	operatorNs, err := deploy.OperatorNamespace()
	if err != nil {
		log.Error(err, "deploy.OperatorNamespace")
		return consts.ReconcileResultError, err
	}

	if request.Name != aro.SingletonClusterName || request.Namespace != operatorNs {
		return consts.ReconcileResultIgnore, nil
	}
	log.Info("Polling outgoing internet connection")

	// TODO https://github.com/Azure/OpenShift/issues/185

	req, err := http.NewRequest("GET", "https://redhat.com", nil)
	if err != nil {
		log.Error(err, "failed building request")
		return consts.ReconcileResultError, err
	}
	req.Header.Set("Content-Type", "application/json")

	ctx := context.TODO()
	sr := statusreporter.NewStatusReporter(r.client, request.Namespace, request.Name)
	client := &http.Client{}
	resp, err := client.Do(req)
	log.Info("response", "code", resp.Status, "err", err)
	if err != nil || resp.StatusCode != http.StatusOK {
		err = sr.SetNoInternetConnection(ctx, err)
	} else {
		err = sr.SetInternetConnected(ctx)
	}
	if err != nil {
		log.Error(err, "StatusReporter")
		return consts.ReconcileResultError, err
	}

	log.Info("done, requeueing")
	return consts.ReconcileResultRequeue, nil
}
