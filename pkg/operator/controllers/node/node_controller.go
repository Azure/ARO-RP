package node

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/sirupsen/logrus"

	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/kubectl/pkg/drain"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/Azure/ARO-RP/pkg/operator"
	"github.com/Azure/ARO-RP/pkg/operator/controllers/base"
	"github.com/Azure/ARO-RP/pkg/util/ready"
)

const (
	ControllerName = "Node"
)

// Reconciler spots nodes that look like they're stuck upgrading.  When this
// happens, it tries to drain them disabling eviction (so PDBs don't count).
type Reconciler struct {
	base.AROController

	kubernetescli kubernetes.Interface
}

func NewReconciler(log *logrus.Entry, client client.Client, kubernetescli kubernetes.Interface) *Reconciler {
	return &Reconciler{
		AROController: base.AROController{
			Log:    log,
			Client: client,
			Name:   ControllerName,
		},

		kubernetescli: kubernetescli,
	}
}

func (r *Reconciler) Reconcile(ctx context.Context, request ctrl.Request) (ctrl.Result, error) {
	instance, err := r.GetCluster(ctx)
	if err != nil {
		return reconcile.Result{}, err
	}

	if !instance.Spec.OperatorFlags.GetSimpleBoolean(operator.NodeDrainerEnabled) {
		r.Log.Debug("controller is disabled")
		return reconcile.Result{}, nil
	}

	r.Log.Debug("running")

	node := &corev1.Node{}
	err = r.Client.Get(ctx, types.NamespacedName{Name: request.Name}, node)
	if err != nil {
		if kerrors.IsNotFound(err) {
			r.Log.Debug(fmt.Sprintf("node %s not found", node.Name))
			return reconcile.Result{}, nil
		}
		r.Log.Error(err)
		r.SetDegraded(ctx, err)
		return reconcile.Result{}, err
	}

	// don't interfere with masters, don't want to trample etcd-quorum-guard.
	if node.Labels != nil {
		if _, ok := node.Labels["node-role.kubernetes.io/master"]; ok {
			r.ClearConditions(ctx)
			return reconcile.Result{}, nil
		}
	}

	if !isDraining(node) {
		// we're not draining: ensure our annotation is not set and return
		if getAnnotation(&node.ObjectMeta, annotationDrainStartTime) == "" {
			r.ClearConditions(ctx)
			return reconcile.Result{}, nil
		}

		delete(node.Annotations, annotationDrainStartTime)

		err = r.Client.Update(ctx, node)
		if err != nil {
			r.Log.Error(err)
			r.SetDegraded(ctx, err)
		}

		return reconcile.Result{}, err
	}

	// we're draining: ensure our annotation is set
	t, err := time.Parse(time.RFC3339, getAnnotation(&node.ObjectMeta, annotationDrainStartTime))
	if err != nil {
		t = time.Now().UTC()
		setAnnotation(&node.ObjectMeta, annotationDrainStartTime, t.Format(time.RFC3339))

		err = r.Client.Update(ctx, node)
		if err != nil {
			r.Log.Error(err)
			r.SetDegraded(ctx, err)

			return reconcile.Result{}, err
		}
	}

	// if our deadline hasn't expired, requeue ourselves and return
	deadline := t.Add(gracePeriod)
	now := time.Now()
	if deadline.After(now) {
		r.SetProgressing(ctx, fmt.Sprintf("Draining node %s", request.Name))

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
			r.Log.Printf("deleted pod %s/%s", pod.Namespace, pod.Name)
		},
		Out:    r.Log.Writer(),
		ErrOut: r.Log.Writer(),
	}, request.Name)
	if err != nil {
		r.Log.Error(err)
		r.SetDegraded(ctx, err)

		return reconcile.Result{}, err
	}

	// ensure our annotation is not set and return
	delete(node.Annotations, annotationDrainStartTime)

	err = r.Client.Update(ctx, node)
	if err != nil {
		r.Log.Error(err)
		r.SetDegraded(ctx, err)

		return reconcile.Result{}, err
	}

	r.ClearConditions(ctx)
	return reconcile.Result{}, nil
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
