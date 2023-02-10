package node

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/kubectl/pkg/drain"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	"github.com/Azure/ARO-RP/pkg/util/ready"
)

const (
	ControllerName = "Node"

	controllerEnabled = "aro.nodedrainer.enabled"
)

// Reconciler spots nodes that look like they're stuck upgrading.  When this
// happens, it tries to drain them disabling eviction (so PDBs don't count).
type Reconciler struct {
	log *logrus.Entry

	kubernetescli kubernetes.Interface

	client client.Client
}

func NewReconciler(log *logrus.Entry, client client.Client, kubernetescli kubernetes.Interface) *Reconciler {
	return &Reconciler{
		log:           log,
		kubernetescli: kubernetescli,
		client:        client,
	}
}

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
	node := &corev1.Node{}
	err = r.client.Get(ctx, types.NamespacedName{Name: request.Name}, node)
	if err != nil {
		r.log.Error(err)
		return reconcile.Result{}, err
	}

	// don't interfere with masters, don't want to trample etcd-quorum-guard.
	if node.Labels != nil {
		if _, ok := node.Labels["node-role.kubernetes.io/master"]; ok {
			return reconcile.Result{}, nil
		}
	}

	if !isDraining(node) {
		// we're not draining: ensure our annotation is not set and return
		if getAnnotation(&node.ObjectMeta, annotationDrainStartTime) == "" {
			return reconcile.Result{}, nil
		}

		delete(node.Annotations, annotationDrainStartTime)

		err = r.client.Update(ctx, node)
		if err != nil {
			r.log.Error(err)
		}

		return reconcile.Result{}, err
	}

	// we're draining: ensure our annotation is set
	t, err := time.Parse(time.RFC3339, getAnnotation(&node.ObjectMeta, annotationDrainStartTime))
	if err != nil {
		t = time.Now().UTC()
		setAnnotation(&node.ObjectMeta, annotationDrainStartTime, t.Format(time.RFC3339))

		err = r.client.Update(ctx, node)
		if err != nil {
			r.log.Error(err)
			return reconcile.Result{}, err
		}
	}

	// if our deadline hasn't expired, requeue ourselves and return
	deadline := t.Add(gracePeriod)
	now := time.Now()
	if deadline.After(now) {
		return reconcile.Result{RequeueAfter: deadline.Sub(now)}, nil
	}

	// drain the node disabling eviction
	err = drain.RunNodeDrain(&drain.Helper{
		Client:              r.kubernetescli,
		Force:               true,
		GracePeriodSeconds:  -1,
		IgnoreAllDaemonSets: true,
		Timeout:             60 * time.Second,
		DeleteEmptyDirData:  true,
		DisableEviction:     true,
		OnPodDeletedOrEvicted: func(pod *corev1.Pod, usingEviction bool) {
			r.log.Printf("deleted pod %s/%s", pod.Namespace, pod.Name)
		},
		Out:    r.log.Writer(),
		ErrOut: r.log.Writer(),
	}, request.Name)
	if err != nil {
		r.log.Error(err)
		return reconcile.Result{}, err
	}

	// ensure our annotation is not set and return
	delete(node.Annotations, annotationDrainStartTime)

	err = r.client.Update(ctx, node)
	if err != nil {
		r.log.Error(err)
	}

	return reconcile.Result{}, err
}

// SetupWithManager setup our mananger
func (r *Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Node{}).
		Named(ControllerName).
		Complete(r)
}

func getAnnotation(m *metav1.ObjectMeta, k string) string {
	if m.Annotations == nil {
		return ""
	}

	return m.Annotations[k]
}

func setAnnotation(m *metav1.ObjectMeta, k, v string) {
	if m.Annotations == nil {
		m.Annotations = map[string]string{}
	}

	m.Annotations[k] = v
}

func isDraining(node *corev1.Node) bool {
	if !ready.NodeIsReady(node) ||
		!node.Spec.Unschedulable ||
		getAnnotation(&node.ObjectMeta, annotationCurrentConfig) == "" ||
		getAnnotation(&node.ObjectMeta, annotationDesiredConfig) == "" ||
		getAnnotation(&node.ObjectMeta, annotationCurrentConfig) == getAnnotation(&node.ObjectMeta, annotationDesiredConfig) {
		return false
	}

	if getAnnotation(&node.ObjectMeta, annotationState) == stateWorking {
		return true
	}

	if getAnnotation(&node.ObjectMeta, annotationState) == stateDegraded &&
		strings.HasPrefix(getAnnotation(&node.ObjectMeta, annotationReason), "failed to drain node") {
		return true
	}

	return false
}
