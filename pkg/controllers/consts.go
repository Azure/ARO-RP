package controllers

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"time"

	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	OperatorNamespace = "openshift-azure-operator"
)

var (
	ReconcileResultRequeue = reconcile.Result{RequeueAfter: 2 * time.Minute, Requeue: true}
	ReconcileResultError   = reconcile.Result{RequeueAfter: time.Minute, Requeue: true}
	ReconcileResultIgnore  = reconcile.Result{Requeue: false}
	ReconcileResultDone    = reconcile.Result{Requeue: false}
)
