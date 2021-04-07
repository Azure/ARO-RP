package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/common/model"
)

const promURLPrefix = "http://prometheus-k8s.openshift-monitoring.svc:9090/api/v1/"
const promQuerySuffix = "{endpoint=\"etcd-metrics\"}"

type promQueryResp struct {
	Status string            `json:"status"`
	Data   promQueryRespData `json:"data"`
}

type promQueryRangeResp struct {
	Status string                 `json:"status"`
	Data   promQueryRangeRespData `json:"data"`
}

type promQueryRespData struct {
	ResultType string
	Result     model.Vector `json:"result"`
}

type promQueryRangeRespData struct {
	ResultType string
	Result     []model.Matrix `json:"result"`
}

func promQueryURL(query string) string {
	return promURLPrefix + "query?query=" + query + promQuerySuffix
}

// promQueryRangeURL returns the URL for a range vector query, from current time to minutesAgo.
func promQueryRangeURL(query string, minutesAgo int64, stepSeconds int) string {
	now := time.Now().UTC()
	end := now.Format(time.RFC3339)
	start := now.Add(time.Minute * time.Duration(-1*minutesAgo)).Format(time.RFC3339)
	url := promURLPrefix + "query_range?query=" + query + promQuerySuffix
	return url + "&start=" + start + "&end=" + end + "&step=" + strconv.Itoa(stepSeconds) + "s"
}

func (mon *Monitor) emitEtcdHeapObjects(ctx context.Context) error {
	resp, err := mon.requestMetricHTTP(ctx, "prometheus-k8s", promQueryURL("go_memstats_heap_objects"))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code %d", resp.StatusCode)
	}

	var queryResp promQueryResp
	err = json.NewDecoder(resp.Body).Decode(&queryResp)
	if err != nil {
		return err
	}

	var total model.SampleValue
	for _, sample := range queryResp.Data.Result {
		total += sample.Value
	}

	mon.emitGauge("prometheus.metrics.etcdheapobjects", int64(total), nil)
	return nil
}

func (mon *Monitor) emitEtcdHeapAllocBytes(ctx context.Context) error {
	resp, err := mon.requestMetricHTTP(ctx, "prometheus-k8s", promQueryURL("go_memstats_heap_alloc_bytes"))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code %d", resp.StatusCode)
	}

	var queryResp promQueryResp
	err = json.NewDecoder(resp.Body).Decode(&queryResp)
	if err != nil {
		return err
	}

	var total model.SampleValue
	for _, sample := range queryResp.Data.Result {
		total += sample.Value
	}

	mon.emitGauge("prometheus.metrics.etcdheapallocbytes", int64(total), nil)
	return nil
}

func (mon *Monitor) emitEtcdHasLeader(ctx context.Context) error {
	resp, err := mon.requestMetricHTTP(ctx, "prometheus-k8s", promQueryURL("etcd_server_has_leader"))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code %d", resp.StatusCode)
	}

	var queryResp promQueryResp
	err = json.NewDecoder(resp.Body).Decode(&queryResp)
	if err != nil {
		return err
	}

	// should be same value for every pod, so just emit first result in list
	var result model.SampleValue
	for _, sample := range queryResp.Data.Result {
		result = sample.Value
		break
	}

	mon.emitGauge("prometheus.metrics.etcdhasleader", int64(result), nil)
	return nil
}

func (mon *Monitor) emitEtcdLeaderChanges(ctx context.Context) error {
	resp, err := mon.requestMetricHTTP(ctx, "prometheus-k8s", promQueryURL("etcd_server_leader_changes_seen_total"))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code %d", resp.StatusCode)
	}

	var queryResp promQueryResp
	err = json.NewDecoder(resp.Body).Decode(&queryResp)
	if err != nil {
		return err
	}

	// not sure why, but value can differ between pods.
	// assuming the highest number is correct.
	var max model.SampleValue
	for _, sample := range queryResp.Data.Result {
		if max < sample.Value {
			max = sample.Value
		}
	}

	mon.emitGauge("prometheus.metrics.etcdleaderchanges", int64(max), nil)
	return nil
}

// TODO: need to find a way to call prometheus `rate` function
func (mon *Monitor) emitEtcdFsyncDurationSeconds(ctx context.Context) error {
	minutesAgo := int64(5)
	stepSeconds := 15
	for _, m := range []struct {
		promName   string
		statsdName string
	}{
		{
			"etcd_disk_wal_fsync_duration_seconds_sum",
			"etcdfsyncdurationsum",
		},
		{
			"etcd_disk_wal_fsync_duration_seconds_count",
			"etcdfsyncdurationcount",
		},
	} {
		resp, err := mon.requestMetricHTTP(ctx, "prometheus-k8s", promQueryRangeURL(m.promName, minutesAgo, stepSeconds))
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("unexpected status code %d", resp.StatusCode)
		}

		bodyBytes, _ := ioutil.ReadAll(resp.Body)
		bodyString := string(bodyBytes)
		mon.log.Info(bodyString)

		// var queryResp promQueryRangeResp
		// err = json.NewDecoder(resp.Body).Decode(&queryResp)
		// if err != nil {
		// 	return err
		// }

		// var total model.SampleValue
		// for _, sample := range queryResp.Data.Result {
		// 	total += sample.Value
		// }

		// mon.emitGauge("prometheus.metrics."+m.statsdName, int64(total), nil)
	}
	return nil
}
