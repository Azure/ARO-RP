package tasks

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"reflect"
	"testing"
	"time"

	. "github.com/onsi/gomega"

	"github.com/Azure/ARO-RP/pkg/api"
	mimoconst "github.com/Azure/ARO-RP/pkg/mimo"
	"github.com/Azure/ARO-RP/pkg/operator"
	testdatabase "github.com/Azure/ARO-RP/test/database"
	testtasks "github.com/Azure/ARO-RP/test/mimo/tasks"
	testlog "github.com/Azure/ARO-RP/test/util/log"
)

func TestDefaultMaintenanceTasksIncludeGenevaOTelProfileTasks(t *testing.T) {
	g := NewWithT(t)

	for _, tt := range []struct {
		name   string
		taskID api.MIMOTaskID
		fn     MaintenanceTask
	}{
		{
			name:   "default geneva otel",
			taskID: mimoconst.OPERATOR_FLAG_SET_GENEVA_OTEL,
			fn:     SetOperatorFlagGenevaLoggingUseOTel,
		},
		{
			name:   "max logs profile",
			taskID: mimoconst.OPERATOR_FLAG_SET_GENEVA_OTEL_PROFILE_MAX_LOGS,
			fn:     SetOperatorFlagGenevaLoggingOTelProfileMaxLogs,
		},
		{
			name:   "reduced logs profile",
			taskID: mimoconst.OPERATOR_FLAG_SET_GENEVA_OTEL_PROFILE_REDUCED_LOGS,
			fn:     SetOperatorFlagGenevaLoggingOTelProfileReducedLogs,
		},
		{
			name:   "minimal logs profile",
			taskID: mimoconst.OPERATOR_FLAG_SET_GENEVA_OTEL_PROFILE_MINIMAL_LOGS,
			fn:     SetOperatorFlagGenevaLoggingOTelProfileMinimalLogs,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			taskFn, ok := DEFAULT_MAINTENANCE_TASKS[tt.taskID]
			g.Expect(ok).To(BeTrue())
			g.Expect(reflect.ValueOf(taskFn).Pointer()).To(Equal(reflect.ValueOf(tt.fn).Pointer()))
		})
	}
}

func TestSetGenevaLoggingOTelProfileInClusterDoc(t *testing.T) {
	ctx := context.Background()

	for _, tt := range []struct {
		name            string
		profile         string
		expectedProfile string
	}{
		{
			name:            "max logs",
			profile:         operator.GenevaLoggingOTelProfileMaxLogs,
			expectedProfile: operator.GenevaLoggingOTelProfileMaxLogs,
		},
		{
			name:            "reduced logs",
			profile:         operator.GenevaLoggingOTelProfileReducedLogs,
			expectedProfile: operator.GenevaLoggingOTelProfileReducedLogs,
		},
		{
			name:            "minimal logs",
			profile:         operator.GenevaLoggingOTelProfileMinimalLogs,
			expectedProfile: operator.GenevaLoggingOTelProfileMinimalLogs,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)
			_, log := testlog.New()

			dbOpenShiftClusters, _ := testdatabase.NewFakeOpenShiftClusters()
			doc := &api.OpenShiftClusterDocument{
				Key: "cluster-key",
				OpenShiftCluster: &api.OpenShiftCluster{
					Properties: api.OpenShiftClusterProperties{
						OperatorFlags: api.OperatorFlags{
							"existing-flag": "existing-value",
						},
					},
				},
			}

			fixture := testdatabase.NewFixture().WithOpenShiftClusters(dbOpenShiftClusters)
			fixture.AddOpenShiftClusterDocuments(doc)
			err := fixture.Create()
			g.Expect(err).ToNot(HaveOccurred())

			tc := testtasks.NewFakeTestContext(
				ctx, nil, log, func() time.Time { return time.Unix(100, 0) },
				testtasks.WithOpenShiftDatabase(dbOpenShiftClusters),
				testtasks.WithOpenShiftClusterDocument(doc),
			)

			err = setGenevaLoggingOTelProfileInClusterDoc(tc, tt.profile)
			g.Expect(err).ToNot(HaveOccurred())

			updatedDoc, err := dbOpenShiftClusters.Get(ctx, doc.Key)
			g.Expect(err).ToNot(HaveOccurred())
			g.Expect(updatedDoc.OpenShiftCluster.Properties.OperatorFlags["existing-flag"]).To(Equal("existing-value"))
			g.Expect(updatedDoc.OpenShiftCluster.Properties.OperatorFlags[operator.GenevaLoggingEnabled]).To(Equal(operator.FlagTrue))
			g.Expect(updatedDoc.OpenShiftCluster.Properties.OperatorFlags[operator.GenevaLoggingOTelProfile]).To(Equal(tt.expectedProfile))
			g.Expect(updatedDoc.OpenShiftCluster.Properties.OperatorFlags[operator.GenevaLoggingOTelMasterProfile]).To(Equal(tt.expectedProfile))
			g.Expect(updatedDoc.OpenShiftCluster.Properties.OperatorFlags[operator.GenevaLoggingOTelWorkerProfile]).To(Equal(tt.expectedProfile))
		})
	}
}
