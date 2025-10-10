package redact

import (
	"reflect"
	"testing"

	"github.com/Azure/ARO-RP/pkg/api"
)

func TestRedactPullSecretBasic(t *testing.T) {
	oc := &api.OpenShiftCluster{}
	oc.Properties.ClusterProfile.PullSecret = api.SecureString("pull-secret")
	oc.Properties.AdminKubeconfig = []byte("admin-kube")
	oc.Properties.KubeadminPassword = api.SecureString("kubeadmin")
	oc.Properties.AROServiceKubeconfig = []byte("aro-kube")
	oc.Properties.AROSREKubeconfig = []byte("sre-kube")
	oc.Properties.UserAdminKubeconfig = []byte("user-kube")
	oc.Properties.SSHKey = []byte("ssh-key")

	oc.Properties.ServicePrincipalProfile = &api.ServicePrincipalProfile{
		ClientID:     "cid",
		ClientSecret: api.SecureString("sp-secret"),
	}

	oc.Properties.RegistryProfiles = []*api.RegistryProfile{{
		Name: "r", Username: "u", Password: api.SecureString("rpw"),
	}}

	// keep a copy for mutation check
	//nolint:govet
	before := *oc

	redacted, err := RedactPullSecret(oc)
	if err != nil {
		t.Fatalf("RedactPullSecret error: %v", err)
	}

	// assert redacted fields
	if string(redacted.Properties.AdminKubeconfig) == string(before.Properties.AdminKubeconfig) {
		t.Fatalf("admin kubeconfig not redacted")
	}
	if redacted.Properties.KubeadminPassword == before.Properties.KubeadminPassword {
		t.Fatalf("kubeadmin password not redacted")
	}
	if redacted.Properties.ClusterProfile.PullSecret == before.Properties.ClusterProfile.PullSecret {
		t.Fatalf("pull secret not redacted")
	}
	if redacted.Properties.ServicePrincipalProfile == nil || redacted.Properties.ServicePrincipalProfile.ClientSecret == before.Properties.ServicePrincipalProfile.ClientSecret {
		t.Fatalf("service principal secret not redacted")
	}
	if redacted.Properties.RegistryProfiles[0].Password == before.Properties.RegistryProfiles[0].Password {
		t.Fatalf("registry password not redacted")
	}

	// ensure original wasn't mutated
	if !reflect.DeepEqual(before.Properties.AdminKubeconfig, oc.Properties.AdminKubeconfig) {
		t.Fatalf("original AdminKubeconfig mutated")
	}
	if !reflect.DeepEqual(before.Properties.RegistryProfiles[0].Password, oc.Properties.RegistryProfiles[0].Password) {
		t.Fatalf("original registry password mutated")
	}
}
