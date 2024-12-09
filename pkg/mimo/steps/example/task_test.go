package example

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"testing"
	"time"

	. "github.com/onsi/gomega"

	configv1 "github.com/openshift/api/config/v1"
	"go.uber.org/mock/gomock"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/Azure/ARO-RP/pkg/util/clienthelper"
	mock_env "github.com/Azure/ARO-RP/pkg/util/mocks/env"
	testtasks "github.com/Azure/ARO-RP/test/mimo/tasks"
	testclienthelper "github.com/Azure/ARO-RP/test/util/clienthelper"
	testlog "github.com/Azure/ARO-RP/test/util/log"
)

func TestTask(t *testing.T) {
	RegisterTestingT(t)
	ctx := context.Background()

	controller := gomock.NewController(t)
	_env := mock_env.NewMockInterface(controller)
	_, log := testlog.New()

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
	tc := testtasks.NewFakeTestContext(
		ctx, _env, log, func() time.Time { return time.Unix(100, 0) },
		testtasks.WithClientHelper(ch),
	)
	err := ReportClusterVersion(tc)
	Expect(err).ToNot(HaveOccurred())
	Expect(tc.GetResultMessage()).To(Equal("cluster version is: 4.99.123"))
}
