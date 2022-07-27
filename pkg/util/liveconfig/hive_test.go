package liveconfig

import (
	"context"
	"embed"
	"encoding/json"
	"testing"

	"github.com/golang/mock/gomock"

	mock_keyvault "github.com/Azure/ARO-RP/pkg/util/mocks/keyvault"
	"github.com/Azure/azure-sdk-for-go/services/keyvault/v7.0/keyvault"
	"github.com/Azure/go-autorest/autorest/to"
)

//go:embed testdata
var hiveEmbeddedFiles embed.FS

func TestProdHive(t *testing.T) {
	ctx := context.Background()

	controller := gomock.NewController(t)
	defer controller.Finish()

	kv := mock_keyvault.NewMockManager(controller)

	kc, err := hiveEmbeddedFiles.ReadFile("testdata/kubeconfig")
	if err != nil {
		t.Fatal(err)
	}

	hc := &hiveConfig{
		Shards: []hiveShard{
			{
				Kubeconfig: kc,
			},
		},
	}

	response, err := json.Marshal(hc)
	if err != nil {
		t.Fatal(err)
	}

	rsp := keyvault.SecretBundle{
		Value: to.StringPtr(string(response)),
	}
	kv.EXPECT().GetSecret(gomock.Any(), "HiveConfig").Return(rsp, nil)

	lc := &prod{kv: kv}

	restConfig, err := lc.HiveRestConfig(ctx, 0)
	if err != nil {
		t.Fatal(err)
	}

	// rudimentary loading checks
	if restConfig.Host != "https://api.crc.testing:6443" {
		t.Error(restConfig.String())
	}

	if restConfig.BearerToken != "none" {
		t.Error(restConfig.String())
	}
}
