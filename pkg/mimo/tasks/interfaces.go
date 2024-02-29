package tasks

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/util/clienthelper"
)

type TaskContext interface {
	Environment() env.Interface
	ClientHelper() (clienthelper.Interface, error)
	Log() *logrus.Entry
}

type TaskFunc func(context.Context, TaskContext, *api.MaintenanceManifestDocument, *api.OpenShiftClusterDocument) (api.MaintenanceManifestState, string)
