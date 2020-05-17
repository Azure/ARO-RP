package controllers

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"time"

	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/Azure/ARO-RP/operator/deploy"
)

var (
	ReconcileResultRequeue = reconcile.Result{RequeueAfter: 5 * time.Minute, Requeue: true}
	OperatorNamespace      = deploy.KubeNamespace
)
