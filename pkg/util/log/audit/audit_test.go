package audit

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"testing"
	"time"

	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/golang/mock/gomock"
	"github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
	uuid "github.com/satori/go.uuid"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"

	mock_env "github.com/Azure/ARO-RP/pkg/util/mocks/env"
	utillog "github.com/Azure/ARO-RP/test/util/log"
)

func TestAudit(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	env := mock_env.NewMockInterface(controller)
	env.EXPECT().Environment().AnyTimes().Return(&azure.PublicCloud)
	env.EXPECT().Location().AnyTimes().Return("eastus")
	env.EXPECT().Hostname().AnyTimes().Return("test-host")

	epoch = uuid.NewV4().String()
	var (
		now          = time.Now().UTC()
		formattedNow = now.Format(time.RFC3339)
		logger, h    = test.NewNullLogger()
	)

	auditLog := NewEntry(env, logger)
	auditLog.WithFields(logrus.Fields{
		PayloadKeyCategory:      CategoryAuthorization,
		PayloadKeyOperationName: "initializeAuthorizers",
		MetadataSource:          SourceRP,
		MetadataCreatedTime:     formattedNow,
	}).Print("see auditFullPayload field for full log data")

	if err := utillog.AssertLoggingOutput(h, []map[string]types.GomegaMatcher{
		{
			"level":         gomega.Equal(logrus.InfoLevel),
			"msg":           gomega.Equal("see auditFullPayload field for full log data"),
			MetadataSource:  gomega.Equal("aro-rp"),
			MetadataLogKind: gomega.Equal("ifxaudit"),
			MetadataCreatedTime: gomega.WithTransform(
				func(s string) time.Time {
					t, err := time.Parse(time.RFC3339, s)
					if err != nil {
						panic(err)
					}
					return t
				},
				gomega.BeTemporally("~", now, time.Second),
			),
			MetadataPayload: gomega.Equal(`{"env_ver":2.1,"env_name":"#Ifx.AuditSchema","env_time":"` + formattedNow + `","env_epoch":"` + epoch + `","env_seqNum":1,"env_flags":257,"env_appId":"","env_cloud_name":"AzurePublicCloud","env_cloud_role":"","env_cloud_roleInstance":"test-host","env_cloud_environment":"AzurePublicCloud","env_cloud_location":"eastus","env_cloud_ver":1,"CallerIdentities":null,"Category":"Authorization","OperationName":"initializeAuthorizers","Result":null,"requestId":"","TargetResources":null}`),
		},
	}); err != nil {
		t.Error(err)
	}
}
