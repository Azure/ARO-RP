package tasks

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"reflect"
	"testing"
	"time"

	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"

	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	configv1 "github.com/openshift/api/config/v1"

	"github.com/Azure/ARO-RP/pkg/api"
	mimoconst "github.com/Azure/ARO-RP/pkg/mimo"
	"github.com/Azure/ARO-RP/pkg/operator"
	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	"github.com/Azure/ARO-RP/pkg/util/clienthelper"
	testdatabase "github.com/Azure/ARO-RP/test/database"
	testtasks "github.com/Azure/ARO-RP/test/mimo/tasks"
	testclienthelper "github.com/Azure/ARO-RP/test/util/clienthelper"
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
				Key: "/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/resourcegroup/providers/microsoft.redhatopenshift/openshiftclusters/cluster",
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

func TestSetGenevaLoggingOTelProfileInClusterDocInitializesNilOperatorFlags(t *testing.T) {
	g := NewWithT(t)
	ctx := context.Background()
	_, log := testlog.New()

	dbOpenShiftClusters, _ := testdatabase.NewFakeOpenShiftClusters()
	doc := &api.OpenShiftClusterDocument{
		Key: "/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/resourcegroup/providers/microsoft.redhatopenshift/openshiftclusters/cluster",
		OpenShiftCluster: &api.OpenShiftCluster{
			Properties: api.OpenShiftClusterProperties{},
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

	err = setGenevaLoggingOTelProfileInClusterDoc(tc, operator.GenevaLoggingOTelProfileMinimalLogs)
	g.Expect(err).ToNot(HaveOccurred())

	updatedDoc, err := dbOpenShiftClusters.Get(ctx, doc.Key)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(updatedDoc.OpenShiftCluster.Properties.OperatorFlags[operator.GenevaLoggingEnabled]).To(Equal(operator.FlagTrue))
	g.Expect(updatedDoc.OpenShiftCluster.Properties.OperatorFlags[operator.GenevaLoggingOTelProfile]).To(Equal(operator.GenevaLoggingOTelProfileMinimalLogs))
	g.Expect(updatedDoc.OpenShiftCluster.Properties.OperatorFlags[operator.GenevaLoggingOTelMasterProfile]).To(Equal(operator.GenevaLoggingOTelProfileMinimalLogs))
	g.Expect(updatedDoc.OpenShiftCluster.Properties.OperatorFlags[operator.GenevaLoggingOTelWorkerProfile]).To(Equal(operator.GenevaLoggingOTelProfileMinimalLogs))
}

func TestSetOperatorFlagGenevaLoggingOTelProfileMaxLogsUpdatesClusterSpec(t *testing.T) {
	g := NewWithT(t)
	ctx := context.Background()
	_, log := testlog.New()

	dbOpenShiftClusters, _ := testdatabase.NewFakeOpenShiftClusters()
	doc := &api.OpenShiftClusterDocument{
		Key: "/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/resourcegroup/providers/microsoft.redhatopenshift/openshiftclusters/cluster",
		OpenShiftCluster: &api.OpenShiftCluster{
			Properties: api.OpenShiftClusterProperties{},
		},
	}

	fixture := testdatabase.NewFixture().WithOpenShiftClusters(dbOpenShiftClusters)
	fixture.AddOpenShiftClusterDocuments(doc)
	err := fixture.Create()
	g.Expect(err).ToNot(HaveOccurred())

	clusterObjects := []runtime.Object{
		&configv1.ClusterOperator{
			ObjectMeta: metav1.ObjectMeta{
				Name: "kube-apiserver",
			},
			Status: configv1.ClusterOperatorStatus{
				Conditions: []configv1.ClusterOperatorStatusCondition{
					{
						Type:   configv1.OperatorAvailable,
						Status: configv1.ConditionTrue,
					},
					{
						Type:   configv1.OperatorProgressing,
						Status: configv1.ConditionFalse,
					},
				},
			},
		},
		&arov1alpha1.Cluster{
			ObjectMeta: metav1.ObjectMeta{
				Name: arov1alpha1.SingletonClusterName,
			},
		},
	}

	ch := clienthelper.NewWithClient(log, testclienthelper.NewHookingClient(fake.NewClientBuilder().WithRuntimeObjects(clusterObjects...).Build()))
	tc := testtasks.NewFakeTestContext(
		ctx, nil, log, func() time.Time { return time.Unix(100, 0) },
		testtasks.WithClientHelper(ch),
		testtasks.WithOpenShiftDatabase(dbOpenShiftClusters),
		testtasks.WithOpenShiftClusterDocument(doc),
	)

	err = SetOperatorFlagGenevaLoggingOTelProfileMaxLogs(tc, nil, nil)
	g.Expect(err).ToNot(HaveOccurred())

	clusterObj := &arov1alpha1.Cluster{}
	err = ch.GetOne(ctx, types.NamespacedName{Name: arov1alpha1.SingletonClusterName}, clusterObj)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(clusterObj.Spec.OperatorFlags[operator.GenevaLoggingEnabled]).To(Equal(operator.FlagTrue))
	g.Expect(clusterObj.Spec.OperatorFlags[operator.GenevaLoggingOTelProfile]).To(Equal(operator.GenevaLoggingOTelProfileMaxLogs))
	g.Expect(clusterObj.Spec.OperatorFlags[operator.GenevaLoggingOTelMasterProfile]).To(Equal(operator.GenevaLoggingOTelProfileMaxLogs))
	g.Expect(clusterObj.Spec.OperatorFlags[operator.GenevaLoggingOTelWorkerProfile]).To(Equal(operator.GenevaLoggingOTelProfileMaxLogs))
}
