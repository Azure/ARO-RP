package manifests

import (
	"fmt"
	"path/filepath"

	"github.com/ghodss/yaml"
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
	kubevirtmanifests "github.com/openshift/installer/pkg/asset/manifests/kubevirt"
	openstackmanifests "github.com/openshift/installer/pkg/asset/manifests/openstack"
	vspheremanifests "github.com/openshift/installer/pkg/asset/manifests/vsphere"
	awstypes "github.com/openshift/installer/pkg/types/aws"
	azuretypes "github.com/openshift/installer/pkg/types/azure"
	baremetaltypes "github.com/openshift/installer/pkg/types/baremetal"
	gcptypes "github.com/openshift/installer/pkg/types/gcp"
	kubevirttypes "github.com/openshift/installer/pkg/types/kubevirt"
	libvirttypes "github.com/openshift/installer/pkg/types/libvirt"
	nonetypes "github.com/openshift/installer/pkg/types/none"
	openstacktypes "github.com/openshift/installer/pkg/types/openstack"
	ovirttypes "github.com/openshift/installer/pkg/types/ovirt"
	vspheretypes "github.com/openshift/installer/pkg/types/vsphere"
)

var (
	cloudProviderConfigFileName         = filepath.Join(manifestDir, "cloud-provider-config.yaml")
	aroCloudProviderRoleFileName        = filepath.Join(manifestDir, "aro-cloud-provider-secret-reader-role.yaml")
	aroCloudProviderRoleBindingFileName = filepath.Join(manifestDir, "aro-cloud-provider-secret-reader-rolebinding.yaml")
	aroCloudProviderSecretFileName      = filepath.Join(manifestDir, "aro-cloud-provider-secret.yaml")
)

const (
	cloudProviderConfigDataKey         = "config"
	cloudProviderConfigCABundleDataKey = "ca-bundle.pem"
)

// CloudProviderConfig generates the cloud-provider-config.yaml files.
type CloudProviderConfig struct {
	ConfigMap *corev1.ConfigMap
	FileList  []*asset.File
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
	installConfig := &installconfig.InstallConfig{}
	clusterID := &installconfig.ClusterID{}
	dependencies.Get(installConfig, clusterID)

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
	case libvirttypes.Name, nonetypes.Name, baremetaltypes.Name, ovirttypes.Name:
		return nil
	case awstypes.Name:
		// Store the additional trust bundle in the ca-bundle.pem key if the cluster is being installed on a C2S region.
		trustBundle := installConfig.Config.AdditionalTrustBundle
		if trustBundle == "" || !awstypes.C2SRegions.Has(installConfig.Config.AWS.Region) {
			return nil
		}
		cm.Data[cloudProviderConfigCABundleDataKey] = trustBundle

	case openstacktypes.Name:
		cloudProviderConfigData, cloudProviderConfigCABundleData, err := openstackmanifests.GenerateCloudProviderConfig(*installConfig.Config)
		if err != nil {
			return errors.Wrap(err, "failed to generate OpenStack provider config")
		}
		cm.Data[cloudProviderConfigDataKey] = cloudProviderConfigData
		if cloudProviderConfigCABundleData != "" {
			cm.Data[cloudProviderConfigCABundleDataKey] = cloudProviderConfigCABundleData
		}

	case azuretypes.Name:
		session, err := installConfig.Azure.Session()
		if err != nil {
			return errors.Wrap(err, "could not get azure session")
		}

		nsg := fmt.Sprintf("%s-nsg", clusterID.InfraID)
		nrg := installConfig.Config.Azure.ClusterResourceGroupName(clusterID.InfraID)
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
		azureConfig, err := azure.CloudProviderConfig{
			CloudName:                installConfig.Config.Azure.CloudName,
			ResourceGroupName:        installConfig.Config.Azure.ClusterResourceGroupName(clusterID.InfraID),
			GroupLocation:            installConfig.Config.Azure.Region,
			ResourcePrefix:           clusterID.InfraID,
			SubscriptionID:           session.Credentials.SubscriptionID,
			TenantID:                 session.Credentials.TenantID,
			NetworkResourceGroupName: nrg,
			NetworkSecurityGroupName: nsg,
			VirtualNetworkName:       vnet,
			SubnetName:               subnet,
			ARO:                      installConfig.Config.Azure.ARO,
		}.JSON()
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
		folderPath := installConfig.Config.Platform.VSphere.Folder
		if len(folderPath) == 0 {
			dataCenter := installConfig.Config.Platform.VSphere.Datacenter
			folderPath = fmt.Sprintf("/%s/vm/%s", dataCenter, clusterID.InfraID)
		}
		vsphereConfig, err := vspheremanifests.CloudProviderConfig(
			folderPath,
			installConfig.Config.Platform.VSphere,
		)
		if err != nil {
			return errors.Wrap(err, "could not create cloud provider config")
		}
		cm.Data[cloudProviderConfigDataKey] = vsphereConfig
	case kubevirttypes.Name:
		kubevirtConfig, err := kubevirtmanifests.CloudProviderConfig{
			Namespace: installConfig.Config.Platform.Kubevirt.Namespace,
			InfraID:   clusterID.InfraID,
		}.JSON()
		if err != nil {
			return errors.Wrap(err, "could not create cloud provider config")
		}
		cm.Data[cloudProviderConfigDataKey] = kubevirtConfig
	default:
		return errors.New("invalid Platform")
	}

	cmData, err := yaml.Marshal(cm)
	if err != nil {
		return errors.Wrapf(err, "failed to create %s manifest", cpc.Name())
	}
	cpc.ConfigMap = cm
	cpc.FileList = []*asset.File{
		{
			Filename: cloudProviderConfigFileName,
			Data:     cmData,
		},
	}
	if installConfig.Config.Azure.ARO {
		session, err := installConfig.Azure.Session()
		if err != nil {
			return errors.Wrap(err, "could not get azure session")
		}

		for _, f := range []struct {
			filename string
			data     func(icazure.Credentials) ([]byte, error)
		}{
			{
				filename: aroCloudProviderRoleFileName,
				data:     aroRole,
			},
			{
				filename: aroCloudProviderRoleBindingFileName,
				data:     aroRoleBinding,
			},
			{
				filename: aroCloudProviderSecretFileName,
				data:     aroSecret,
			},
		} {
			b, err := f.data(session.Credentials)
			if err != nil {
				return errors.Wrapf(err, "failed to create %s manifest", cpc.Name())
			}

			cpc.FileList = append(cpc.FileList, &asset.File{
				Filename: f.filename,
				Data:     b,
			})
		}
	}
	return nil
}

