package cluster_test

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"reflect"
	"testing"

	"github.com/spf13/viper"

	"github.com/Azure/ARO-RP/pkg/api/util/vms"
	"github.com/Azure/ARO-RP/pkg/util/cluster"
)

func TestNewClusterConfigFromEnvDefaultsCICandidateVMSizes(t *testing.T) {
	viper.Reset()
	t.Cleanup(viper.Reset)

	t.Setenv("CLUSTER", "test-cluster")
	t.Setenv("RESOURCEGROUP", "rp-rg")
	t.Setenv("AZURE_FP_SERVICE_PRINCIPAL_ID", "fp-sp")
	t.Setenv("CI", "true")

	conf, err := cluster.NewClusterConfigFromEnv()
	if err != nil {
		t.Fatal(err)
	}

	if conf.VnetResourceGroup != conf.ClusterName {
		t.Fatalf("VnetResourceGroup=%q, want cluster name %q", conf.VnetResourceGroup, conf.ClusterName)
	}

	if !sameVMSizeSet(conf.CandidateMasterVMSizes, vms.GetCICandidateMasterVMSizes()) {
		t.Fatalf("unexpected master candidates: %v", conf.CandidateMasterVMSizes)
	}

	if !sameVMSizeSet(conf.CandidateWorkerVMSizes, vms.GetCICandidateWorkerVMSizes()) {
		t.Fatalf("unexpected worker candidates: %v", conf.CandidateWorkerVMSizes)
	}
}

func TestNewClusterConfigFromEnvUsesExplicitVMSizesAsCandidates(t *testing.T) {
	viper.Reset()
	t.Cleanup(viper.Reset)

	t.Setenv("CLUSTER", "test-cluster")
	t.Setenv("RESOURCEGROUP", "rp-rg")
	t.Setenv("AZURE_FP_SERVICE_PRINCIPAL_ID", "fp-sp")
	t.Setenv("CI", "true")
	t.Setenv("MASTER_VM_SIZE", string(vms.VMSizeStandardD8sV4))
	t.Setenv("WORKER_VM_SIZE", string(vms.VMSizeStandardD4sV4))

	conf, err := cluster.NewClusterConfigFromEnv()
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(conf.CandidateMasterVMSizes, []vms.VMSize{vms.VMSizeStandardD8sV4}) {
		t.Fatalf("CandidateMasterVMSizes=%v, want [%s]", conf.CandidateMasterVMSizes, vms.VMSizeStandardD8sV4)
	}

	if !reflect.DeepEqual(conf.CandidateWorkerVMSizes, []vms.VMSize{vms.VMSizeStandardD4sV4}) {
		t.Fatalf("CandidateWorkerVMSizes=%v, want [%s]", conf.CandidateWorkerVMSizes, vms.VMSizeStandardD4sV4)
	}
}

func TestNewClusterConfigFromEnvRequiresWorkloadIdentityRoleSets(t *testing.T) {
	viper.Reset()
	t.Cleanup(viper.Reset)

	t.Setenv("CLUSTER", "test-cluster")
	t.Setenv("RESOURCEGROUP", "rp-rg")
	t.Setenv("AZURE_FP_SERVICE_PRINCIPAL_ID", "fp-sp")
	t.Setenv("USE_WI", "true")

	_, err := cluster.NewClusterConfigFromEnv()
	if err == nil || err.Error() != "workload Identity Role Set must be set" {
		t.Fatalf("got err %v, want workload identity role set validation", err)
	}
}

func sameVMSizeSet(a, b []vms.VMSize) bool {
	if len(a) != len(b) {
		return false
	}

	counts := map[vms.VMSize]int{}
	for _, size := range a {
		counts[size]++
	}

	for _, size := range b {
		counts[size]--
		if counts[size] < 0 {
			return false
		}
	}

	for _, count := range counts {
		if count != 0 {
			return false
		}
	}

	return true
}
