package cluster

import (
	"context"
)

func (mon *Monitor) emitClusterSyncStatus(ctx context.Context) error {
	clusterSync, err := mon.hiveclustermanager.GetClusterSync(ctx, mon.oc)
	if err != nil {
		return err
	}

	if clusterSync != nil {
		clustersyncLabels := make(map[string]string)

		if clusterSync.Status.SyncSets != nil {
			for _, s := range clusterSync.Status.SyncSets {
				labels := map[string]string{
					"name":               s.Name,
					"result":             string(s.Result),
					"firstSuccessTime":   s.FirstSuccessTime.String(),
					"lastTransitionTime": s.LastTransitionTime.String(),
					"failureMessage":     s.FailureMessage,
				}
				mon.emitGauge("hive.syncsets", 1, labels)
				for k, v := range labels {
					clustersyncLabels[k] = v
				}
			}
		}

		if clusterSync.Status.SelectorSyncSets != nil {
			for _, s := range clusterSync.Status.SelectorSyncSets {
				labels := map[string]string{
					"name":               s.Name,
					"result":             string(s.Result),
					"firstSuccessTime":   s.FirstSuccessTime.String(),
					"lastTransitionTime": s.LastTransitionTime.String(),
					"failureMessage":     s.FailureMessage,
				}
				mon.emitGauge("hive.selectorsyncsets", 1, labels)
				for k, v := range labels {
					clustersyncLabels[k] = v
				}
			}
		}

		if len(clustersyncLabels) > 0 {
			mon.emitGauge("hive.clustersync", 1, clustersyncLabels)
		}
	}

	return nil
}
