package identity

import "github.com/Azure/ARO-RP/pkg/api"

func IsManagedIdentityCluster(cluster *api.OpenShiftCluster) bool {
	if cluster.Properties.ServicePrincipalProfile == nil && cluster.Properties.PlatformWorkloadIdentityProfile != nil {
		return true
	}

	return false
}
