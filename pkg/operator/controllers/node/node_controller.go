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
	"k8s.io/client-go/kubernetes"
	"k8s.io/kubectl/pkg/drain"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/Azure/ARO-RP/pkg/operator/controllers"
	"github.com/Azure/ARO-RP/pkg/util/ready"
)

const (
	annotationCurrentConfig  = "machineconfiguration.openshift.io/currentConfig"
	annotationDesiredConfig  = "machineconfiguration.openshift.io/desiredConfig"
	annotationReason         = "machineconfiguration.openshift.io/reason"
	annotationState          = "machineconfiguration.openshift.io/state"
	annotationDrainStartTime = "aro.openshift.io/drainStartTime"
	stateDegraded            = "Degraded"
	stateWorking             = "Working"
	gracePeriod              = time.Hour
)

// NodeReconciler spots nodes that look like they're stuck upgrading.  When this
// happens, it tries to drain them disabling eviction (so PDBs don't count).
type NodeReconciler struct {
	log           *logrus.Entry
	kubernetescli kubernetes.Interface
}

func NewNodeReconciler(log *logrus.Entry, kubernetescli kubernetes.Interface) *NodeReconciler {
	return &NodeReconciler{
		log:           log,
		kubernetescli: kubernetescli,
	}
}

func (r *NodeReconciler) Reconcile(request ctrl.Request) (ctrl.Result, error) {
	// TODO(mj): controller-runtime master fixes the need for this (https://github.com/kubernetes-sigs/controller-runtime/blob/master/pkg/reconcile/reconcile.go#L93) but it's not yet released.
	ctx := context.Background()

	node, err := r.kubernetescli.CoreV1().Nodes().Get(ctx, request.Name, metav1.GetOptions{})
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

		_, err = r.kubernetescli.CoreV1().Nodes().Update(ctx, node, metav1.UpdateOptions{})
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

		node, err = r.kubernetescli.CoreV1().Nodes().Update(ctx, node, metav1.UpdateOptions{})
		if err != nil {
			r.log.Error(err)
			return reconcile.Result{}, err
		}
	}

	// if our deadline hasn't expired, requeue ourselves and return
	deadline := t.Add(gracePeriod)
	now := time.Now()
	if deadline.After(now) {
		return reconcile.Result{
			RequeueAfter: deadline.Sub(now),
		}, err
	}

	// drain the node disabling eviction
	err = drain.RunNodeDrain(&drain.Helper{
		Client:              r.kubernetescli,
		Force:               true,
		GracePeriodSeconds:  -1,
		IgnoreAllDaemonSets: true,
		Timeout:             60 * time.Second,
		DeleteLocalData:     true,
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

	_, err = r.kubernetescli.CoreV1().Nodes().Update(ctx, node, metav1.UpdateOptions{})
	if err != nil {
		r.log.Error(err)
	}

	return reconcile.Result{}, err
}

// SetupWithManager setup our mananger
func (r *NodeReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Node{}).
		Named(controllers.NodeControllerName).
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
