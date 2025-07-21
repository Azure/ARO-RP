package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v6"
	"github.com/sirupsen/logrus"
	"github.com/Azure/ARO-RP/pkg/frontend/middleware"
)

func (f *frontend) getAdminOpenshiftClusterEffectiveRouteTable(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    log := ctx.Value(middleware.ContextKeyLog).(*logrus.Entry)
    r.URL.Path = filepath.Dir(r.URL.Path)

    effectiveRouteTableColleciton, err := f._getAdminOpenshiftClusterEffectiveRouteTable(ctx, r, log)

    if err != nil {
        log.Fatalf("Unable to get effective route table: %v", err)   
    }

    convertedEffectiveRouteTableCollection, err := effectiveRouteTableColleciton.EffectiveRouteListResult.MarshalJSON()

    adminReply(log, w, nil, convertedEffectiveRouteTableCollection, err)
}

func (f *frontend) _getAdminOpenshiftClusterEffectiveRouteTable(ctx context.Context, r *http.Request, log *logrus.Entry) (armnetwork.InterfacesClientGetEffectiveRouteTableResponse, error) {
    subID := r.URL.Query().Get("subscriptionID")
    rg := r.URL.Query().Get("resourceGroupName")
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

    return res, err
}