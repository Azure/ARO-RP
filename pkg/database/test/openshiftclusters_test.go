package test

import (
	"context"
	"strings"
	"testing"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/database"
	"github.com/Azure/ARO-RP/pkg/database/cosmosdb"
	testdatabase "github.com/Azure/ARO-RP/test/database"
)

func TestSecretVersion(t *testing.T) {
	ctx := context.Background()
	mockSubID := "00000000-0000-0000-0000-000000000000"

	aead := testdatabase.NewFakeAEAD("newversion")

	testCases := []struct {
		desc        string
		doc         *api.OpenShiftClusterDocument
		expectedDoc *api.OpenShiftClusterDocument
	}{
		{
			desc: "secretversion is set",
			doc: &api.OpenShiftClusterDocument{
				ID:           "00000000-2222-0000-0000-000000000000",
				PartitionKey: mockSubID,
				Key:          strings.ToLower(testdatabase.GetResourcePath(mockSubID, "resourceName")),
				OpenShiftCluster: &api.OpenShiftCluster{
					ID:   testdatabase.GetResourcePath(mockSubID, "resourceName"),
					Name: "resourceName",
					Type: "Microsoft.RedHatOpenShift/openshiftClusters",
					Properties: api.OpenShiftClusterProperties{
						KubeadminPassword: api.SecureString("FAKE123"),
					},
				},
			},
			expectedDoc: &api.OpenShiftClusterDocument{
				ID:            "00000000-2222-0000-0000-000000000000",
				Key:           strings.ToLower(testdatabase.GetResourcePath(mockSubID, "resourceName")),
				PartitionKey:  mockSubID,
				SecretVersion: "newversion",
				OpenShiftCluster: &api.OpenShiftCluster{
					ID:   testdatabase.GetResourcePath(mockSubID, "resourceName"),
					Name: "resourceName",
					Type: "Microsoft.RedHatOpenShift/openshiftClusters",
					Properties: api.OpenShiftClusterProperties{
						KubeadminPassword: api.SecureString("FAKE123"),
						ProvisionedBy:     "new",
					},
				},
			},
		},
		{
			desc: "existing secret version is updated",
			doc: &api.OpenShiftClusterDocument{
				ID:            "00000000-2222-0000-0000-000000000000",
				PartitionKey:  mockSubID,
				Key:           strings.ToLower(testdatabase.GetResourcePath(mockSubID, "resourceName")),
				SecretVersion: "oldversion",
				OpenShiftCluster: &api.OpenShiftCluster{
					ID:   testdatabase.GetResourcePath(mockSubID, "resourceName"),
					Name: "resourceName",
					Type: "Microsoft.RedHatOpenShift/openshiftClusters",
					Properties: api.OpenShiftClusterProperties{
						KubeadminPassword: api.SecureString("FAKE123"),
					},
				},
			},
			expectedDoc: &api.OpenShiftClusterDocument{
				ID:            "00000000-2222-0000-0000-000000000000",
				Key:           strings.ToLower(testdatabase.GetResourcePath(mockSubID, "resourceName")),
				PartitionKey:  mockSubID,
				SecretVersion: "newversion",
				OpenShiftCluster: &api.OpenShiftCluster{
					ID:   testdatabase.GetResourcePath(mockSubID, "resourceName"),
					Name: "resourceName",
					Type: "Microsoft.RedHatOpenShift/openshiftClusters",
					Properties: api.OpenShiftClusterProperties{
						KubeadminPassword: api.SecureString("FAKE123"),
						ProvisionedBy:     "new",
					},
				},
			},
		},
	}
	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {
			jsonHandle, err := database.NewJSONHandle(aead)
			if err != nil {
				t.Fatal(err)
			}

			for _, isPatch := range []bool{false, true} {
				db, client := testdatabase.NewFakeOpenShiftClustersWithProvidedJSONHandle(jsonHandle, aead)
				doc, err := client.Create(ctx, mockSubID, tC.doc, &cosmosdb.Options{})
				if err != nil {
					t.Fatal(isPatch, err)
				}

				if isPatch {
					_, err = db.Patch(ctx, strings.ToLower(testdatabase.GetResourcePath(mockSubID, "resourceName")), func(oscd *api.OpenShiftClusterDocument) error {
						oscd.OpenShiftCluster.Properties.ProvisionedBy = "new"
						return nil
					})
					if err != nil {
						t.Fatal(isPatch, err)
					}
				} else {
					doc.OpenShiftCluster.Properties.ProvisionedBy = "new"
					_, err = db.Update(ctx, doc)
					if err != nil {
						t.Fatal(isPatch, err)
					}
				}

				checker := testdatabase.NewChecker()
				checker.AddOpenShiftClusterDocuments(tC.expectedDoc)
				errs := checker.CheckOpenShiftClusters(client)
				for _, err := range errs {
					t.Error(isPatch, err)
				}
			}
		})
	}
}
