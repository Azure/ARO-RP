# API 20201231-preview design

For api `20201231-preview` bellow is the list of proposed changes and features.
These would have to land into API and backend as a features.

Feature work will need to be agreed and converted to working stories. All technical details bellow are overview only

## API changes

### .OpenShiftClusterProperties.SecurityProfile

Add security profile to worker pool configuration for all security enhancements
for the product compute resource.

We need to make sure that these option are validated on the cluster too when customer is interacting with MachineSet objects. We should be able to verify
if these options where enabled on cluster create and set those accordingly.

```
	// MasterProfile represents a master profile.

type MasterProfile struct {
	...

	ComputeSecurityProfile ComputeSecurityProfile `json:"securityProfile,omitempty"`
}


// WorkerProfile represents a worker profile.
type WorkerProfile struct {
	...

	ComputeSecurityProfile ComputeSecurityProfile `json:"securityProfile,omitempty"`
}
```

Re-use same structure for all compute profiles:
```
// ComputeSecurityProfile represents an security profile for all compute
type ComputeSecurityProfile struct {
	// EncryptionAtHost value sets encryptionAtHost option for all VirtualMachines.
	EncryptionAtHost *bool `json:"encryptionAtHost,omitempty"`
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
	// The username for the kubeadmin user
	KubeadminUsername string `json:"kubeadminUsername,omitempty"`

	// The password for the kubeadmin user
	KubeadminPassword string `json:"kubeadminPassword,omitempty"`
}

```

Frontend changes:

```
	s = r.
		Path("/subscriptions/{subscriptionId}/resourcegroups/{resourceGroupName}/providers/{resourceProviderNamespace}/{resourceType}/{resourceName}/listcredentials/listadminkubeconfig").
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

### Networking profile

We could start working on `NetworkingProfile` for few nit features we will want to
deliver in the future releases. Examples:

1.`SDNPlugin` - OpenShift will be changing its default SDN provider[1]. We can start
shipping ability to chose it on install, so customer can start testing, and we
will be able to switch it in the future relase.

2. `OutboundIPAdrressesCount` - if we set `disableOutboundSnat=false` in Azure cloud provider, any new Kubernetes `Loadbalancer.Type=LoadBalancer` will not be used
for outbound traffic. By adding `FrontendIPsCount` and `PortsPerInstance` we would enable customer to control their outbound traffic behaviour. `FrondendIps` would
have to be provisioned by RP backend, to control SNAT behaviour.

```
// NetworkingProfile allows customer to configure networking settings on the
// clusters
type NetworkingProfile struct {
    // SDNPlugin allows override current default SDNPlugin with other. In this
    // case future Kubernetes OVN[1]
    SDNPlugin string `json:"sdnPlugin,omitempty"`
    FrontendIPsCount: 2 `int:"frontendIPsCount,omitempty"`
    PortsPerInstance: 2048   `int:"portsPerInstance,omitempty"`
}
```


1. https://docs.openshift.com/container-platform/4.6/networking/ovn_kubernetes_network_provider/about-ovn-kubernetes.html


# Reference:

AKS API Spec: https://github.com/Azure/azure-rest-api-specs/tree/master/specification/containerservice/resource-manager/Microsoft.ContainerService/stable/2020-11-01/examples and https://docs.microsoft.com/en-us/rest/api/container-instances/

