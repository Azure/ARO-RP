package sets

import (
	"time"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/util/mimo"
	"github.com/Azure/ARO-RP/pkg/util/steps"
)

const DEFAULT_POLL_TIME = time.Second * 10
const DEFAULT_TIMEOUT_DURATION = time.Minute * 20

var DEFAULT_MAINTENANCE_SETS = map[string]MaintenanceSet{
	"9b741734-6505-447f-8510-85eb0ae561a2": TLSCertRotation,
}

func run(t mimo.TaskContext, s []steps.Step) (api.MaintenanceManifestState, string) {
	_, err := steps.Run(t, t.Log(), DEFAULT_POLL_TIME, s, t.Now)

	if err != nil {
		if mimo.IsRetryableError(err) {
			return api.MaintenanceManifestStatePending, err.Error()
		}
		return api.MaintenanceManifestStateFailed, err.Error()
	}
	return api.MaintenanceManifestStateCompleted, t.GetResultMessage()
}
