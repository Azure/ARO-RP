package monitoring

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"reflect"

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

var monitoringName = types.NamespacedName{Name: "cluster-monitoring-config", Namespace: "openshift-monitoring"}

// Config represents cluster monitoring stack configuration.
// Reconciler reconciles retention and storage settings,
// MissingFields are used to preserve settings configured by user.
type Config struct {
	api.MissingFields
	PrometheusK8s struct {
		api.MissingFields
		Retention           string `json:"retention,omitempty"`
		VolumeClaimTemplate struct {
			api.MissingFields
		} `json:"volumeClaimTemplate,omitempty"`
	} `json:"prometheusK8s,omitempty"`
}

var defaultConfig = `prometheusK8s: {}`

type Reconciler struct {
	arocli        aroclient.Interface
	kubernetescli kubernetes.Interface
	log           *logrus.Entry
	jsonHandle    *codec.JsonHandle
}

func NewReconciler(log *logrus.Entry, kubernetescli kubernetes.Interface, arocli aroclient.Interface) *Reconciler {
	return &Reconciler{
		arocli:        arocli,
		kubernetescli: kubernetescli,
		log:           log,
		jsonHandle:    new(codec.JsonHandle),
	}
}

func (r *Reconciler) Reconcile(ctx context.Context, request ctrl.Request) (ctrl.Result, error) {
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
	// we are disabling persistence. We use omitempty on the struct to
	// clean the fields
	if configData.PrometheusK8s.Retention != "" {
		configData.PrometheusK8s.Retention = ""
		changed = true
	}
	if !reflect.DeepEqual(configData.PrometheusK8s.VolumeClaimTemplate, struct{ api.MissingFields }{}) {
		configData.PrometheusK8s.VolumeClaimTemplate = struct{ api.MissingFields }{}
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
			Data: map[string]string{
				"config.yaml": defaultConfig,
			},
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
