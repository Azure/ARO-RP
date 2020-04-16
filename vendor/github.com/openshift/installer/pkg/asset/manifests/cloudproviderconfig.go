package manifests

import (
	"encoding/json"
	"fmt"
	"path/filepath"

	"github.com/ghodss/yaml"
	ospclientconfig "github.com/gophercloud/utils/openstack/clientconfig"
	"github.com/pkg/errors"

	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/openshift/installer/pkg/asset"
	"github.com/openshift/installer/pkg/asset/installconfig"
	icazure "github.com/openshift/installer/pkg/asset/installconfig/azure"
	"github.com/openshift/installer/pkg/asset/manifests/azure"
	gcpmanifests "github.com/openshift/installer/pkg/asset/manifests/gcp"
	openstackmanifests "github.com/openshift/installer/pkg/asset/manifests/openstack"
	vspheremanifests "github.com/openshift/installer/pkg/asset/manifests/vsphere"
	awstypes "github.com/openshift/installer/pkg/types/aws"
	azuretypes "github.com/openshift/installer/pkg/types/azure"
	baremetaltypes "github.com/openshift/installer/pkg/types/baremetal"
	gcptypes "github.com/openshift/installer/pkg/types/gcp"
	libvirttypes "github.com/openshift/installer/pkg/types/libvirt"
	nonetypes "github.com/openshift/installer/pkg/types/none"
	openstacktypes "github.com/openshift/installer/pkg/types/openstack"
	vspheretypes "github.com/openshift/installer/pkg/types/vsphere"
)

var (
	cloudProviderConfigFileName = filepath.Join(manifestDir, "cloud-provider-config.yaml")
)

const (
	cloudProviderConfigDataKey = "config"
)

// CloudProviderConfig generates the cloud-provider-config.yaml files.
type CloudProviderConfig struct {
	ConfigMap *corev1.ConfigMap
	File      []*asset.File
}

var _ asset.WritableAsset = (*CloudProviderConfig)(nil)

// Name returns a human friendly name for the asset.
func (*CloudProviderConfig) Name() string {
	return "Cloud Provider Config"
}

// Dependencies returns all of the dependencies directly needed to generate
// the asset.
func (*CloudProviderConfig) Dependencies() []asset.Asset {
	return []asset.Asset{
		&installconfig.PlatformCreds{},
		&installconfig.InstallConfig{},
		&installconfig.ClusterID{},
		// PlatformCredsCheck just checks the creds (and asks, if needed)
		// We do not actually use it in this asset directly, hence
		// it is put in the dependencies but not fetched in Generate
		&installconfig.PlatformCredsCheck{},
	}
}

