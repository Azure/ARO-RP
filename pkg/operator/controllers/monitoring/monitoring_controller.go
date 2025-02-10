package monitoring

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/json"

	"github.com/sirupsen/logrus"
	"github.com/ugorji/go/codec"

	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/yaml"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/operator"
	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	"github.com/Azure/ARO-RP/pkg/operator/controllers/base"
	"github.com/Azure/ARO-RP/pkg/operator/predicates"
)

const (
	ControllerName = "Monitoring"
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

type MonitoringReconciler struct {
	base.AROController

	jsonHandle *codec.JsonHandle
}

func NewReconciler(log *logrus.Entry, client client.Client) *MonitoringReconciler {
	return &MonitoringReconciler{
		AROController: base.AROController{
			Log:    log,
			Client: client,
			Name:   ControllerName,
		},
		jsonHandle: new(codec.JsonHandle),
	}
}

func (r *MonitoringReconciler) Reconcile(ctx context.Context, request ctrl.Request) (ctrl.Result, error) {
	instance, err := r.GetCluster(ctx)
	if err != nil {
		return reconcile.Result{}, err
	}

	if !instance.Spec.OperatorFlags.GetSimpleBoolean(operator.MonitoringEnabled) {
		r.Log.Debug("controller is disabled")
		return reconcile.Result{}, nil
	}

	r.Log.Debug("running")
	for _, f := range []func(context.Context) (ctrl.Result, error){
		r.reconcileConfiguration,
		r.reconcilePVC, // TODO(mj): This should be removed once we don't have PVC anymore
	} {
		result, err := f(ctx)
		if err != nil {
			r.Log.Error(err)
			r.SetDegraded(ctx, err)
			return result, err
		}
	}

	return reconcile.Result{}, nil
}

func (r *MonitoringReconciler) reconcilePVC(ctx context.Context) (ctrl.Result, error) {
	pvcList := &corev1.PersistentVolumeClaimList{}
	selector, _ := labels.Parse(prometheusLabels)
	err := r.Client.List(ctx, pvcList, &client.ListOptions{
		Namespace:     monitoringName.Namespace,
		LabelSelector: selector,
	})
	if err != nil {
		return reconcile.Result{}, err
	}

	for _, pvc := range pvcList.Items {
		err = r.Client.Delete(ctx, &pvc)
		if err != nil {
			return reconcile.Result{}, err
		}
	}
	return reconcile.Result{}, nil
}

func (r *MonitoringReconciler) reconcileConfiguration(ctx context.Context) (ctrl.Result, error) {
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
		r.Log.Infof("re-creating monitoring configmap. %s", monitoringName.Name)
		err = r.Client.Create(ctx, cm)
	} else {
		r.Log.Infof("updating monitoring configmap. %s", monitoringName.Name)
		err = r.Client.Update(ctx, cm)
	}
	return reconcile.Result{}, err
}

func (r *MonitoringReconciler) monitoringConfigMap(ctx context.Context) (*corev1.ConfigMap, bool, error) {
	cm := &corev1.ConfigMap{}
	err := r.Client.Get(ctx, monitoringName, cm)
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
func (r *MonitoringReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.Log.Info("starting cluster monitoring controller")

	monitoringConfigMapPredicate := predicate.NewPredicateFuncs(func(o client.Object) bool {
		return o.GetName() == monitoringName.Name && o.GetNamespace() == monitoringName.Namespace
	})

	return ctrl.NewControllerManagedBy(mgr).
		For(&arov1alpha1.Cluster{}, builder.WithPredicates(predicate.And(predicates.AROCluster, predicate.GenerationChangedPredicate{}))).
		// https://github.com/kubernetes-sigs/controller-runtime/issues/1173
		// equivalent to For(&v1.ConfigMap{}, ...)., but can't call For multiple times on one builder
		Watches(
			&corev1.ConfigMap{},
			&handler.EnqueueRequestForObject{},
			builder.WithPredicates(monitoringConfigMapPredicate),
		).
		Named(ControllerName).
		Complete(r)
}
