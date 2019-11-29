package v20191231preview

import (
	"context"
	"net/http"
	"regexp"

	"github.com/jim-minter/rp/pkg/api"
)

var (
	rxKubeadminPassword = regexp.MustCompile(`(?i)^[-a-z0-9]{0,64}$`)
)

// Validate validates an OpenShift cluster's credentials
func (occ *OpenShiftClusterCredentials) Validate(ctx context.Context, tenantID, resourceID string, current *api.OpenShiftCluster) error {
	err := occ.validate(resourceID)
	if err != nil {
		return err
	}

	if current == nil {
		return nil
	}

	return occ.validateDelta(OpenShiftClusterToExternal(current))
}

func (occ *OpenShiftClusterCredentials) validate(resourceID string) error {
	if !rxKubeadminPassword.MatchString(occ.KubeadminPassword) {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, "kubeadminPassword", "The provided kubeadmin password is invalid.")
	}

	return nil
}

func (occ *OpenShiftClusterCredentials) validateDelta(current *OpenShiftCluster) error {
	return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodePropertyChangeNotAllowed, "kubeadminPassword", "Changing property 'kubeadminPassword' is not allowed.")
}
