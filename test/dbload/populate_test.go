package dbload

import (
	"context"
	"crypto/rand"
	"math/big"
	"sync"
	"testing"

	uuid "github.com/satori/go.uuid"

	"github.com/Azure/ARO-RP/pkg/api"
	utillog "github.com/Azure/ARO-RP/pkg/util/log"
)

func TestPopulate(t *testing.T) {
	ctx := context.Background()
	log := utillog.GetLogger()

	c, err := get(ctx, log)
	if err != nil {
		t.Fatal(err)
	}

	ch := make(chan *api.OpenShiftClusterDocument)

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for doc := range ch {
				_, err = c.Create(ctx, doc.PartitionKey, doc, nil)
				if err != nil {
					t.Log(err)
				}
			}
		}()
	}

	for sub := 0; sub < 500; sub++ {
		subscriptionID := uuid.NewV4().String()
		for cluster := 0; cluster < 10; cluster++ {
			resourceGroup, err := randomLowerCaseAlphanumericString(8)
			if err != nil {
				t.Fatal(err)
			}

			resourceName, err := randomLowerCaseAlphanumericString(8)
			if err != nil {
				t.Fatal(err)
			}

			i, err := rand.Int(rand.Reader, big.NewInt(10))
			if err != nil {
				t.Fatal(err)
			}

			provisioningState := api.ProvisioningStateSucceeded
			if i.Int64() == 9 {
				provisioningState = api.ProvisioningStateCreating
			}

			i, err = rand.Int(rand.Reader, big.NewInt(1000000))
			if err != nil {
				t.Fatal(err)
			}

			ch <- &api.OpenShiftClusterDocument{
				ID:                        uuid.NewV4().String(),
				Key:                       "/subscriptions/" + subscriptionID + "/resourcegroups/" + resourceGroup + "/providers/microsoft.redhatopenshift/openshiftclusters/" + resourceName,
				PartitionKey:              subscriptionID,
				ClusterResourceGroupIDKey: "/subscriptions/" + subscriptionID + "/resourcegroups/" + resourceGroup,
				LeaseExpires:              int(i.Int64()),
				OpenShiftCluster: &api.OpenShiftCluster{
					ID:       "/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.RedHatOpenshift/openshiftClusters/" + resourceName,
					Name:     resourceName,
					Type:     "Microsoft.RedHatOpenShift/OpenShiftClusters",
					Location: "location",
					Properties: api.Properties{
						ProvisioningState: provisioningState,
						ClusterProfile: api.ClusterProfile{
							Domain:          resourceName,
							Version:         "4.3.0",
							ResourceGroupID: "/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup,
						},
						ServicePrincipalProfile: api.ServicePrincipalProfile{
							ClientSecret: "clientSecret",
							ClientID:     "clientId",
						},
						NetworkProfile: api.NetworkProfile{
							PodCIDR:     "10.128.0.0/14",
							ServiceCIDR: "172.30.0.0/16",
						},
						MasterProfile: api.MasterProfile{
							VMSize:   api.VMSizeStandardD8sV3,
							SubnetID: "/subscriptions/subscriptionId/resourceGroups/" + resourceGroup + "/providers/Microsoft.Network/virtualNetworks/vnet/subnets/master",
						},
						WorkerProfiles: []api.WorkerProfile{
							{
								Name:       "worker",
								VMSize:     api.VMSizeStandardD2sV3,
								DiskSizeGB: 128,
								SubnetID:   "/subscriptions/subscriptionId/resourceGroups/" + resourceGroup + "/providers/Microsoft.Network/virtualNetworks/vnet/subnets/worker",
								Count:      3,
							},
						},
						APIServerProfile: api.APIServerProfile{
							Visibility: api.VisibilityPublic,
							URL:        "https://api." + resourceName + ".location.aroapp.io:6443/",
							IP:         "1.2.3.4",
						},
						IngressProfiles: []api.IngressProfile{
							{
								Name:       "default",
								Visibility: api.VisibilityPublic,
								IP:         "1.2.3.4",
							},
						},
						ConsoleProfile: api.ConsoleProfile{
							URL: "https://console-openshift-console.apps." + resourceName + ".location.aroapp.io/",
						},
					},
				},
			}
		}
	}

	close(ch)
	wg.Wait()
}

func randomLowerCaseAlphanumericString(n int) (string, error) {
	return randomString("abcdefghijklmnopqrstuvwxyz0123456789", n)
}

func randomString(letterBytes string, n int) (string, error) {
	b := make([]byte, n)
	for i := range b {
		o, err := rand.Int(rand.Reader, big.NewInt(int64(len(letterBytes))))
		if err != nil {
			return "", err
		}
		b[i] = letterBytes[o.Int64()]
	}

	return string(b), nil
}
