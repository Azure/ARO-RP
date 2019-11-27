package openshiftcluster

import (
	"context"
)

func (m *Manager) Update(ctx context.Context) error {
	if m.oc.Properties.Installation != nil {
		return m.install(ctx)
	}

	return m.scale(ctx)
}
