package validation

import (
	"regexp"

	"github.com/openshift/installer/pkg/types/azure"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

var resourceGroupRx = regexp.MustCompile(`(?i)^[-a-z0-9_().]{0,89}[-a-z0-9_()]$`)

// ValidatePlatform checks that the specified platform is valid.
func ValidatePlatform(p *azure.Platform, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	if p.Region == "" {
		allErrs = append(allErrs, field.Required(fldPath.Child("region"), "region should be set to one of the supported Azure regions"))
	}
	if p.ResourceGroup == "" {
		allErrs = append(allErrs, field.Required(fldPath.Child("resourceGroup"), "resourceGroup should be set"))
	}
	if !resourceGroupRx.MatchString(p.ResourceGroup) {
		allErrs = append(allErrs, field.Invalid(fldPath.Child("resourceGroup"), p.ResourceGroup, "resourceGroup is invalid"))
	}
	if p.BaseDomainResourceGroupName == "" {
		allErrs = append(allErrs, field.Required(fldPath.Child("baseDomainResourceGroupName"), "baseDomainResourceGroupName is the resource group name where the azure dns zone is deployed"))
	}
	if p.DefaultMachinePlatform != nil {
		allErrs = append(allErrs, ValidateMachinePool(p.DefaultMachinePlatform, fldPath.Child("defaultMachinePlatform"))...)
	}
	return allErrs
}
