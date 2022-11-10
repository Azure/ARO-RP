package vsphere

import (
	"encoding/json"
	"strings"

	"github.com/pkg/errors"

	machineapi "github.com/openshift/api/machine/v1beta1"

	"github.com/openshift/installer/pkg/tfvars/internal/cache"
	vtypes "github.com/openshift/installer/pkg/types/vsphere"
)

type config struct {
	VSphereURL        string          `json:"vsphere_url"`
	VSphereUsername   string          `json:"vsphere_username"`
	VSpherePassword   string          `json:"vsphere_password"`
	MemoryMiB         int64           `json:"vsphere_control_plane_memory_mib"`
	DiskGiB           int32           `json:"vsphere_control_plane_disk_gib"`
	NumCPUs           int32           `json:"vsphere_control_plane_num_cpus"`
	NumCoresPerSocket int32           `json:"vsphere_control_plane_cores_per_socket"`
	Cluster           string          `json:"vsphere_cluster"`
	ResourcePool      string          `json:"vsphere_resource_pool"`
	Datacenter        string          `json:"vsphere_datacenter"`
	Datastore         string          `json:"vsphere_datastore"`
	Folder            string          `json:"vsphere_folder"`
	Network           string          `json:"vsphere_network"`
	Template          string          `json:"vsphere_template"`
	OvaFilePath       string          `json:"vsphere_ova_filepath"`
	PreexistingFolder bool            `json:"vsphere_preexisting_folder"`
	DiskType          vtypes.DiskType `json:"vsphere_disk_type"`
}

// TFVarsSources contains the parameters to be converted into Terraform variables
type TFVarsSources struct {
	ControlPlaneConfigs []*machineapi.VSphereMachineProviderSpec
	Username            string
	Password            string
	Cluster             string
	ImageURL            string
	PreexistingFolder   bool
	DiskType            vtypes.DiskType
	NetworkID           string
}

//TFVars generate vSphere-specific Terraform variables
func TFVars(sources TFVarsSources) ([]byte, error) {
	controlPlaneConfig := sources.ControlPlaneConfigs[0]

	cachedImage, err := cache.DownloadImageFile(sources.ImageURL)
	if err != nil {
		return nil, errors.Wrap(err, "failed to use cached vsphere image")
	}

	// The vSphere provider needs the relativepath of the folder,
	// so get the relPath from the absolute path. Absolute path is always of the form
	// /<datacenter>/vm/<folder_path> so we can split on "vm/".
	folderRelPath := strings.SplitAfterN(controlPlaneConfig.Workspace.Folder, "vm/", 2)[1]

	cfg := &config{
		VSphereURL:        controlPlaneConfig.Workspace.Server,
		VSphereUsername:   sources.Username,
		VSpherePassword:   sources.Password,
		MemoryMiB:         controlPlaneConfig.MemoryMiB,
		DiskGiB:           controlPlaneConfig.DiskGiB,
		NumCPUs:           controlPlaneConfig.NumCPUs,
		NumCoresPerSocket: controlPlaneConfig.NumCoresPerSocket,
		Cluster:           sources.Cluster,
		ResourcePool:      controlPlaneConfig.Workspace.ResourcePool,
		Datacenter:        controlPlaneConfig.Workspace.Datacenter,
		Datastore:         controlPlaneConfig.Workspace.Datastore,
		Folder:            folderRelPath,
		Network:           sources.NetworkID,
		Template:          controlPlaneConfig.Template,
		OvaFilePath:       cachedImage,
		PreexistingFolder: sources.PreexistingFolder,
		DiskType:          sources.DiskType,
	}

	return json.MarshalIndent(cfg, "", "  ")
}
