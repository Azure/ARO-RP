package alertwebhook

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"reflect"

	"github.com/ghodss/yaml"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
)

const (
	ControllerName = "Alertwebhook"

	controllerEnabled = "aro.alertwebhook.enabled"
)

var alertManagerName = types.NamespacedName{Name: "alertmanager-main", Namespace: "openshift-monitoring"}

// Reconciler reconciles the alertmanager webhook
type Reconciler struct {
	log *logrus.Entry

	client client.Client
}

func NewReconciler(log *logrus.Entry, client client.Client) *Reconciler {
	return &Reconciler{
		log:    log,
		client: client,
	}
}

// Reconcile makes sure that the Alertmanager default webhook is set.
func (r *Reconciler) Reconcile(ctx context.Context, request ctrl.Request) (ctrl.Result, error) {
	instance := &arov1alpha1.Cluster{}
	err := r.client.Get(ctx, types.NamespacedName{Name: arov1alpha1.SingletonClusterName}, instance)
	if err != nil {
		return reconcile.Result{}, err
	}

	if !instance.Spec.OperatorFlags.GetSimpleBoolean(controllerEnabled) {
		r.log.Debug("controller is disabled")
		return reconcile.Result{}, nil
	}

	r.log.Debug("running")
	return reconcile.Result{}, r.setAlertManagerWebhook(ctx, "http://aro-operator-master.openshift-azure-operator.svc.cluster.local:8080/healthz/ready")
}

// setAlertManagerWebhook is a hack to disable the
// AlertmanagerReceiversNotConfigured warning added in 4.3.8.
func (r *Reconciler) setAlertManagerWebhook(ctx context.Context, addr string) error {
	s := &corev1.Secret{}
	err := r.client.Get(ctx, alertManagerName, s)
	if err != nil {
		return err
	}

	var am map[string]interface{}
	err = yaml.Unmarshal(s.Data["alertmanager.yaml"], &am)
	if err != nil {
		return err
	}

	receivers, ok := am["receivers"].([]interface{})
	if !ok {
		return nil
	}

	var changed bool
	for _, r := range receivers {
		r, ok := r.(map[string]interface{})
		if !ok {
			continue
		}

		if name, ok := r["name"].(string); !ok || (name != "null" && name != "Default") {
			continue
		}

		webhookConfigs := []interface{}{
			map[string]interface{}{"url": addr},
		}

		if !reflect.DeepEqual(r["webhook_configs"], webhookConfigs) {
			r["webhook_configs"] = webhookConfigs
			changed = true
		}
	}

	if !changed {
		return nil
	}

	s.Data["alertmanager.yaml"], err = yaml.Marshal(am)
	if err != nil {
		return err
	}

	return r.client.Update(ctx, s)
}

// SetupWithManager setup our manager
func (r *Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	isAlertManagerPredicate := predicate.NewPredicateFuncs(func(o client.Object) bool {
		return o.GetName() == alertManagerName.Name && o.GetNamespace() == alertManagerName.Namespace
	})

	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Secret{}, builder.WithPredicates(isAlertManagerPredicate)).
		Named(ControllerName).
		Complete(r)
}
