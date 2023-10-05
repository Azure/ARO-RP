package cluster

import (
	"reflect"
	"testing"

	utilerror "github.com/Azure/ARO-RP/test/util/error"

	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func Test_getClient(t *testing.T) {
	tests := []struct {
		name          string
		restConfig    *rest.Config
		clientOptions client.Options
		want          client.Client
		wantErr       string
	}{
		{
			name:          "should return error when restConfig is nil",
			restConfig:    nil,
			clientOptions: client.Options{},
			want:          nil,
			wantErr:       "must provide non-nil rest.Config to client.New",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getClient(tt.restConfig, tt.clientOptions)
			utilerror.AssertErrorMessage(t, err, tt.wantErr)

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getClient() = %v, want %v", got, tt.want)
			}
		})
	}
}
