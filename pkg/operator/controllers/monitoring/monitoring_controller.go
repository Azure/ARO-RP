package monitoring

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"github.com/ghodss/yaml"
	"github.com/sirupsen/logrus"
	"github.com/ugorji/go/codec"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/util/retry"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"github.com/Azure/ARO-RP/pkg/api"
	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
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
			Spec struct {
				api.MissingFields
				Resources struct {
					api.MissingFields
					Requests struct {
						api.MissingFields
						Storage string `json:"storage,omitempty"`
					} `json:"requests,omitempty"`
				} `json:"resources,omitempty"`
			} `json:"spec,omitempty"`
		} `json:"volumeClaimTemplate,omitempty"`
	} `json:"prometheusK8s,omitempty"`
}

var defaultConfig = `prometheusK8s:
  retention: 15d
  volumeClaimTemplate:
    spec:
      resources:
        requests:
          storage: 100Gi
`

type Reconciler struct {
	kubernetescli kubernetes.Interface
	log           *logrus.Entry
	jsonHandle    *codec.JsonHandle
}

func NewReconciler(log *logrus.Entry, kubernetescli kubernetes.Interface) *Reconciler {
	return &Reconciler{
		kubernetescli: kubernetescli,
		log:           log,
		jsonHandle:    new(codec.JsonHandle),
	}
}

func (r *Reconciler) Reconcile(request ctrl.Request) (ctrl.Result, error) {
	// TODO(mj): controller-runtime master fixes the need for this (https://github.com/kubernetes-sigs/controller-runtime/blob/master/pkg/reconcile/reconcile.go#L93) but it's not yet released.
	ctx := context.Background()
	if request.NamespacedName != monitoringName &&
		request.Name != arov1alpha1.SingletonClusterName {
		return reconcile.Result{}, nil
	}

	return reconcile.Result{}, retry.RetryOnConflict(retry.DefaultRetry, func() error {
		cm, isCreate, err := r.monitoringConfigMap(ctx)
		if err != nil {
			return err
		}
		if cm.Data == nil {
			cm.Data = map[string]string{}
		}

		configDataJSON, err := yaml.YAMLToJSON([]byte(cm.Data["config.yaml"]))
		if err != nil {
			return err
		}

		var configData Config
		err = codec.NewDecoderBytes(configDataJSON, r.jsonHandle).Decode(&configData)
		if err != nil {
			return err
		}

		changed := false
		if configData.PrometheusK8s.Retention != "15d" {
			configData.PrometheusK8s.Retention = "15d"
			changed = true
		}

		if configData.PrometheusK8s.VolumeClaimTemplate.Spec.Resources.Requests.Storage != "100Gi" {
			configData.PrometheusK8s.VolumeClaimTemplate.Spec.Resources.Requests.Storage = "100Gi"
			changed = true
		}

		if !isCreate && !changed {
			return nil
		}

		var b []byte
		err = codec.NewEncoderBytes(&b, r.jsonHandle).Encode(configData)
		if err != nil {
			return err
		}

		cmYaml, err := yaml.JSONToYAML(b)
		if err != nil {
			return err
		}
		cm.Data["config.yaml"] = string(cmYaml)

		if isCreate {
			r.log.Info("re-creating monitoring configmap")
			_, err = r.kubernetescli.CoreV1().ConfigMaps(monitoringName.Namespace).Create(ctx, cm, metav1.CreateOptions{})
		} else {
			r.log.Info("updating monitoring configmap")
			_, err = r.kubernetescli.CoreV1().ConfigMaps(monitoringName.Namespace).Update(ctx, cm, metav1.UpdateOptions{})
		}
		return err
	})
}

func (r *Reconciler) monitoringConfigMap(ctx context.Context) (*v1.ConfigMap, bool, error) {
	cm, err := r.kubernetescli.CoreV1().ConfigMaps(monitoringName.Namespace).Get(ctx, monitoringName.Name, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		return &v1.ConfigMap{
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

func triggerReconcile(cm *corev1.ConfigMap) bool {
	return cm.Name == monitoringName.Name && cm.Namespace == monitoringName.Namespace
}

// SetupWithManager setup the manager
func (r *Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.log.Info("starting starting cluster monitoring controller")

	isMonitoringConfigMap := predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			_, ok := e.ObjectOld.(*arov1alpha1.Cluster)
			if ok {
				return true
			}

			cm, ok := e.ObjectOld.(*corev1.ConfigMap)
			return ok && triggerReconcile(cm)
		},
		CreateFunc: func(e event.CreateEvent) bool {
			_, ok := e.Object.(*arov1alpha1.Cluster)
			if ok {
				return true
			}

			cm, ok := e.Object.(*corev1.ConfigMap)
			return ok && triggerReconcile(cm)
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			_, ok := e.Object.(*arov1alpha1.Cluster)
			if ok {
				return true
			}

			cm, ok := e.Object.(*corev1.ConfigMap)
			return ok && triggerReconcile(cm)
		},
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&arov1alpha1.Cluster{}).
		// https://github.com/kubernetes-sigs/controller-runtime/issues/1173
		// equivalent to For(&v1.ConfigMap{})., but can't call For multiple times on one builder
		Watches(&source.Kind{Type: &v1.ConfigMap{}}, &handler.EnqueueRequestForObject{}).
		WithEventFilter(isMonitoringConfigMap).
		Named(controllers.MonitoringControllerName).
		Complete(r)
}
