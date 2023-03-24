## Adding new instance types

Full support for new instance types in ARO relies on OpenShift support, Azure billing support, and RP support. The below document outlines how to introduce and test RP support for new instance types, after upstream OpenShift compatibility has been confirmed, and billing for each desired instance has been set up.

At the time of writing, new instance types need to be added in the following places:

- `pkg/api/openshiftcluster.go`
- `pkg/admin/api/openshiftcluster.go`
- `pkg/api/validate/vm.go`
    - If adding support for a new master instance, ensure that it is tested accordingly as we distinguish between master and worker support.

There are also vmSize consts in the `openshiftcluster.go` files of older versioned APIs, but this was deprecated in `v20220401` and is no longer necessary.

## Testing new instance types

First, confirm that the desired machines are available in the test region:
~~~
$ az vm list-skus --location westus --size Standard_L --all --output table
ResourceType     Locations    Name              Zones    Restrictions
---------------  -----------  ----------------  -------  --------------
virtualMachines  westus       Standard_L16s              None
virtualMachines  westus       Standard_L16s_v2           None
virtualMachines  westus       Standard_L32s              None
virtualMachines  westus       Standard_L32s_v2           None
virtualMachines  westus       Standard_L48s_v2           None
virtualMachines  westus       Standard_L4s               None
virtualMachines  westus       Standard_L64s_v2           None
virtualMachines  westus       Standard_L80s_v2           None
virtualMachines  westus       Standard_L8s               None
virtualMachines  westus       Standard_L8s_v2            None
~~~
The desired instance types should be free of any restrictions. The subscription should also have quota for the new instance types, which you may need to request.

### CLI Method

1) Comment out `FeatureRequireD2sV3Workers` from the range of features in `pkg/env/dev.go`. This will allow you to create development clusters with other VM sizes.

> __NOTE:__ Please be responsible with your usage of larger VM sizes, as they incur additional cost.

2) Follow the usual steps to [deploy a development RP](https://github.com/Azure/ARO-RP/blob/master/docs/deploy-development-rp.md), but don't use the hack script to create a cluster.

3) Follow steps in https://docs.microsoft.com/en-us/azure/openshift/tutorial-create-cluster to create a cluster, specifying `-worker-vm-size` and/or `--master-vm-size` in the `az aro create` step to specify an alternate sku:

~~~
az aro create   --resource-group $RESOURCEGROUP   --name $CLUSTER   --vnet aro-lseries   --master-subnet master-subnet   --worker-subnet worker-subnet   --worker-vm-size "Standard_L8s_v2"
~~~

4) Once an install with an alternate size is successful, a basic check of cluster health can be conducted, as well as local e2e tests to confirm supportability.

### Hack scripts method

1) Comment out `FeatureRequireD2sV3Workers` from the range of features in `pkg/env/dev.go`, and modify the worker and master profiles defined in `createCluster()` at `pkg/util/cluster/cluster.go` to contain your desired instance size. For example:
~~~
oc.Properties.WorkerProfiles[0].VMSize = api.VMSizeStandardL4s
~~~

2) Use the [hack script to create a cluster.](https://github.com/cadenmarchese/ARO-RP/blob/master/docs/deploy-development-rp.md#run-the-rp-and-create-a-cluster)

3) Once an install with an alternate size is successful, a basic check of cluster health can be conducted, as well as local e2e tests to confirm supportability.

### Post-install method

> __NOTE:__ This is useful for testing functionality of a specific size in an existing cluster. If adding support for a new size that is expected to be available on install, use the CLI method above.

1)  Follow the usual steps to [deploy a development RP](https://github.com/Azure/ARO-RP/blob/master/docs/deploy-development-rp.md), using the hack script or the MSFT documentation to create a cluster. 

2) Edit the `vmSize` in a worker MachineSet to the desired SKU.

3) Delete the corresponding machine object (`oc delete machine`) . The updated MachineSet will provision a new one with desired instance type.
~~~
$ oc get machines
NAME                                  PHASE     TYPE              REGION   ZONE   AGE
cluster-m9ttf-master-0               Running   Standard_D8s_v3   eastus   1      40m
cluster-m9ttf-master-1               Running   Standard_D8s_v3   eastus   2      40m
cluster-m9ttf-master-2               Running   Standard_D8s_v3   eastus   3      40m
cluster-m9ttf-worker-eastus1-86696   Running   Standard_L8s_v2   eastus   1      6m43s
cluster-m9ttf-worker-eastus2-tb5hn   Running   Standard_D2s_v3   eastus   2      34m
cluster-m9ttf-worker-eastus3-szf9d   Running   Standard_D2s_v3   eastus   3      34m
~~~