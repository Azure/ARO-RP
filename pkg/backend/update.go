package backend

import (
	"context"

	"github.com/sirupsen/logrus"

	"github.com/jim-minter/rp/pkg/api"
)

func (b *backend) update(ctx context.Context, log *logrus.Entry, doc *api.OpenShiftClusterDocument) error {
	if doc.OpenShiftCluster.Properties.Installation != nil {
		return b.install(ctx, log, doc)
	}

	return b.scale(ctx, log, doc)
}
