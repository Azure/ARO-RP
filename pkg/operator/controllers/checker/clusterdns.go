// Implements a check that provides detail on openshift-dns-operator
// configurations.
//
// Included checks are:
//  - existence of custom DNS entries
//  - malformed zone forwarding configuration

package checker

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"strings"

	operatorv1 "github.com/openshift/api/operator/v1"
	operatorclient "github.com/openshift/client-go/operator/clientset/versioned"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	aroclient "github.com/Azure/ARO-RP/pkg/operator/clientset/versioned"
	"github.com/Azure/ARO-RP/pkg/util/conditions"
)

type ClusterDNSChecker struct {
	log *logrus.Entry

	arocli      aroclient.Interface
	operatorcli operatorclient.Interface

	role string
}

func NewClusterDNSChecker(log *logrus.Entry, arocli aroclient.Interface, operatorcli operatorclient.Interface, role string) *ClusterDNSChecker {
	return &ClusterDNSChecker{
		log:         log,
		arocli:      arocli,
		operatorcli: operatorcli,
	}
}

func (r *ClusterDNSChecker) Name() string {
	return "ClusterDNSChecker"
}

func (r *ClusterDNSChecker) Check(ctx context.Context) error {
	cond := &operatorv1.OperatorCondition{
		Type:    arov1alpha1.DefaultClusterDNS,
		Status:  operatorv1.ConditionUnknown,
		Message: "",
		Reason:  "CheckDone",
	}

	dns, err := r.operatorcli.OperatorV1().DNSes().Get(ctx, "default", metav1.GetOptions{})
	if err != nil {
		cond.Message = err.Error()
		cond.Reason = "CheckFailed"
		return conditions.SetCondition(ctx, r.arocli, cond, r.role)
	}

	var upstreams []string
	for _, s := range dns.Spec.Servers {
		for _, z := range s.Zones {
			if z == "." {
				// If "." is set as a zone, bail out and warn about the
				// malformed config, as this will prevent CoreDNS from rolling
				// out
				cond.Message = `Malformed config: "." in zones`
				cond.Status = operatorv1.ConditionFalse
				return conditions.SetCondition(ctx, r.arocli, cond, r.role)
			}
		}

		upstreams = append(upstreams, s.ForwardPlugin.Upstreams...)
	}

	if len(upstreams) > 0 {
		cond.Status = operatorv1.ConditionFalse
		cond.Message = fmt.Sprintf("Custom upstream DNS servers in use: %s", strings.Join(upstreams, ", "))
	} else {
		cond.Status = operatorv1.ConditionTrue
		cond.Message = "No in-cluster upstream DNS servers"
	}

	return conditions.SetCondition(ctx, r.arocli, cond, r.role)
}