// Files returns the files generated by the asset.
func (cpc *CloudProviderConfig) Files() []*asset.File {
	return cpc.FileList
}

// Load loads the already-rendered files back from disk.
func (cpc *CloudProviderConfig) Load(f asset.FileFetcher) (bool, error) {
	return false, nil
}

func aroRole(icazure.Credentials) ([]byte, error) {
	return yaml.Marshal(&rbacv1.Role{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Role",
			APIVersion: "rbac.authorization.k8s.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "aro-cloud-provider-secret-reader",
			Namespace: "kube-system",
		},
		Rules: []rbacv1.PolicyRule{
			{
				Verbs:         []string{"get"},
				APIGroups:     []string{""},
				Resources:     []string{"secrets"},
				ResourceNames: []string{"azure-cloud-provider"},
			},
		},
	})
}

func aroRoleBinding(icazure.Credentials) ([]byte, error) {
	return yaml.Marshal(&rbacv1.RoleBinding{
		TypeMeta: metav1.TypeMeta{
			Kind:       "RoleBinding",
			APIVersion: "rbac.authorization.k8s.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "aro-cloud-provider-secret-read",
			Namespace: "kube-system",
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      "azure-cloud-provider",
				Namespace: "kube-system",
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "Role",
			Name:     "aro-cloud-provider-secret-reader",
		},
	})
}

func aroSecret(platformCreds icazure.Credentials) ([]byte, error) {
	// config is used to created compatible secret to trigger azure cloud
	// controller config merge behaviour
	// https://github.com/openshift/origin/blob/release-4.3/vendor/k8s.io/kubernetes/staging/src/k8s.io/legacy-cloud-providers/azure/azure_config.go#L82
	config := struct {
		AADClientID     string `json:"aadClientId" yaml:"aadClientId"`
		AADClientSecret string `json:"aadClientSecret" yaml:"aadClientSecret"`
	}{
		AADClientID:     platformCreds.ClientID,
		AADClientSecret: platformCreds.ClientSecret,
	}

	b, err := yaml.Marshal(config)
	if err != nil {
		return nil, err
	}

	return yaml.Marshal(&v1.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: corev1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "azure-cloud-provider",
			Namespace: "kube-system",
		},
		Data: map[string][]byte{
			"cloud-config": b,
		},
		Type: v1.SecretTypeOpaque,
	})
}
