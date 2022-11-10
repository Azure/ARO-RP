package validation

import (
	"math"
	"regexp"
	"strconv"

	"github.com/google/uuid"
	"github.com/openshift/installer/pkg/types/powervs"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

// ValidateMachinePool checks that the specified machine pool is valid.
func ValidateMachinePool(p *powervs.MachinePool, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	// Validate VolumeIDs
	volumes := make(map[string]bool)
	for i, volumeID := range p.VolumeIDs {
		_, err := uuid.Parse(volumeID)
		if err != nil {
			allErrs = append(allErrs, field.Invalid(fldPath.Child("volumeIDs").Index(i), volumeID, "volume ID must be a valid UUID"))
		}
		if _, ok := volumes[volumeID]; ok {
			allErrs = append(allErrs, field.Duplicate(fldPath.Child("volumeIDs").Index(i), volumeID))
			continue
		}
		volumes[volumeID] = true
	}

	// Validate Memory
	if p.MemoryGiB != 0 {
		if p.MemoryGiB < 2 || p.MemoryGiB > 64 {
			allErrs = append(allErrs, field.Invalid(fldPath.Child("memory"), p.MemoryGiB, "memory must be an integer number of GB that is at least 2 and no more than 64"))
		}
	}

	// Validate Processors
	var processors float64
	var err error

	switch p.Processors.Type {
	case intstr.Int:
		processors = float64(p.Processors.IntVal)
	case intstr.String:
		processors, err = strconv.ParseFloat(p.Processors.StrVal, 64)
		if err != nil {
			allErrs = append(allErrs, field.Invalid(fldPath.Child("processors"), p.Processors.StrVal, "processors must be a valid floating point number"))
		}
	}
	if err == nil && processors != 0 {
		if processors < 0.5 || processors > 32 {
			allErrs = append(allErrs, field.Invalid(fldPath.Child("processors"), processors, "number of processors must be from .5 to 32 cores"))
		}
		if math.Mod(processors*1000, 2) != 0 || math.Mod(processors*100, 25) != 0 {
			allErrs = append(allErrs, field.Invalid(fldPath.Child("processors"), processors, "processors must be in increments of .25"))
		}
	}

	// Validate ProcType
	if p.ProcType != "" {
		procTypes := sets.NewString("Shared", "Dedicated", "Capped")
		if !procTypes.Has(string(p.ProcType)) {
			allErrs = append(allErrs, field.NotSupported(fldPath.Child("procType"), p.ProcType, procTypes.List()))
		}
	}

	// Validate SysType
	if p.SysType != "" {
		const sysTypeRegex = `^(?:e980|s922(-.*|))$`
		// Allowing for a staging-only pattern of s922-* but not exposing here
		if !regexp.MustCompile(sysTypeRegex).MatchString(p.SysType) {
			allErrs = append(allErrs, field.Invalid(fldPath.Child("sysType"), p.SysType, "system type must be one of {e980,s922}"))
		}
	}
	return allErrs
}
