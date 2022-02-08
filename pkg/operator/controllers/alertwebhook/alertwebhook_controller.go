package alertwebhook

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"reflect"

	"github.com/ghodss/yaml"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	aroclient "github.com/Azure/ARO-RP/pkg/operator/clientset/versioned"
	"github.com/Azure/ARO-RP/pkg/operator/controllers"
)

const (
	CONFIG_NAMESPACE string = "aro.alertwebhook"
	ENABLED          string = CONFIG_NAMESPACE + ".enabled"
)

var alertManagerName = types.NamespacedName{Name: "alertmanager-main", Namespace: "openshift-monitoring"}

// Reconciler reconciles the alertmanager webhook
type Reconciler struct {
	log *logrus.Entry

	arocli        aroclient.Interface
	kubernetescli kubernetes.Interface
}

func NewReconciler(log *logrus.Entry, arocli aroclient.Interface, kubernetescli kubernetes.Interface) *Reconciler {
	return &Reconciler{
		log:           log,
		arocli:        arocli,
		kubernetescli: kubernetescli,
	}
}

// Reconcile makes sure that the Alertmanager default webhook is set.
func (r *Reconciler) Reconcile(ctx context.Context, request ctrl.Request) (ctrl.Result, error) {
	instance, err := r.arocli.AroV1alpha1().Clusters().Get(ctx, arov1alpha1.SingletonClusterName, metav1.GetOptions{})
	if err != nil {
		return reconcile.Result{}, err
	}

	if !instance.Spec.OperatorFlags.GetSimpleBoolean(ENABLED) {
		// controller is disabled
		return reconcile.Result{}, nil
	}

	return reconcile.Result{}, r.setAlertManagerWebhook(ctx, "http://aro-operator-master.openshift-azure-operator.svc.cluster.local:8080/healthz/ready")
}

// setAlertManagerWebhook is a hack to disable the
// AlertmanagerReceiversNotConfigured warning added in 4.3.8.
func (r *Reconciler) setAlertManagerWebhook(ctx context.Context, addr string) error {
	s, err := r.kubernetescli.CoreV1().Secrets(alertManagerName.Namespace).Get(ctx, alertManagerName.Name, metav1.GetOptions{})
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

	_, err = r.kubernetescli.CoreV1().Secrets(alertManagerName.Namespace).Update(ctx, s, metav1.UpdateOptions{})
	return err
}

// SetupWithManager setup our manager
func (r *Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.log.Info("starting alertmanager sink")

	isAlertManagerPredicate := predicate.NewPredicateFuncs(func(o client.Object) bool {
		return o.GetName() == alertManagerName.Name && o.GetNamespace() == alertManagerName.Namespace
	})

	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Secret{}, builder.WithPredicates(isAlertManagerPredicate)).
		Named(controllers.AlertwebhookControllerName).
		Complete(r)
}
