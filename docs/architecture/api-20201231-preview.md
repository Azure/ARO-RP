# API 20201231-preview design

For api `20201231-preview` bellow is the list of proposed changes and features.
These would have to land into API and backend as a features.

Feature work will need to be agreed and converted to working stories. All technical details bellow are overview only

## API changes

### .OpenShiftClusterProperties.SecurityProfile

Add security profile for all security enhancements for the product.
Proposal is to use new profile structure, so we do not scatter these options
to corresponding objects. In example, `EncryptionAtHost` can be set on individual
worker pools. But this creates possibility to configure some nodes with/without
encryption.

We need to make sure that these option are validated on the cluster too when customer is interacting with MachineSet objects. We should be able to verify
if these options where enabled on cluster create and set those accordingly.

This validation should be part of machine-api webhooks.

```
	// The cluster security profile
	SecurityProfile SecurityProfile `json:"securityProfile,omitempty"`
```

```
// SecurityProfile represents an security profile
type SecurityProfile struct {
	// EncryptionAtHost value sets encryptionAtHost option for all VirtualMachines.
	EncryptionAtHost *bool `json:"encryptionAtHost,omitempty"`

	// ManagedDiskEncryptionSetID represents Virtual Machine managed disk DiskEncryptionSetID
	// encryption
	ManagedDiskEncryptionSetID string `json:"diskEncryptionSetID,omitempty"`
}
```

### Extend OpenShiftClusterCredentials with GET

Extend `OpenShiftClusterCredentials` where current behaviour (POST to get credentials), is maintained. But if GET call is executed we allow to download kubeConfig for the cluster. This would allow customer to have system credentials in a cold storage if needed and access cluster in case of oAuth or Console outage.

```
	// Downloads kubeconfig file
	s.Methods(http.MethodGet).HandlerFunc(f.getOpenShiftClusterCredentials).Name("getOpenShiftClusterCredentials")

```

### Add Isolated compute SKUs

For Government cloud we need to extend our current support SKUs with isolated compute SKUs. With adding of https://github.com/Azure/ARO-RP/commit/213abf54a65ff17abeebc214c862a6c6b10c6d82 to our API this should be no-op change for new APIs in the future.

This has hard requirement on OpenShift being able to rotate master nodes,
as Isolated compute SKU's are retirable, so recovery process is pre-requisite.

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


