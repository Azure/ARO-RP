package mimo

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/Azure/go-autorest/autorest"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/util/clienthelper"
)

type TaskContext interface {
	context.Context
	Now() time.Time
	Environment() env.Interface
	ClientHelper() (clienthelper.Interface, error)
	Log() *logrus.Entry
	LocalFpAuthorizer() (autorest.Authorizer, error)
	TaskID() api.MIMOTaskID

	// OpenShiftCluster
	GetClusterUUID() string
	GetOpenShiftClusterProperties() api.OpenShiftClusterProperties
	GetOpenshiftClusterDocument() *api.OpenShiftClusterDocument

	SetResultMessage(string)
	GetResultMessage() string
}

func GetTaskContext(c context.Context) (TaskContext, error) {
	r, ok := c.(TaskContext)
	if !ok {
		return nil, fmt.Errorf("cannot convert %v", r)
	}

	return r, nil
}
