package azure

// MachinePool stores the configuration for a machine pool installed
// on Azure.
type MachinePool struct {
	// Zones is list of availability zones that can be used.
	// eg. ["1", "2", "3"]
	//
	// +optional
	Zones []string `json:"zones,omitempty"`

	// InstanceType defines the azure instance type.
	// eg. Standard_DS_V2
	//
	// +optional
	InstanceType string `json:"type"`

	// EncryptionAtHost
	//
	// +optional
	EncryptionAtHost bool `json:"encryptionAtHost,omitempty"`

	// OSDisk defines the storage for instance.
	//
	// +optional
	OSDisk `json:"osDisk"`

	// ultraSSDCapability defines if the instance should use Ultra SSD disks.
	//
	// +optional
	// +kubebuilder:validation:Enum=Enabled;Disabled
	UltraSSDCapability string `json:"ultraSSDCapability,omitempty"`

	// VMNetworkingType specifies whether to enable accelerated networking.
	// Accelerated networking enables single root I/O virtualization (SR-IOV) to a VM, greatly improving its
	// networking performance.
	// eg. values: "Accelerated", "Basic"
	//
	// +kubebuilder:validation:Enum="Accelerated"; "Basic"
	// +optional
	VMNetworkingType string `json:"vmNetworkingType,omitempty"`

	// OSImage defines the image to use for the OS.
	// +optional
	OSImage OSImage `json:"osImage,omitempty"`
}

// OSDisk defines the disk for machines on Azure.
type OSDisk struct {
	// DiskSizeGB defines the size of disk in GB.
	//
	// +kubebuilder:validation:Minimum=0
	DiskSizeGB int32 `json:"diskSizeGB"`
	// DiskType defines the type of disk.
	// The valid values are Standard_LRS, Premium_LRS, StandardSSD_LRS
	// For control plane nodes, the valid values are Premium_LRS and StandardSSD_LRS.
	// Default is Premium_LRS
	// +optional
	// +kubebuilder:validation:Enum=Standard_LRS;Premium_LRS;StandardSSD_LRS
	DiskType string `json:"diskType"`

	// DiskEncryptionSetID is a resource ID of disk encryption set
	// +optional
	DiskEncryptionSetID string `json:"diskEncryptionSetId,omitempty"`

	// DiskEncryptionSet defines a disk encryption set.
	//
	// +optional
	*DiskEncryptionSet `json:"diskEncryptionSet,omitempty"`
}

// DiskEncryptionSet defines the configuration for a disk encryption set.
type DiskEncryptionSet struct {
	// SubscriptionID defines the Azure subscription the disk encryption
	// set is in.
	SubscriptionID string `json:"subscriptionId,omitempty"`
	// ResourceGroup defines the Azure resource group used by the disk
	// encryption set.
	ResourceGroup string `json:"resourceGroup"`
	// Name is the name of the disk encryption set.
	Name string `json:"name"`
}

// DefaultDiskType holds the default Azure disk type used by the VMs.
const DefaultDiskType string = "Premium_LRS"

// Set sets the values from `required` to `a`.
func (a *MachinePool) Set(required *MachinePool) {
	if required == nil || a == nil {
		return
	}

	if len(required.Zones) > 0 {
		a.Zones = required.Zones
	}

	if required.InstanceType != "" {
		a.InstanceType = required.InstanceType
	}

	if required.EncryptionAtHost {
		a.EncryptionAtHost = required.EncryptionAtHost
	}

	if required.OSDisk.DiskSizeGB != 0 {
		a.OSDisk.DiskSizeGB = required.OSDisk.DiskSizeGB
	}

	if required.OSDisk.DiskType != "" {
		a.OSDisk.DiskType = required.OSDisk.DiskType
	}

	if required.OSDisk.DiskEncryptionSetID != "" {
		a.OSDisk.DiskEncryptionSetID = required.OSDisk.DiskEncryptionSetID
	}
}

// OSImage is the image to use for the OS of a machine.
type OSImage struct {
	// Publisher is the publisher of the image.
	Publisher string `json:"publisher"`
	// Offer is the offer of the image.
	Offer string `json:"offer"`
	// SKU is the SKU of the image.
	SKU string `json:"sku"`
	// Version is the version of the image.
	Version string `json:"version"`
}
