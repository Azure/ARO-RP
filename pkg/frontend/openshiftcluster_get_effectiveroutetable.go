package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"net/http"
	"strings"

	"github.com/sirupsen/logrus"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v6"
)

func (f *frontend) _getOpenshiftClusterEffectiveRouteTable(ctx context.Context, w http.ResponseWriter, r *http.Request, log *logrus.Entry) ([]byte, error) {
	subID := r.URL.Query().Get("subid")
	rg := r.URL.Query().Get("rgn")
	nicName := r.URL.Query().Get("nic")

	resourceID := strings.TrimPrefix(r.URL.Path, "/admin")

	dbOpenShiftClusters, err := f.dbGroup.OpenShiftClusters()

	if err != nil {
		//return nil, api.NewCloudError(http.StatusInternalServerError, api.CloudErrorCodeInternalServerError, "", err.Error())
		log.Fatalf("failed to load cluster from cosmosDB: %v", err)
	}

	doc, err := dbOpenShiftClusters.Get(ctx, resourceID)

	subscriptionDoc, err := f.getSubscriptionDocument(ctx, doc.Key)

	if err != nil {
		//return nil, err
		log.Fatalf("failed to retrieve subscription document: %v", err)
	}

	credential, err := f.env.FPNewClientCertificateCredential(subscriptionDoc.Subscription.Properties.TenantID, nil)

	if err != nil {
		//return nil, err
		log.Fatalf("failed to create client: %v", err)
	}

	clientFactory, err := armnetwork.NewClientFactory(subID, credential, nil)

	if err != nil {
		log.Fatalf("failed to create client: %v", err)
	}

	poller, err := clientFactory.NewInterfacesClient().BeginGetEffectiveRouteTable(context.Background(), rg, nicName, nil)

	if err != nil {
		log.Fatalf("failed to finish the request: %v", err)
	}

	res, err := poller.PollUntilDone(ctx, nil)

	if err != nil {
		log.Fatalf("failed to pull the result: %v", err)
	}

	e, err := res.EffectiveRouteListResult.MarshalJSON()

	reply(log, w, nil, e, err)

	return e, err
}
