package sets

import (
	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/mimo/tasks/cluster"
	"github.com/Azure/ARO-RP/pkg/util/mimo"
	"github.com/Azure/ARO-RP/pkg/util/steps"
)

func TLSCertRotation(t mimo.TaskContext, doc *api.MaintenanceManifestDocument, oc *api.OpenShiftClusterDocument) (api.MaintenanceManifestState, string) {
	s := []steps.Step{
		steps.Action(cluster.EnsureAPIServerIsUp),
	}

	return run(t, s)
}
