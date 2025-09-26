package monitor

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"
	"sync"
	"testing"
	"time"

	clientcmdv1 "k8s.io/client-go/tools/clientcmd/api/v1"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/util/bucket"
	"github.com/Azure/ARO-RP/pkg/util/pointerutils"
	"github.com/Azure/ARO-RP/pkg/util/uuid"
)

var fakeBucketAllocator = bucket.Random{}
var fakeDefaultLocation = "centralus"

func TestMonitor(t *testing.T) {
	numWorker := 3

	// Setup test environment
	env := SetupTestEnvironment(t)
	defer env.Cleanup()

	// Create multiple monitors for worker testing
	workers := make([]Runnable, numWorker)
	for i := 0; i < numWorker; i++ {
		mon := env.CreateTestMonitor(fmt.Sprintf("worker-%d", i))
		workers[i] = mon
	}

	for i := 0; i < 10; i++ {
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
		fakeClusterVisitMonitoringAttempts[clusterDoc.ResourceID] = pointerutils.ToPtr(0)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	wg := sync.WaitGroup{}

	for _, mon := range workers {
		wg.Add(1)
		go func() {
			err := mon.Run(ctx)
			if err != nil && err != context.DeadlineExceeded {
				t.Logf("Unexpected error: %v", err)
			}
			wg.Done()
		}()
	}

	time.Sleep(5 * time.Second)
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
	fakeClusterVisitMonitoringAttempts[clusterDoc.ResourceID] = pointerutils.ToPtr(0)

	wg.Wait()

	for k, v := range fakeClusterVisitMonitoringAttempts {
		if *v < 1 {
			t.Errorf("Expected that cluster %s got visits, but it got %v", k, v)
		}
	}

	if *fakeClusterVisitMonitoringAttempts[clusterDoc.ResourceID] < 1 {
		t.Errorf("Last added cluster %s didn't get any visit: %v", clusterDoc.ResourceID, fakeClusterVisitMonitoringAttempts[clusterDoc.ResourceID])
	}
}

func newFakeSubscription() *api.SubscriptionDocument {
	subID := uuid.DefaultGenerator.Generate()
	return &api.SubscriptionDocument{
		ID:         subID,
		ResourceID: subID,
		Metadata:   map[string]interface{}{},
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
		Metadata:      map[string]interface{}{},
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

func TestPointerStuff(t *testing.T) {
	t.Log("Starting")
	thingmap := map[string]*int{}
	one := 0
	two := 0
	thingmap["one"] = &one
	thingmap["two"] = &two
	wg := &sync.WaitGroup{}
	increase := func(incNum *int) {
		*incNum++
		wg.Done()
	}
	wg.Add(1)
	go increase(thingmap["one"])
	wg.Add(1)
	go increase(thingmap["two"])
	time.Sleep(time.Second)
	wg.Add(1)
	go increase(thingmap["one"])

	wg.Wait()
	for k, v := range thingmap {
		t.Logf("%s: %d", k, *v)
	}
	t.Fail()
}
