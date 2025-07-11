package portal

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"encoding/json"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/gorilla/mux"

	"github.com/Azure/go-autorest/autorest/azure"

	"github.com/Azure/ARO-RP/pkg/portal/cluster"
	"github.com/Azure/ARO-RP/pkg/portal/prometheus"
)

type AdminOpenShiftCluster struct {
	Key                     string `json:"key"`
	Name                    string `json:"name"`
	Subscription            string `json:"subscription"`
	ResourceGroup           string `json:"resourceGroup"`
	ResourceId              string `json:"resourceId"`
	ProvisioningState       string `json:"provisioningState"`
	FailedProvisioningState string `json:"failedprovisioningState"`
	Version                 string `json:"version"`
	CreatedAt               string `json:"createdAt"`
	LastModified            string `json:"lastModified"`
	ProvisionedBy           string `json:"provisionedBy"`
}

func (p *portal) clusters(w http.ResponseWriter, r *http.Request) {
	dbOpenShiftClusters, err := p.dbGroup.OpenShiftClusters()
	if err != nil {
		p.internalServerError(w, err)
		return
	}
	ctx := r.Context()

	docs, err := dbOpenShiftClusters.ListAll(ctx)
	if err != nil {
		p.internalServerError(w, err)
		return
	}

	clusters := make([]*AdminOpenShiftCluster, 0, len(docs.OpenShiftClusterDocuments))
	for _, doc := range docs.OpenShiftClusterDocuments {
		if doc.OpenShiftCluster == nil {
			continue
		}

		ps := doc.OpenShiftCluster.Properties.ProvisioningState
		fps := doc.OpenShiftCluster.Properties.FailedProvisioningState
		subscription := "Unknown"
		resourceGroup := "Unknown"
		name := "Unknown"

		if resource, err := azure.ParseResourceID(doc.OpenShiftCluster.ID); err == nil {
			subscription = resource.SubscriptionID
			resourceGroup = resource.ResourceGroup
			name = resource.ResourceName
		}

		createdAt := "Unknown"
		if !doc.OpenShiftCluster.Properties.CreatedAt.IsZero() {
			createdAt = doc.OpenShiftCluster.Properties.CreatedAt.Format(time.RFC3339)
		}

		lastModified := "Unknown"
		if doc.OpenShiftCluster.SystemData.LastModifiedAt != nil {
			lastModified = doc.OpenShiftCluster.SystemData.LastModifiedAt.Format(time.RFC3339)
		}

		clusters = append(clusters, &AdminOpenShiftCluster{
			Key:                     doc.ID,
			ResourceId:              doc.OpenShiftCluster.ID,
			Name:                    name,
			Subscription:            subscription,
			ResourceGroup:           resourceGroup,
			Version:                 doc.OpenShiftCluster.Properties.ClusterProfile.Version,
			CreatedAt:               createdAt,
			LastModified:            lastModified,
			ProvisionedBy:           doc.OpenShiftCluster.Properties.ProvisionedBy,
			ProvisioningState:       ps.String(),
			FailedProvisioningState: fps.String(),
		})
	}

	sort.SliceStable(clusters, func(i, j int) bool { return strings.Compare(clusters[i].Key, clusters[j].Key) < 0 })

	b, err := json.MarshalIndent(clusters, "", "    ")
	if err != nil {
		p.internalServerError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(b)
}

func (p *portal) clusterOperators(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	fetcher, err := p.makeFetcher(ctx, r)
	if err != nil {
		p.internalServerError(w, err)
		return
	}

	clusterOperators, err := fetcher.ClusterOperators(ctx)
	if err != nil {
		p.internalServerError(w, err)
		return
	}

	b, err := json.MarshalIndent(clusterOperators, "", "    ")
	if err != nil {
		p.internalServerError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(b)
}

func (p *portal) nodes(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	fetcher, err := p.makeFetcher(ctx, r)
	if err != nil {
		p.internalServerError(w, err)
		return
	}

	nodes, err := fetcher.Nodes(ctx)
	if err != nil {
		p.internalServerError(w, err)
		return
	}

	b, err := json.MarshalIndent(nodes, "", "    ")
	if err != nil {
		p.internalServerError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(b)
}

func (p *portal) machines(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	fetcher, err := p.makeFetcher(ctx, r)
	if err != nil {
		p.internalServerError(w, err)
		return
	}

	machines, err := fetcher.Machines(ctx)
	if err != nil {
		p.internalServerError(w, err)
		return
	}

	b, err := json.MarshalIndent(machines, "", "    ")
	if err != nil {
		p.internalServerError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(b)
}

func (p *portal) machineSets(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	fetcher, err := p.makeFetcher(ctx, r)
	if err != nil {
		p.internalServerError(w, err)
		return
	}

	machineSets, err := fetcher.MachineSets(ctx)
	if err != nil {
		p.internalServerError(w, err)
		return
	}

	b, err := json.MarshalIndent(machineSets, "", "    ")
	if err != nil {
		p.internalServerError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(b)
}

func (p *portal) statistics(w http.ResponseWriter, r *http.Request) {
	dbOpenShiftClusters, err := p.dbGroup.OpenShiftClusters()
	if err != nil {
		p.internalServerError(w, err)
		return
	}

	ctx := r.Context()
	duration := r.URL.Query().Get("duration")
	parsedDuration, err := time.ParseDuration(duration)
	if err != nil {
		p.badRequest(w, err)
		return
	}
	endTimeString := r.URL.Query().Get("endtime")
	endTime, err := time.Parse(time.RFC3339, endTimeString)
	if err != nil {
		p.badRequest(w, err)
		return
	}
	apiVars := mux.Vars(r)
	statisticsType := apiVars["statisticsType"]
	subscriptionID := apiVars["subscription"]
	resourceGroup := apiVars["resourceGroup"]
	clusterName := apiVars["clusterName"]
	resourceID := p.getResourceID(subscriptionID, resourceGroup, clusterName)
	promQuery, err := cluster.GetPromQuery(statisticsType)
	if err != nil {
		p.badRequest(w, err)
		return
	}
	prom := prometheus.New(p.log, dbOpenShiftClusters, p.dialer)
	httpClient, err := prom.Cli(ctx, resourceID)
	if err != nil {
		p.internalServerError(w, err)
		return
	}

	fetcher, err := p.makeFetcher(ctx, r)
	if err != nil {
		p.internalServerError(w, err)
		return
	}
	promHost, promScheme := prom.GetPrometheusHostAndScheme()
	prometheusURL := promScheme + "://" + promHost
	APIStatistics, err := fetcher.Statistics(ctx, httpClient, promQuery, parsedDuration, endTime, prometheusURL)
	if err != nil {
		p.internalServerError(w, err)
		return
	}
	b, err := json.MarshalIndent(APIStatistics, "", "    ")
	if err != nil {
		p.internalServerError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write(b)
	if err != nil {
		p.log.Error(err)
	}
}
