package config

import (
	"fmt"
	"strings"

	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
)

func GetPolicyConfig(instance *arov1alpha1.Cluster, filename string) (*GuardRailsPolicyConfig, error) {
	name, _, found := strings.Cut(filename, ".")
	if !found {
		return nil, fmt.Errorf("malformed name: '%s'", name)
	}

	managedPath := fmt.Sprintf(controllerPolicyManagedTemplate, name)
	managed := instance.Spec.OperatorFlags.GetWithDefault(managedPath, "false")

	enforcementPath := fmt.Sprintf(controllerPolicyEnforcementTemplate, name)
	enforcement := instance.Spec.OperatorFlags.GetWithDefault(enforcementPath, "dryrun")

	return &GuardRailsPolicyConfig{
		Managed:     managed,
		Enforcement: enforcement,
	}, nil
}
