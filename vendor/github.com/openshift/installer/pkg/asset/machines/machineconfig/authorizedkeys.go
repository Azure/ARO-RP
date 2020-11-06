package machineconfig

import (
	"encoding/json"
	"fmt"

	ignv2_2types "github.com/coreos/ignition/config/v2_2/types"
	mcfgv1 "github.com/openshift/machine-config-operator/pkg/apis/machineconfiguration.openshift.io/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// ForAuthorizedKeys creates the MachineConfig to set the authorized key for `core` user.
func ForAuthorizedKeys(key string, role string) (*mcfgv1.MachineConfig, error) {
	b, err := json.Marshal(ignv2_2types.Config{
		Ignition: ignv2_2types.Ignition{
			Version: ignv2_2types.MaxVersion.String(),
		},
		Passwd: ignv2_2types.Passwd{
			Users: []ignv2_2types.PasswdUser{{
				Name: "core", SSHAuthorizedKeys: []ignv2_2types.SSHAuthorizedKey{ignv2_2types.SSHAuthorizedKey(key)},
			}},
		},
	})
	if err != nil {
		return nil, err
	}

	return &mcfgv1.MachineConfig{
		TypeMeta: metav1.TypeMeta{
			APIVersion: mcfgv1.SchemeGroupVersion.String(),
			Kind:       "MachineConfig",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: fmt.Sprintf("99-%s-ssh", role),
			Labels: map[string]string{
				"machineconfiguration.openshift.io/role": role,
			},
		},
		Spec: mcfgv1.MachineConfigSpec{
			Config: runtime.RawExtension{
				Raw: b,
			},
		},
	}, nil
}
