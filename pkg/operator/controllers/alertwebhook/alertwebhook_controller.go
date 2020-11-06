package alertwebhook

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"net"
	"net/http"
	"reflect"

	"github.com/ghodss/yaml"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/util/retry"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/Azure/ARO-RP/pkg/operator/controllers"
)

var alertManagerName = types.NamespacedName{Name: "alertmanager-main", Namespace: "openshift-monitoring"}

// AlertWebhookReconciler reconciles the alertmanager webhook
type AlertWebhookReconciler struct {
	kubernetescli kubernetes.Interface
	log           *logrus.Entry
}

func NewReconciler(log *logrus.Entry, kubernetescli kubernetes.Interface) *AlertWebhookReconciler {
	return &AlertWebhookReconciler{
		kubernetescli: kubernetescli,
		log:           log,
	}
}

// Reconcile makes sure that the Alertmanager default webhook is set.
func (r *AlertWebhookReconciler) Reconcile(request ctrl.Request) (ctrl.Result, error) {
	// TODO(mj): controller-runtime master fixes the need for this (https://github.com/kubernetes-sigs/controller-runtime/blob/master/pkg/reconcile/reconcile.go#L93) but it's not yet released.
	ctx := context.Background()
	if request.NamespacedName != alertManagerName {
		return reconcile.Result{}, nil
	}

	return reconcile.Result{}, r.setAlertManagerWebhook(ctx, "http://aro-operator-master.openshift-azure-operator.svc.cluster.local:8080")
}

// setAlertManagerWebhook is a hack to disable the
// AlertmanagerReceiversNotConfigured warning added in 4.3.8.
func (r *AlertWebhookReconciler) setAlertManagerWebhook(ctx context.Context, addr string) error {
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
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

			if name, ok := r["name"].(string); !ok || name != "null" {
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
	})
}

func triggerAlertReconcile(secret *corev1.Secret) bool {
	return secret.Name == alertManagerName.Name && secret.Namespace == alertManagerName.Namespace
}

func aroserverRun() error {
	l, err := net.Listen("tcp", ":8080")
	if err != nil {
		return err
	}

	go http.Serve(l, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))

	return nil
}

// SetupWithManager setup our mananger
func (r *AlertWebhookReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.log.Info("starting alertmanager sink")

	err := aroserverRun()
	if err != nil {
		return err
	}

	isAlertManager := predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			secret, ok := e.ObjectOld.(*corev1.Secret)
			return ok && triggerAlertReconcile(secret)
		},
		CreateFunc: func(e event.CreateEvent) bool {
			secret, ok := e.Object.(*corev1.Secret)
			return ok && triggerAlertReconcile(secret)
		},
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&v1.Secret{}).
		WithEventFilter(isAlertManager).
		Named(controllers.AlertwebhookControllerName).
		Complete(r)
}
