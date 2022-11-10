package validation

import (
	"k8s.io/apimachinery/pkg/util/validation/field"

	"github.com/openshift/installer/pkg/types/vsphere"
)

// ValidateMachinePool checks that the specified machine pool is valid.
func ValidateMachinePool(p *vsphere.MachinePool, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	if p.DiskSizeGB < 0 {
		allErrs = append(allErrs, field.Invalid(fldPath.Child("diskSizeGB"), p.DiskSizeGB, "storage disk size must be positive"))
	}
	if p.MemoryMiB < 0 {
		allErrs = append(allErrs, field.Invalid(fldPath.Child("memoryMB"), p.MemoryMiB, "memory size must be positive"))
	}
	if p.NumCPUs < 0 {
		allErrs = append(allErrs, field.Invalid(fldPath.Child("cpus"), p.NumCPUs, "number of CPUs must be positive"))
	}
	if p.NumCoresPerSocket < 0 {
		allErrs = append(allErrs, field.Invalid(fldPath.Child("coresPerSocket"), p.NumCoresPerSocket, "cores per socket must be positive"))
	}

	defaultCoresPerSocket := int32(4)
	defaultNumCPUs := int32(4)
	if p.NumCPUs > 0 {
		if p.NumCoresPerSocket > p.NumCPUs {
			allErrs = append(allErrs, field.Invalid(fldPath.Child("coresPerSocket"), p.NumCoresPerSocket, "cores per socket must be less than number of CPUs"))
		} else if p.NumCoresPerSocket > 0 && p.NumCPUs%p.NumCoresPerSocket != 0 {
			allErrs = append(allErrs, field.Invalid(fldPath.Child("cpus"), p.NumCPUs, "numCPUs specified should be a multiple of cores per socket"))
		} else if p.NumCPUs%defaultCoresPerSocket != 0 {
			allErrs = append(allErrs, field.Invalid(fldPath.Child("cpus"), p.NumCPUs, "numCPUs specified should be a multiple of cores per socket which is by default 4"))
		}
	} else if p.NumCoresPerSocket > defaultNumCPUs {
		allErrs = append(allErrs, field.Invalid(fldPath.Child("coresPerSocket"), p.NumCoresPerSocket, "cores per socket must be less than number of CPUs which is by default 4"))
	}
	return allErrs
}
