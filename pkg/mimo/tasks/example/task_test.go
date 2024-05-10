package example

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"strings"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/golang/mock/gomock"
	configv1 "github.com/openshift/api/config/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/util/clienthelper"
	mock_env "github.com/Azure/ARO-RP/pkg/util/mocks/env"
	testdatabase "github.com/Azure/ARO-RP/test/database"
	testtasks "github.com/Azure/ARO-RP/test/mimo/tasks"
	testclienthelper "github.com/Azure/ARO-RP/test/util/clienthelper"
	testlog "github.com/Azure/ARO-RP/test/util/log"
)

func TestTask(t *testing.T) {
	RegisterTestingT(t)
	ctx := context.Background()

	mockSubID := "00000000-0000-0000-0000-000000000000"
	clusterResourceID := fmt.Sprintf("/subscriptions/%s/resourcegroups/resourceGroup/providers/Microsoft.RedHatOpenShift/openShiftClusters/resourceName", mockSubID)

	controller := gomock.NewController(t)
	_env := mock_env.NewMockInterface(controller)
	_, log := testlog.New()

	fixtures := testdatabase.NewFixture()
	clusters, _ := testdatabase.NewFakeOpenShiftClusters()

	fixtures.AddOpenShiftClusterDocuments(&api.OpenShiftClusterDocument{
		Key: strings.ToLower(clusterResourceID),
		OpenShiftCluster: &api.OpenShiftCluster{
			ID: clusterResourceID,
		},
	})

	manifest := &api.MaintenanceManifestDocument{
		ClusterID: strings.ToLower(clusterResourceID),
		MaintenanceManifest: &api.MaintenanceManifest{
			State:     api.MaintenanceManifestStatePending,
			RunBefore: 60,
			RunAfter:  0,
		},
	}

	err := fixtures.WithOpenShiftClusters(clusters).Create()
	Expect(err).ToNot(HaveOccurred())

	builder := fake.NewClientBuilder().WithRuntimeObjects(
		&configv1.ClusterVersion{
			ObjectMeta: metav1.ObjectMeta{
				Name: "version",
			},
			Status: configv1.ClusterVersionStatus{
				History: []configv1.UpdateHistory{
					{
						State:   configv1.CompletedUpdate,
						Version: "4.99.123",
					},
				},
			},
		},
	)
	ch := clienthelper.NewWithClient(log, testclienthelper.NewHookingClient(builder.Build()))
	tc := testtasks.NewFakeTestContext(_env, log, ch)

	oc, err := clusters.Get(ctx, strings.ToLower(clusterResourceID))
	Expect(err).ToNot(HaveOccurred())
	r, text := ExampleTask(ctx, tc, manifest, oc)

	Expect(r).To(Equal(api.MaintenanceManifestStateCompleted))
	Expect(text).To(Equal("cluster version is: 4.99.123"))
}