// Generate generates the CloudProviderConfig.
func (cpc *CloudProviderConfig) Generate(dependencies asset.Parents) error {
	platformCreds := &installconfig.PlatformCreds{}
	installConfig := &installconfig.InstallConfig{}
	clusterID := &installconfig.ClusterID{}
	dependencies.Get(platformCreds, installConfig, clusterID)

	cm := &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			APIVersion: corev1.SchemeGroupVersion.String(),
			Kind:       "ConfigMap",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "openshift-config",
			Name:      "cloud-provider-config",
		},
		Data: map[string]string{},
	}

	switch installConfig.Config.Platform.Name() {
	case awstypes.Name, libvirttypes.Name, nonetypes.Name, baremetaltypes.Name:
		return nil
	case openstacktypes.Name:
		opts := &ospclientconfig.ClientOpts{}
		opts.Cloud = installConfig.Config.Platform.OpenStack.Cloud
		cloud, err := ospclientconfig.GetCloudFromYAML(opts)
		if err != nil {
			return errors.Wrap(err, "failed to get cloud config for openstack")
		}

		cm.Data[cloudProviderConfigDataKey] = openstackmanifests.CloudProviderConfig(cloud)
	case azuretypes.Name:
		session, err := icazure.GetSession(platformCreds.Azure)
		if err != nil {
			return errors.Wrap(err, "could not get azure session")
		}

		nsg := fmt.Sprintf("%s-node-nsg", clusterID.InfraID)
		nrg := installConfig.Config.Azure.ResourceGroupName
		if installConfig.Config.Azure.NetworkResourceGroupName != "" {
			nrg = installConfig.Config.Azure.NetworkResourceGroupName
		}
		vnet := fmt.Sprintf("%s-vnet", clusterID.InfraID)
		if installConfig.Config.Azure.VirtualNetwork != "" {
			vnet = installConfig.Config.Azure.VirtualNetwork
		}
		subnet := fmt.Sprintf("%s-worker-subnet", clusterID.InfraID)
		if installConfig.Config.Azure.ComputeSubnet != "" {
			subnet = installConfig.Config.Azure.ComputeSubnet
		}
		config := azure.CloudProviderConfig{
			GroupLocation:            installConfig.Config.Azure.Region,
			ResourcePrefix:           clusterID.InfraID,
			SubscriptionID:           session.Credentials.SubscriptionID,
			TenantID:                 session.Credentials.TenantID,
			ResourceGroupName:        installConfig.Config.Azure.ResourceGroupName,
			NetworkResourceGroupName: nrg,
			NetworkSecurityGroupName: nsg,
			VirtualNetworkName:       vnet,
			SubnetName:               subnet,
		}
		azureConfig, err := config.JSON()
		if err != nil {
			return errors.Wrap(err, "could not create cloud provider config")
		}
		cm.Data[cloudProviderConfigDataKey] = azureConfig
	case gcptypes.Name:
		subnet := fmt.Sprintf("%s-worker-subnet", clusterID.InfraID)
		if installConfig.Config.GCP.ComputeSubnet != "" {
			subnet = installConfig.Config.GCP.ComputeSubnet
		}
		gcpConfig, err := gcpmanifests.CloudProviderConfig(clusterID.InfraID, installConfig.Config.GCP.ProjectID, subnet)
		if err != nil {
			return errors.Wrap(err, "could not create cloud provider config")
		}
		cm.Data[cloudProviderConfigDataKey] = gcpConfig
	case vspheretypes.Name:
		vsphereConfig, err := vspheremanifests.CloudProviderConfig(
			installConfig.Config.ObjectMeta.Name,
			installConfig.Config.Platform.VSphere,
		)
		if err != nil {
			return errors.Wrap(err, "could not create cloud provider config")
		}
		cm.Data[cloudProviderConfigDataKey] = vsphereConfig
	default:
		return errors.New("invalid Platform")
	}

	cmData, err := yaml.Marshal(cm)
	if err != nil {
		return errors.Wrapf(err, "failed to create %s manifest", cpc.Name())
	}
	cpc.ConfigMap = cm
	cpc.File = append(cpc.File, &asset.File{
		Filename: cloudProviderConfigFileName,
		Data:     cmData,
	})

	fl := map[string]func(platformCreds *installconfig.PlatformCreds) ([]byte, error){
		filepath.Join(manifestDir, "aro-cloud-provider-secret-reader-role.yaml"):        getRole,
		filepath.Join(manifestDir, "aro-cloud-provider-secret-reader-rolebinding.yaml"): getRoleBinding,
		filepath.Join(manifestDir, "aro-cloud-provider-secret.yaml"):                    getSecret,
	}
	for name, f := range fl {
		data, err := f(platformCreds)
		if err != nil {
			return errors.Wrapf(err, "failed to create %s manifest", name)
		}
		cpc.File = append(cpc.File, &asset.File{
			Filename: name,
			Data:     data,
		})
	}

	return nil
}

func (cpc *CloudProviderConfig) generateAROSecrets(dependencies asset.Parents) error {

	return nil
}

// Files returns the files generated by the asset.
func (cpc *CloudProviderConfig) Files() []*asset.File {
	if cpc.File != nil {
		return cpc.File
	}
	return []*asset.File{}
}

// Load loads the already-rendered files back from disk.
func (cpc *CloudProviderConfig) Load(f asset.FileFetcher) (bool, error) {
	return false, nil
}

func getRole(platformCreds *installconfig.PlatformCreds) ([]byte, error) {
	return json.Marshal(&rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "aro-cloud-provider-secret-reader",
			Namespace: "kube-system",
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups:     []string{""},
				Resources:     []string{"secrets"},
				ResourceNames: []string{"azure-cloud-provider"},
				Verbs:         []string{"get"},
			},
		},
	})
}

func getRoleBinding(platformCreds *installconfig.PlatformCreds) ([]byte, error) {
	return json.Marshal(&rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "aro-cloud-provider-secret-read",
			Namespace: "kube-system",
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "Role",
			Name:     "aro-cloud-provider-secret-reader",
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      "azure-cloud-provider",
				Namespace: "kube-system",
			},
		},
	})
}

func getSecret(platformCreds *installconfig.PlatformCreds) ([]byte, error) {
	secret := &v1.Secret{
		Type: v1.SecretTypeOpaque,
		ObjectMeta: metav1.ObjectMeta{
			Name:      "azure-cloud-provider",
			Namespace: "kube-system",
		},
	}
	// config is used to created compatible secret to trigger azure cloud
	// controller config merge behaviour
	// https://github.com/openshift/origin/blob/release-4.3/vendor/k8s.io/kubernetes/staging/src/k8s.io/legacy-cloud-providers/azure/azure_config.go#L82
	config := struct {
		AADClientID     string `json:"aadClientId" yaml:"aadClientId"`
		AADClientSecret string `json:"aadClientSecret" yaml:"aadClientSecret"`
	}{
		AADClientID:     platformCreds.Azure.ClientID,
		AADClientSecret: platformCreds.Azure.ClientSecret,
	}
	secretData, err := yaml.Marshal(config)
	if err != nil {
		return nil, err
	}
	secret.Data = map[string][]byte{
		"cloud-config": secretData,
	}
	return json.Marshal(secret)
}
