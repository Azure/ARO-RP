package controllers

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"github.com/ghodss/yaml"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/util/retry"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var alertManagerName = types.NamespacedName{Name: "alertmanager-main", Namespace: "openshift-monitoring"}

// AlertWebhookReconciler reconciles the alertmanager webhook
type AlertWebhookReconciler struct {
	Kubernetescli kubernetes.Interface
	Log           *logrus.Entry
	Scheme        *runtime.Scheme
}

// This is the permissions that this controller needs to work.
// "make generate" will run kubebuilder and cause operator/deploy/staticresources/*/role.yaml to be updated
// from the annotation below.
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch;update;patch;create

// Reconcile makes sure that the Alertmanager default webhook is set.
func (r *AlertWebhookReconciler) Reconcile(request ctrl.Request) (ctrl.Result, error) {
	if request.NamespacedName != alertManagerName {
		// filter out other secrets.
		return reconcile.Result{}, nil
	}

	// TODO run our own web server and use that address
	return reconcile.Result{}, r.setAlertManagerWebhook("http://localhost:1234/")
}

// setAlertManagerWebhook is a hack to disable the
// AlertmanagerReceiversNotConfigured warning added in 4.3.8.
func (r *AlertWebhookReconciler) setAlertManagerWebhook(addr string) error {
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		s, err := r.Kubernetescli.CoreV1().Secrets(alertManagerName.Namespace).Get(alertManagerName.Name, metav1.GetOptions{})
		if err != nil {
			return err
		}

		var am map[string]interface{}
		err = yaml.Unmarshal(s.Data["alertmanager.yaml"], &am)
		if err != nil {
			return err
		}

		for _, r := range am["receivers"].([]interface{}) {
			r := r.(map[string]interface{})
			if name, ok := r["name"]; !ok || name != "null" {
				continue
			}

			r["webhook_configs"] = []interface{}{
				map[string]interface{}{"url": addr},
			}
		}

		s.Data["alertmanager.yaml"], err = yaml.Marshal(am)
		if err != nil {
			return err
		}

		_, err = r.Kubernetescli.CoreV1().Secrets(alertManagerName.Namespace).Update(s)
		return err
	})
}

func alertwebhookRelatedObjects() []corev1.ObjectReference {
	return []corev1.ObjectReference{
		{Kind: "Secret", Name: alertManagerName.Name, Namespace: alertManagerName.Namespace},
	}
}

func triggerAlertReconcile(secret *corev1.Secret) bool {
	return secret.Name == alertManagerName.Name && secret.Namespace == alertManagerName.Namespace
}

// SetupWithManager setup our mananger
func (r *AlertWebhookReconciler) SetupWithManager(mgr ctrl.Manager) error {
	isAlertManager := predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			oldSecret, ok := e.ObjectOld.(*corev1.Secret)
			if !ok {
				return false
			}
			newSecret, ok := e.ObjectNew.(*corev1.Secret)
			if !ok {
				return false
			}
			return (triggerAlertReconcile(oldSecret) || triggerAlertReconcile(newSecret))
		},
		CreateFunc: func(e event.CreateEvent) bool {
			secret, ok := e.Object.(*corev1.Secret)
			if !ok {
				return false
			}
			return triggerAlertReconcile(secret)
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			secret, ok := e.Object.(*corev1.Secret)
			if !ok {
				return false
			}
			return triggerAlertReconcile(secret)
		},
	}
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1.Secret{}).WithEventFilter(isAlertManager).Named(AlertwebhookControllerName).
		Complete(r)
}
