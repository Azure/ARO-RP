//go:generate go run github.com/jim-minter/go-cosmosdb/cmd/gencosmosdb github.com/jim-minter/rp/pkg/api,OpenShiftClusterDocument

package cosmosdb

import (
	"github.com/jim-minter/rp/pkg/api"
)

func init() {
	api.AddExtensions(&JSONHandle.BasicHandle)
}
