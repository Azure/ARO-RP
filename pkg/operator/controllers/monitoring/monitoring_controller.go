package monitoring

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/json"

	"github.com/ghodss/yaml"
	"github.com/sirupsen/logrus"
	"github.com/ugorji/go/codec"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"github.com/Azure/ARO-RP/pkg/api"
	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	aroclient "github.com/Azure/ARO-RP/pkg/operator/clientset/versioned"
	"github.com/Azure/ARO-RP/pkg/operator/controllers"
)

const (
	CONFIG_NAMESPACE string = "aro.monitoring"
	ENABLED          string = CONFIG_NAMESPACE + ".enabled"
)

var (
	monitoringName   = types.NamespacedName{Name: "cluster-monitoring-config", Namespace: "openshift-monitoring"}
	prometheusLabels = "app=prometheus,prometheus=k8s"
)

// Config represents cluster monitoring stack configuration.
// Reconciler reconciles retention and storage settings,
// MissingFields are used to preserve settings configured by user.
type Config struct {
	api.MissingFields
	PrometheusK8s struct {
		api.MissingFields
		Retention           string           `json:"retention,omitempty"`
		VolumeClaimTemplate *json.RawMessage `json:"volumeClaimTemplate,omitempty"`
	} `json:"prometheusK8s,omitempty"`
	AlertManagerMain struct {
		api.MissingFields
		VolumeClaimTemplate *json.RawMessage `json:"volumeClaimTemplate,omitempty"`
	} `json:"alertmanagerMain,omitempty"`
}

type Reconciler struct {
	log *logrus.Entry

	arocli        aroclient.Interface
	kubernetescli kubernetes.Interface

	jsonHandle *codec.JsonHandle
}

func NewReconciler(log *logrus.Entry, arocli aroclient.Interface, kubernetescli kubernetes.Interface) *Reconciler {
	return &Reconciler{
		arocli:        arocli,
		kubernetescli: kubernetescli,
		log:           log,
		jsonHandle:    new(codec.JsonHandle),
	}
}

func (r *Reconciler) Reconcile(ctx context.Context, request ctrl.Request) (ctrl.Result, error) {
	instance, err := r.arocli.AroV1alpha1().Clusters().Get(ctx, arov1alpha1.SingletonClusterName, metav1.GetOptions{})
	if err != nil {
		return reconcile.Result{}, err
	}

	if !instance.Spec.OperatorFlags.GetSimpleBoolean(ENABLED) {
		// controller is disabled
		return reconcile.Result{}, nil
	}

	for _, f := range []func(context.Context, ctrl.Request) (ctrl.Result, error){
		r.reconcileConfiguration,
		r.reconcilePVC, // TODO(mj): This should be removed once we don't have PVC anymore
	} {
		result, err := f(ctx, request)
		if err != nil {
			return result, err
		}
	}
	return reconcile.Result{}, nil
}

func (r *Reconciler) reconcilePVC(ctx context.Context, request ctrl.Request) (ctrl.Result, error) {
	pvcList, err := r.kubernetescli.CoreV1().PersistentVolumeClaims(monitoringName.Namespace).List(ctx, metav1.ListOptions{LabelSelector: prometheusLabels})
	if err != nil {
		return reconcile.Result{}, err
	}

	for _, pvc := range pvcList.Items {
		err = r.kubernetescli.CoreV1().PersistentVolumeClaims(monitoringName.Namespace).Delete(ctx, pvc.Name, metav1.DeleteOptions{})
		if err != nil {
			return reconcile.Result{}, err
		}
	}
	return reconcile.Result{}, nil
}

func (r *Reconciler) reconcileConfiguration(ctx context.Context, request ctrl.Request) (ctrl.Result, error) {
	cm, isCreate, err := r.monitoringConfigMap(ctx)
	if err != nil {
		return reconcile.Result{}, err
	}

	if cm.Data == nil {
		cm.Data = map[string]string{}
	}

	configDataJSON, err := yaml.YAMLToJSON([]byte(cm.Data["config.yaml"]))
	if err != nil {
		return reconcile.Result{}, err
	}

	var configData Config
	err = codec.NewDecoderBytes(configDataJSON, r.jsonHandle).Decode(&configData)
	if err != nil {
		return reconcile.Result{}, err
	}

	changed := false

	// Nil out the fields we don't want set
	if configData.AlertManagerMain.VolumeClaimTemplate != nil {
		configData.AlertManagerMain.VolumeClaimTemplate = nil
		changed = true
	}

	if configData.PrometheusK8s.Retention != "" {
		configData.PrometheusK8s.Retention = ""
		changed = true
	}

	if configData.PrometheusK8s.VolumeClaimTemplate != nil {
		configData.PrometheusK8s.VolumeClaimTemplate = nil
		changed = true
	}

	if !isCreate && !changed {
		return reconcile.Result{}, nil
	}

	var b []byte
	err = codec.NewEncoderBytes(&b, r.jsonHandle).Encode(configData)
	if err != nil {
		return reconcile.Result{}, err
	}

	cmYaml, err := yaml.JSONToYAML(b)
	if err != nil {
		return reconcile.Result{}, err
	}
	cm.Data["config.yaml"] = string(cmYaml)

	if isCreate {
		r.log.Infof("re-creating monitoring configmap. %s", monitoringName.Name)
		_, err = r.kubernetescli.CoreV1().ConfigMaps(monitoringName.Namespace).Create(ctx, cm, metav1.CreateOptions{})
	} else {
		r.log.Infof("updating monitoring configmap. %s", monitoringName.Name)
		_, err = r.kubernetescli.CoreV1().ConfigMaps(monitoringName.Namespace).Update(ctx, cm, metav1.UpdateOptions{})
	}
	return reconcile.Result{}, err
}

func (r *Reconciler) monitoringConfigMap(ctx context.Context) (*corev1.ConfigMap, bool, error) {
	cm, err := r.kubernetescli.CoreV1().ConfigMaps(monitoringName.Namespace).Get(ctx, monitoringName.Name, metav1.GetOptions{})
	if kerrors.IsNotFound(err) {
		return &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      monitoringName.Name,
				Namespace: monitoringName.Namespace,
			},
			Data: nil,
		}, true, nil
	}
	if err != nil {
		return nil, false, err
	}
	return cm, false, nil
}

// SetupWithManager setup the manager
func (r *Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.log.Info("starting cluster monitoring controller")

	aroClusterPredicate := predicate.NewPredicateFuncs(func(o client.Object) bool {
		return o.GetName() == arov1alpha1.SingletonClusterName
	})

	monitoringConfigMapPredicate := predicate.NewPredicateFuncs(func(o client.Object) bool {
		return o.GetName() == monitoringName.Name && o.GetNamespace() == monitoringName.Namespace
	})

	return ctrl.NewControllerManagedBy(mgr).
		For(&arov1alpha1.Cluster{}, builder.WithPredicates(aroClusterPredicate)).
		// https://github.com/kubernetes-sigs/controller-runtime/issues/1173
		// equivalent to For(&v1.ConfigMap{}, ...)., but can't call For multiple times on one builder
		Watches(
			&source.Kind{Type: &corev1.ConfigMap{}},
			&handler.EnqueueRequestForObject{},
			builder.WithPredicates(monitoringConfigMapPredicate),
		).
		Named(controllers.MonitoringControllerName).
		Complete(r)
}
