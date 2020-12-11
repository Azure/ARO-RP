package audit

import (
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/golang/mock/gomock"
	"github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/util/audit/schema"
	mock_env "github.com/Azure/ARO-RP/pkg/util/mocks/env"
	utillog "github.com/Azure/ARO-RP/test/util/log"
)

func TestAudit(t *testing.T) {
	h, log := utillog.New()

	controller := gomock.NewController(t)
	defer controller.Finish()

	env := mock_env.NewMockInterface(controller)
	env.EXPECT().Environment().AnyTimes().Return(&azure.PublicCloud)
	env.EXPECT().Location().AnyTimes().Return("eastus")

	auditLog := &Log{
		AzureEnvironment: env.Environment().Name,
		CallerIdentities: []*schema.CallerIdentity{
			{
				CallerIdentityType:  schema.CallerIdentityTypePUID,
				CallerIdentityValue: strconv.Itoa(os.Getpid()),
			},
		},
		Category:      schema.CategoryAuthorization,
		OperationName: "initializeAuthorizers",
		Region:        env.Location(),
		TargetResources: []*schema.TargetResource{
			{
				TargetResourceType: "resource provider",
				TargetResourceName: "aro-rp",
			},
		},
	}

	err := EmitRPLog(*log, auditLog)
	if err != nil {
		t.Fatal(err)
	}

	err = utillog.AssertLoggingOutput(h, []map[string]types.GomegaMatcher{
		{
			"level":         gomega.Equal(logrus.InfoLevel),
			"msg":           gomega.Equal("see auditFullPayload field for full log data"),
			"auditCategory": gomega.BeEquivalentTo("Authorization"),
			"auditCreatedTime": gomega.WithTransform(
				func(s string) time.Time {
					t, err := time.Parse(time.RFC3339, s)
					if err != nil {
						panic(err)
					}
					return t
				},
				gomega.BeTemporally("~", time.Now(), time.Second),
			),
			// auditFullPayload: "{"env_ver":2.1,"env_name":"#Ifx.AuditSchema","env_time":"2020-12-11T13:47:27Z","env_epoch":"ab34cafa-b047-4dc4-a8ff-e24b7f854b4d","env_seqNum":1,"env_popSample":0,"env_iKey":null,"env_flags":257,"env_cv":"","env_os":"linux","env_osVer":null,"env_appId":null,"env_appVer":null,"env_cloud_ver":1,"env_cloud_name":"AzurePublicCloud","env_cloud_role":"","env_cloud_roleVer":null,"env_cloud_roleInstance":"","env_cloud_environment":null,"env_cloud_location":"eastus","env_cloud_deploymentUnit":null,"CallerIdentities":[{"CallerDisplayName":"","CallerIdentityType":"PUID","CallerIdentityValue":"1261453","CallerIpAddress":""}],"Category":"Authorization","nCloud":"AzurePublicCloud","OperationName":"initializeAuthorizers","Result":{"ResultType":"","ResultDescription":""},"requestId":"","TargetResources":[{"TargetResourceType":"resource provider","TargetResourceName":"aro-rp"}]}"
			"auditOperation": gomega.Equal("initializeAuthorizers"),
			"auditResult":    gomega.Equal(""),
			"auditSource":    gomega.Equal("aro-rp"),
			"logKind":        gomega.Equal("ifxaudit"),
		},
	})
	if err != nil {
		t.Fatal(err)
	}
}
