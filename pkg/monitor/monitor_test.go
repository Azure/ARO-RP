package monitor

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"math/big"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	clientcmdv1 "k8s.io/client-go/tools/clientcmd/api/v1"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/util/bucket"
	"github.com/Azure/ARO-RP/pkg/util/uuid"
)

var (
	fakeBucketAllocator = bucket.Random{}
	fakeDefaultLocation = "centralus"
)

func TestMonitor(t *testing.T) {
	numWorker := 3

	// Setup test environment
	env := SetupTestEnvironment(t)
	defer env.Cleanup()

	// Create multiple monitors for worker testing (simulating three VMSSes running workers)
	workers := make([]*monitor, numWorker)
	for i := range numWorker {
		mon := env.CreateTestMonitor(fmt.Sprintf("worker-%d", i))
		workers[i] = mon
	}

	for range 10 {
		subDoc := newFakeSubscription()
		clusterDoc := newFakeCluster(subDoc.ResourceID)
		_, err := env.OpenShiftClusterDB.Create(context.Background(), clusterDoc)
		if err != nil {
			t.Errorf("Couldn't create new test cluster doc: %v", err)
			t.FailNow()
		}
		_, err = env.SubscriptionsDB.Create(context.Background(), subDoc)
		if err != nil {
			t.Errorf("Couldn't create new test cluster doc: %v", err)
			t.FailNow()
		}
		fakeClusterVisitMonitoringAttempts.Store(clusterDoc.ResourceID, &atomic.Int64{})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	wg := sync.WaitGroup{}
	t.Cleanup(func() {
		cancel()
		wg.Wait()
	})

	for _, mon := range workers {
		wg.Go(func() {
			err := mon.Run(ctx)
			if err != nil && err != context.DeadlineExceeded && err != context.Canceled {
				t.Logf("Unexpected error: %v", err)
			}
		})
	}

	// Wait for the workers to go ready
	require.Eventually(t, func() bool {
		ready := true
		for _, w := range workers {
			if !w.checkReady() {
				ready = false
			}
		}
		return ready
	}, time.Second*5, time.Millisecond*100, "workers did not go ready after 5s")

	// Buckets should be distributed amongst the workers
	buckets := []int{}
	for _, w := range workers {
		// bucketcount is the total number of buckets that should be across all
		// workers, each one should have less than that
		workerBuckets := w.clusters.workerPool.GetBuckets()
		require.Less(t, len(workerBuckets), w.bucketCount)
		buckets = append(buckets, workerBuckets...)
	}
	require.Len(t, buckets, 256)
	// Sort + compact to remove any dupes to ensure there isn't any
	slices.Sort(buckets)
	require.Len(t, slices.Compact(buckets), 256, "buckets contained duplicates")

	// add a new cluster
	subDoc := newFakeSubscription()
	clusterDoc := newFakeCluster(subDoc.ResourceID)
	_, err := env.OpenShiftClusterDB.Create(context.Background(), clusterDoc)
	if err != nil {
		t.Errorf("Couldn't create new test cluster doc: %v", err)
		t.FailNow()
	}
	_, err = env.SubscriptionsDB.Create(context.Background(), subDoc)
	if err != nil {
		t.Errorf("Couldn't create new test cluster doc: %v", err)
		t.FailNow()
	}
	fakeClusterVisitMonitoringAttempts.Store(clusterDoc.ResourceID, &atomic.Int64{})

	require.Eventually(t, func() bool {
		for _, v := range fakeClusterVisitMonitoringAttempts.All() {
			if v.Load() < 1 {
				// Cluster should have visits
				return false
			}
		}
		return true
	}, time.Second*5, time.Millisecond*100, "not all clusters were visited at least once")

	// The monitors should still be ready
	for _, w := range workers {
		require.True(t, w.checkReady(), "worker was not ready")
	}
}

func newFakeSubscription() *api.SubscriptionDocument {
	subID := uuid.DefaultGenerator.Generate()
	return &api.SubscriptionDocument{
		ID:         subID,
		ResourceID: subID,
		Metadata:   map[string]any{},
		Deleting:   false,
		Subscription: &api.Subscription{
			State: api.SubscriptionStateRegistered,
			Properties: &api.SubscriptionProperties{
				TenantID: uuid.DefaultGenerator.Generate(),
			},
		},
	}
}

func newFakeCluster(subscriptionID string) *api.OpenShiftClusterDocument {
	bucketNumber, _ := fakeBucketAllocator.Allocate()

	clusterResID := randomClusterResourceID(subscriptionID)
	lowercaseResourceID := strings.ToLower(clusterResID.String())

	kubeconf := clientcmdv1.Config{
		Clusters: []clientcmdv1.NamedCluster{{
			Name: clusterResID.Name,
			Cluster: clientcmdv1.Cluster{
				Server: "https://kubernetes:8443",
			},
		}},
		AuthInfos: []clientcmdv1.NamedAuthInfo{{
			Name: clusterResID.Name,
			AuthInfo: clientcmdv1.AuthInfo{
				Username: "user",
				Password: "pw",
			},
		}},
		Contexts: []clientcmdv1.NamedContext{{
			Name: clusterResID.Name,
			Context: clientcmdv1.Context{
				Cluster:   clusterResID.Name,
				AuthInfo:  clusterResID.Name,
				Namespace: "default",
			},
		}},
		CurrentContext: clusterResID.Name,
	}

	kubeconfbytes, _ := json.Marshal(kubeconf)

	return &api.OpenShiftClusterDocument{
		MissingFields: api.MissingFields{},
		ID:            uuid.DefaultGenerator.Generate(),
		ResourceID:    lowercaseResourceID,
		Metadata:      map[string]any{},
		Key:           lowercaseResourceID,
		Bucket:        bucketNumber,
		OpenShiftCluster: &api.OpenShiftCluster{
			ID:         lowercaseResourceID,
			Name:       clusterResID.Name,
			Type:       clusterResID.ResourceType.Namespace + "/" + clusterResID.ResourceType.Type,
			Location:   fakeDefaultLocation,
			SystemData: api.SystemData{},
			Properties: api.OpenShiftClusterProperties{
				ProvisioningState:       api.ProvisioningStateSucceeded,
				LastProvisioningState:   api.ProvisioningStateCreating,
				FailedProvisioningState: "",
				AdminKubeconfig:         []byte{},
				AROServiceKubeconfig:    kubeconfbytes,
				NetworkProfile: api.NetworkProfile{
					APIServerPrivateEndpointIP: "10.0.0.1",
				},
			},
		},
	}
}

func randomClusterResourceID(subscriptionID string) arm.ResourceID {
	if subscriptionID == "" {
		subscriptionID = uuid.DefaultGenerator.Generate()
	}

	resourceGroupName := fmt.Sprintf("rg-%s", randomString(6))
	clusterName := fmt.Sprintf("cl-%s", randomString(4))
	clusterID := fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.RedHatOpenShift/openShiftClusters/%s", subscriptionID, resourceGroupName, clusterName)

	resourceID, _ := arm.ParseResourceID(clusterID)
	return *resourceID
}

func randomString(n int) string {
	letters := "abcdfghjklmnpqrstvwxyz0123456789"
	b := make([]byte, n)
	for i := range b {
		o, _ := rand.Int(rand.Reader, big.NewInt(int64(len(letters))))
		b[i] = letters[o.Int64()]
	}

	return string(b)
}
