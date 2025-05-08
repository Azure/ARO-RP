package adminactions

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/json"

	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/applens"
)

type AppLensActions interface {
	AppLensGetDetector(ctx context.Context, detectorId string) ([]byte, error)
	AppLensListDetectors(ctx context.Context) ([]byte, error)
}

type appLensActions struct {
	log *logrus.Entry
	env env.Interface
	oc  *api.OpenShiftCluster

	appLens applens.AppLensClient
}

func NewAppLensActions(log *logrus.Entry, env env.Interface, oc *api.OpenShiftCluster,
	subscriptionDoc *api.SubscriptionDocument) (AppLensActions, error) {
	fpClientCertCred, err := env.FPNewClientCertificateCredential(env.Environment().AppLensTenantID, nil)
	if err != nil {
		return nil, err
	}

	appLensClient, err := applens.NewAppLensClient(env.Environment(), fpClientCertCred)
	if err != nil {
		return nil, err
	}

	return &appLensActions{
		log:     log,
		env:     env,
		oc:      oc,
		appLens: appLensClient,
	}, nil
}

func (a *appLensActions) AppLensGetDetector(ctx context.Context, detectorId string) ([]byte, error) {
	detector, err := a.appLens.GetDetector(ctx, &applens.GetDetectorOptions{ResourceID: a.oc.ID, DetectorID: detectorId, Location: a.oc.Location})

	if err != nil {
		return nil, err
	}

	return json.Marshal(detector)
}

func (a *appLensActions) AppLensListDetectors(ctx context.Context) ([]byte, error) {
	detectors, err := a.appLens.ListDetectors(ctx, &applens.ListDetectorsOptions{ResourceID: a.oc.ID, Location: a.oc.Location})

	if err != nil {
		return nil, err
	}

	return json.Marshal(detectors)
}
