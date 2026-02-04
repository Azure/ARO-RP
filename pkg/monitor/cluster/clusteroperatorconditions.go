package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"regexp"

	"github.com/sirupsen/logrus"

	"sigs.k8s.io/controller-runtime/pkg/client"

	configv1 "github.com/openshift/api/config/v1"
)

type clusterOperatorConditionsIgnoreRegexStruct struct {
	Name   *regexp.Regexp
	Type   *regexp.Regexp
	Status *regexp.Regexp
}
type clusterOperatorConditionsIgnoreStruct struct {
	Name   string
	Type   configv1.ClusterStatusConditionType
	Status configv1.ConditionStatus
}

// clusterOperatorConditionsIgnore contains list of statuses that we don't want to scrape
var clusterOperatorConditionsIgnore = map[clusterOperatorConditionsIgnoreStruct]struct{}{
	{"cloud-controller-manager", "TrustedCABundleControllerControllerDegraded", configv1.ConditionFalse}: {},
	{"cloud-controller-manager", "TrustedCABundleControllerControllerAvailable", configv1.ConditionTrue}: {},
	{"cloud-controller-manager", "CloudConfigControllerDegraded", configv1.ConditionFalse}:               {},
	{"cloud-controller-manager", "CloudConfigControllerUpgradeable", configv1.ConditionTrue}:             {},
	{"cloud-controller-manager", "CloudControllerOwner", configv1.ConditionTrue}:                         {},

	{"baremetal", "Disabled", configv1.ConditionTrue}:               {},
	{"network", "ManagementStateDegraded", configv1.ConditionFalse}: {},
}

// clusterOperatorConditionsIgnoreRegexes contains more complex checks
var clusterOperatorConditionsIgnoreRegexes = []clusterOperatorConditionsIgnoreRegexStruct{
	// Ignore all the Unknowns that we don't care about
	{regexp.MustCompile(".*"), regexp.MustCompile("(EvaluationConditionsDetected|Upgradeable)"), regexp.MustCompile(string(configv1.ConditionUnknown))},

	// Ignore all RecentBackup metrics
	{regexp.MustCompile("etcd"), regexp.MustCompile("RecentBackup"), regexp.MustCompile(".*")},

	// Ignore all Insights metrics (Available/Degraded gets caught before this check happens)
	{regexp.MustCompile("insights"), regexp.MustCompile(".*"), regexp.MustCompile(".*")},

	// Ignore cluster-api metrics that are almost certainly user-deployed and
	// won't be an OpenShift component
	{regexp.MustCompile("cluster-api"), regexp.MustCompile("(CapiInstaller|SecretSync|InfraCluster)ControllerAvailable"), regexp.MustCompile(".*")},
}

var clusterOperatorConditionsExpected = map[configv1.ClusterStatusConditionType]configv1.ConditionStatus{
	configv1.OperatorAvailable:            configv1.ConditionTrue,
	configv1.OperatorDegraded:             configv1.ConditionFalse,
	configv1.OperatorProgressing:          configv1.ConditionFalse,
	configv1.OperatorUpgradeable:          configv1.ConditionTrue,
	configv1.EvaluationConditionsDetected: configv1.ConditionFalse,
}

func (mon *Monitor) emitClusterOperatorConditions(ctx context.Context) error {
	var cont string
	l := &configv1.ClusterOperatorList{}
	count := 0

	for {
		err := mon.ocpclientset.List(ctx, l, client.Continue(cont), client.Limit(mon.queryLimit))
		if err != nil {
			return fmt.Errorf("error in ClusterOperator list operation: %w", err)
		}

		for _, co := range l.Items {
			for _, c := range co.Status.Conditions {
				if clusterOperatorConditionIsExpected(&co, &c) {
					continue
				}

				mon.emitGauge("clusteroperator.conditions", 1, map[string]string{
					"name":   co.Name,
					"status": string(c.Status),
					"type":   string(c.Type),
				})

				if mon.hourlyRun {
					mon.log.WithFields(logrus.Fields{
						"metric":  "clusteroperator.conditions",
						"name":    co.Name,
						"status":  c.Status,
						"type":    c.Type,
						"message": c.Message,
					}).Print()
				}
			}
		}

		count += len(l.Items)

		cont = l.Continue
		if cont == "" {
			break
		}
	}

	mon.emitGauge("clusteroperator.count", int64(count), nil)

	return nil
}

func clusterOperatorConditionIsExpected(co *configv1.ClusterOperator, c *configv1.ClusterOperatorStatusCondition) bool {
	// This ordering is so that the majority of conditions are simple lookups --
	// first we check for a known-expected value for all operators (e.g.
	// Available=True) first, then known-expected value for specific operators
	// (e.g. something=True for operator XYZ), and then check the regex filters.
	expected, ok := clusterOperatorConditionsExpected[c.Type]
	if !ok {
		if _, ok := clusterOperatorConditionsIgnore[clusterOperatorConditionsIgnoreStruct{
			Name:   co.Name,
			Type:   c.Type,
			Status: c.Status,
		}]; ok {
			return true
		}

		for _, i := range clusterOperatorConditionsIgnoreRegexes {
			if i.Status.MatchString(string(c.Status)) && i.Name.MatchString(co.Name) && i.Type.MatchString(string(c.Type)) {
				return true
			}
		}
	}

	return expected == c.Status
}
