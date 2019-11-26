package backend

import (
	"context"

	"github.com/sirupsen/logrus"

	"github.com/jim-minter/rp/pkg/api"
)

func (b *backend) update(ctx context.Context, log *logrus.Entry, oc *api.OpenShiftCluster) error {
	if oc.Properties.Installation != nil {
		return b.install(ctx, log, oc)
	}

	return b.scale(ctx, log, oc)
}
