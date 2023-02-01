package portal

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/gorilla/mux"

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
	ctx := r.Context()

	docs, err := p.dbOpenShiftClusters.ListAll(ctx)
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
	ctx := r.Context()
	duration := r.URL.Query().Get("duration")
	endTimeString := r.URL.Query().Get("endtime")
	endTime, err := time.Parse(time.RFC3339, endTimeString)
	if err != nil {
		p.internalServerError(w, err)
		return
	}
	apiVars := mux.Vars(r)
	subscription := apiVars["subscription"]
	resourceGroup := apiVars["resourceGroup"]
	clusterName := apiVars["clusterName"]
	statisticsType := apiVars["statisticsType"]
	promQuery := ""

	switch statisticsType {
	//kube-apiserver
	case "kubeapicodes":
		promQuery = "sum(rate(apiserver_request_total{job=\"apiserver\",code=~\"[45]..\"}[10m])) by (code, verb)"
	case "kubeapicpu":
		promQuery = "rate(process_cpu_seconds_total{job=\"apiserver\"}[5m])"
	case "kubeapimemory":
		promQuery = "process_resident_memory_bytes{job=\"apiserver\"}"
	//kube-controller-manager
	case "kubecontrollermanagercodes":
		promQuery = "sum(rate(rest_client_requests_total{job=\"kube-controller-manager\"}[5m])) by (code)"
	case "kubecontrollermanagercpu":
		promQuery = "rate(process_cpu_seconds_total{job=\"kube-controller-manager\"}[5m])"
	case "kubecontrollermanagermemory":
		promQuery = "process_resident_memory_bytes{job=\"kube-controller-manager\"}"
	//DNS
	case "dnsresponsecodes":
		promQuery = "sum(rate(coredns_dns_responses_total[5m])) by (rcode)"
	case "dnserrorrate":
		promQuery = "sum(rate(coredns_dns_responses_total{rcode=~\"SERVFAIL|NXDOMAIN\"}[5m])) by (pod) / sum(rate(coredns_dns_responses_total{rcode=~\"NOERROR\"}[5m])) by (pod)"

	case "dnshealthcheck":
		promQuery = "histogram_quantile(0.99, sum(rate(coredns_health_request_duration_seconds_bucket[5m])) by (le))"
	case "dnsforwardedtraffic":
		promQuery = ""
	case "dnsalltraffic":
		promQuery = "histogram_quantile(0.95, sum(rate(coredns_dns_request_duration_seconds_bucket[5m])) by (le))"
	//Ingress
	case "ingresscontrollercondition":
		promQuery = "sum(ingress_controller_conditions) by (condition)"

	default:
		p.internalServerError(w, errors.New("invalid statistic type '"+statisticsType+"'"))
		return
	}

	resourceID :=
		strings.ToLower(
			fmt.Sprintf(
				"/subscriptions/%s/resourceGroups/%s/providers/Microsoft.RedHatOpenShift/openShiftClusters/%s",
				subscription, resourceGroup, clusterName))

	prom := prometheus.New(p.log, p.dbOpenShiftClusters, p.dialer, p.authenticatedRouter)
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
	APIStatistics, err := fetcher.Statistics(ctx, httpClient, promQuery, duration, endTime)
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
	_, _ = w.Write(b)
}
