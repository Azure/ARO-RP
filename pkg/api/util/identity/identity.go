package identity

import (
	"github.com/Azure/ARO-RP/pkg/api"
)

func IsManagedWorkloadIdentityEnabled(cluster *api.OpenShiftCluster) bool {
	if cluster.Properties.ServicePrincipalProfile == nil && cluster.Properties.PlatformWorkloadIdentityProfile != nil && cluster.Identity != nil {
		return true
	}

	return false
}
