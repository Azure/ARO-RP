package dbload

import (
	"context"
	"testing"

	"github.com/Azure/ARO-RP/pkg/database/cosmosdb"
	utillog "github.com/Azure/ARO-RP/pkg/util/log"
)

func TestQuery(t *testing.T) {
	ctx := context.Background()
	log := utillog.GetLogger()

	c, err := get(ctx, log)
	if err != nil {
		t.Fatal(err)
	}

	docs, err := c.QueryAll(ctx, "", &cosmosdb.Query{
		Query: `SELECT * FROM OpenShiftClusters doc ` +
			`WHERE doc.openShiftCluster.properties.provisioningState = "Creating" ` +
			`AND (doc.leaseExpires ?? 0) < 1000000`,
	}, nil)
	if err != nil {
		t.Fatal(err)
	}

	t.Log(docs.Count)
}
