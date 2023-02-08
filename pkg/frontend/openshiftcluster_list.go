package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/url"
	"time"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/database/cosmosdb"
	"github.com/Azure/ARO-RP/pkg/frontend/middleware"
)

func (f *frontend) getOpenShiftClusters(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := ctx.Value(middleware.ContextKeyLog).(*logrus.Entry)
	vars := mux.Vars(r)

	b, err := f._getOpenShiftClusters(ctx, log, r, f.apis[r.URL.Query().Get(api.APIVersionKey)].OpenShiftClusterConverter, func(skipToken string) (cosmosdb.OpenShiftClusterDocumentIterator, error) {
		prefix := "/subscriptions/" + vars["subscriptionId"] + "/"
		if vars["resourceGroupName"] != "" {
			prefix += "resourcegroups/" + vars["resourceGroupName"] + "/"
		}

		return f.dbOpenShiftClusters.ListByPrefix(vars["subscriptionId"], prefix, skipToken)
	})

	reply(log, w, nil, b, err)
}

func (f *frontend) _getOpenShiftClusters(ctx context.Context, log *logrus.Entry, r *http.Request, converter api.OpenShiftClusterConverter, lister func(string) (cosmosdb.OpenShiftClusterDocumentIterator, error)) ([]byte, error) {
	skipToken, err := f.parseSkipToken(r.URL.String())
	if err != nil {
		return nil, err
	}

	i, err := lister(skipToken)
	if err != nil {
		return nil, err
	}

	docs, err := i.Next(ctx, 10)
	if err != nil {
		return nil, err
	}

	var ocs []*api.OpenShiftCluster
	if docs != nil {
		for _, doc := range docs.OpenShiftClusterDocuments {
			ocs = append(ocs, doc.OpenShiftCluster)
		}
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	ocEnricher := f.ocEnricherFactory(log, f.env, f.m)
	ocEnricher.Enrich(timeoutCtx, ocs...)

	for i := range ocs {
		ocs[i].Properties.ClusterProfile.PullSecret = ""
		ocs[i].Properties.ServicePrincipalProfile.ClientSecret = ""
	}

	nextLink, err := f.buildNextLink(r.Header.Get("Referer"), i.Continuation())
	if err != nil {
		return nil, err
	}

	return json.MarshalIndent(converter.ToExternalList(ocs, nextLink), "", "    ")
}

// parseSkipToken parses originalURL and retrieves skipToken.
// Returns an empty string without an error, if there is no $skipToken parameter in originalURL
func (f *frontend) parseSkipToken(originalURL string) (string, error) {
	u, err := url.Parse(originalURL)
	if err != nil {
		return "", err
	}

	skipToken := u.Query().Get("$skipToken")
	if skipToken == "" {
		return "", nil
	}

	b, err := base64.StdEncoding.DecodeString(skipToken)
	if err != nil {
		return "", err
	}

	output, err := f.aead.Open(b)
	if err != nil {
		return "", err
	}

	return string(output), nil
}

// buildNextLink adds $skipToken parameter into baseURL.
// Returns an empty string without an error, if skipToken is empty.
func (f *frontend) buildNextLink(baseURL, skipToken string) (string, error) {
	if skipToken == "" {
		return "", nil
	}

	u, err := url.Parse(baseURL)
	if err != nil {
		return "", err
	}

	output, err := f.aead.Seal([]byte(skipToken))
	if err != nil {
		return "", err
	}

	query := u.Query()
	query.Set("$skipToken", base64.StdEncoding.EncodeToString(output))
	u.RawQuery = query.Encode()

	return u.String(), nil
}
