package redact

import (
	"fmt"

	"github.com/Azure/ARO-RP/pkg/api"
)

const (
	// RedactedString is the token used to replace sensitive data.
	RedactedString = "##REDACTED##"
	// MaxSecretSizeBytes aims to defend from excessive memory allocation.
	MaxSecretSizeBytes = 10 * 1024 * 1024 // 10MB
)

// RedactPullSecret creates a deep c of the OpenshiftCluster with the pull secret redacted.
// It return a new object with Properties.ClusterProfile.PullSecret set to RedactedString.
//
// This function is designed to be used in admin API responses where pull secrets should not
// be exposed but the presence of a secret should be indicated.
func RedactPullSecret(oc *api.OpenShiftCluster) (*api.OpenShiftCluster, error) {
	if oc == nil {
		return nil, fmt.Errorf("cannot redact pull secret from nil OpenshiftCluster")
	}

	// Defensive c of the entire OpenshiftCluster object.
	redacted, err := DeepCopy(oc)
	if err != nil {
		return nil, err
	}

	return redacted, nil
}

func DeepCopy(oc *api.OpenShiftCluster) (*api.OpenShiftCluster, error) {
	if oc == nil {
		return nil, nil
	}
	// Instantiate a new *OpenShiftCluster without copying a sync.Mutex value from a live object.
	c := &api.OpenShiftCluster{
		ID:         oc.ID,
		Name:       oc.Name,
		Type:       oc.Type,
		Location:   oc.Location,
		Tags:       make(map[string]string, len(oc.Tags)),
		Properties: oc.Properties,
	}

	// copy Tags map to avoid sharing backing map
	for k, v := range oc.Tags {
		c.Tags[k] = v
	}

	// Redact secrets if non-empty
	if oc.Properties.ServicePrincipalProfile != nil {
		sp := *oc.Properties.ServicePrincipalProfile
		sp.ClientSecret = RedactedString
		c.Properties.ServicePrincipalProfile = &sp
	}
	if c.Properties.ClusterProfile.PullSecret != "" {
		c.Properties.ClusterProfile.PullSecret = RedactedString
	}
	if c.Properties.ClusterProfile.BoundServiceAccountSigningKey != nil {
		s := api.SecureString(RedactedString)
		c.Properties.ClusterProfile.BoundServiceAccountSigningKey = &s
	}
	if len(oc.Properties.AdminKubeconfig) > 0 {
		c.Properties.AdminKubeconfig = []byte(RedactedString)
	}
	if oc.Properties.KubeadminPassword != "" {
		c.Properties.KubeadminPassword = api.SecureString(RedactedString)
	}
	if len(oc.Properties.AROServiceKubeconfig) > 0 {
		c.Properties.AROServiceKubeconfig = []byte(RedactedString)
	}
	if len(oc.Properties.AROSREKubeconfig) > 0 {
		c.Properties.AROSREKubeconfig = []byte(RedactedString)
	}
	if len(oc.Properties.UserAdminKubeconfig) > 0 {
		c.Properties.UserAdminKubeconfig = []byte(RedactedString)
	}
	if len(oc.Properties.SSHKey) > 0 {
		c.Properties.SSHKey = []byte(RedactedString)
	}

	// deep-copy slices of pointers (e.g., RegistryProfiles []*RegistryProfile)
	if oc.Properties.RegistryProfiles != nil {
		newRegs := make([]*api.RegistryProfile, 0, len(oc.Properties.RegistryProfiles))
		for _, r := range oc.Properties.RegistryProfiles {
			if r == nil {
				newRegs = append(newRegs, nil)
				continue
			}
			rr := *r // shallow copy element
			rr.Password = api.SecureString(RedactedString)
			// IssueDate is safe to keep pointer; copy pointer or value as needed
			newRegs = append(newRegs, &rr)
		}
		c.Properties.RegistryProfiles = newRegs
	}

	// deep copy WorkerProfiles slice (copy each element explicitly so the
	// new slice has its own backing array and we avoid accidental sharing of
	// mutable internals if fields are added later)
	if oc.Properties.WorkerProfiles != nil {
		wp := make([]api.WorkerProfile, 0, len(oc.Properties.WorkerProfiles))
		wp = append(wp, oc.Properties.WorkerProfiles...)
		c.Properties.WorkerProfiles = wp
	}

	return c, nil
}
