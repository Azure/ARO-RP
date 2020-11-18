# API 20201231-preview design

For api `20201231-preview` bellow is the list of proposed changes and features.
These would have to land into API and backend as a features.

Feature work will need to be agreed and converted to working stories. All technical details bellow are overview only

## API changes

### ComputeSecurityProfile

Add security profile to worker and master pools configuration for all security enhancements
for the compute instances

We need to make sure that these option are validated on the cluster too when customer is interacting with MachineSet objects. We should be able to verify if these options where enabled on cluster create and set those accordingly. This does not prevent customer to create instances with and without encryption in the same cluster.

```
// MasterProfile represents a master profile.
type MasterProfile struct {
	...

	ComputeSecurityProfile
}


// WorkerProfile represents a worker profile.
type WorkerProfile struct {
	...

	ComputeSecurityProfile
}
```

// EncryptionAtHostEnum enumerates the values for Encryption at host
type EncryptionAtHostEnum string

const (
	// Disabled ...
	Disabled EncryptionAtHostEnum = "Disabled"
	// Enabled ...
	Enabled EncryptionAtHostEnum = "Enabled"
)

// ComputeSecurityProfile represents an security profile for compute instance
type ComputeSecurityProfile struct {
	// EncryptionAtHost defines value encryptionAtHost option for all VirtualMachines.
	EncryptionAtHost EncryptionAtHostEnum `json:"encryptionAtHost,omitempty"`
	// / DiskEncryptionSetID defines resourceID for diskEncryptionSet resource. It must be in the same subscription
	DiskEncryptionSetID string `json:"diskEncryptionSetID,omitempty"`
}

```

### Extend OpenShiftClusterCredentials to provide kube-config

For backwards compatability we leave `OpenShiftClusterCredentials` as it is.
In the future we might want to refactor all this to Separate `AccessProfile` to
be aligned with AKS. But for now we just do simple extension.

Introduce new structure `OpenShiftClusterAdminCredentials` and extend frontend API with one more method to trigger download of the `adminKubeConfig`

```
// OpenShiftClusterCredentials represents a default an OpenShift cluster's
// console credentials
type OpenShiftClusterCredentials struct {
	...
}

// OpenShiftClusterAdminCredentials represents an OpenShift cluster's credentials
type OpenShiftClusterAdminCredentials struct {
	// KubeConfig - Base64-encoded Kubernetes configuration file.
	KubeConfig *[]byte `json:"kubeConfig,omitempty"`
}

```

Frontend changes:

Existing method `listcredentials` stays as it is.

```
	s = r.
		Path("/subscriptions/{subscriptionId}/resourcegroups/{resourceGroupName}/providers/{resourceProviderNamespace}/{resourceType}/{resourceName}/listadmincredentials")
		Queries("api-version", "{api-version}").
		Subrouter()

	s.Methods(http.MethodPost).HandlerFunc(f.postOpenShiftClusterAdminCredentials).Name("postOpenShiftClusterAdminCredentials")
```

### Add Isolated compute SKUs

For Government cloud we need to extend our current support SKUs with isolated compute SKUs. With adding of https://github.com/Azure/ARO-RP/commit/213abf54a65ff17abeebc214c862a6c6b10c6d82 to our API this should be no-op change for new APIs in the future.

This has hard requirement on OpenShift being able to rotate master nodes in place.
Isolated compute instances has retire date, this means they are not migrated in the backend and retire notice is given. After this SRE's must perform master node rotation in place to avoid downtime when retirement happens.

Isolated SKUs for decision making:

```
Standard_E64is_v3
Standard_E64i_v3
Standard_M128ms
Standard_GS5
Standard_G5
Standard_F72s_v2
```

### NetworkProfile

We should extend existing `NetworkProfile` with SDN plugin option for 4.6 onwards.

`SDNPlugin` - OpenShift will be changing its default SDN provider[1]. We can start
shipping ability to chose it on install, so customer can start testing, and we
will be able to switch it in the future release.


// SDNPluginName enumerates the values for Supported SDN plugins
type SDNPluginName string

const (
	// OpenShiftSDN ...
	OpenShiftSDN SDNPluginName = "OpenShiftSDN"
	// OVNKubernetes ...
	OVNKubernetes SDNPluginName = "OVNKubernetes"
)

```
// NetworkProfile represents a network profile.
type NetworkProfile struct {
	// The CIDR used for OpenShift/Kubernetes Pods (immutable).
	PodCIDR string `json:"podCidr,omitempty"`

	// The CIDR used for OpenShift/Kubernetes Services (immutable).
	ServiceCIDR string `json:"serviceCidr,omitempty"`

	// SDNPlugin defines SDN plugin, used in the cluster
	SDNPluginName SDNPluginName `json:"sdnPluginName,omitempty"`
}
```


1. https://docs.openshift.com/container-platform/4.6/networking/ovn_kubernetes_network_provider/about-ovn-kubernetes.html


# Reference:

AKS API Spec: https://github.com/Azure/azure-rest-api-specs/tree/master/specification/containerservice/resource-manager/Microsoft.ContainerService/stable/2020-11-01/examples and https://docs.microsoft.com/en-us/rest/api/container-instances/

