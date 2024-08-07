# Using a custom Disk Encryption Set

## What is the Disk Encryption Set used for?

In summary, it allows a customer to control the keys that are used to encrypt/decrypt VM disks.
See [deploy-a-vm-with-customer-managed-keys](https://docs.microsoft.com/en-us/azure/virtual-machines/disks-enable-host-based-encryption-portal#deploy-a-vm-with-customer-managed-keys) for more information.

## How to deploy?
First, install and use the AzureCLI extension with
```bash
make az
```

>You can check if the extension is in use by running:
```bash
az extension list
[
  {
    "experimental": false,
    "extensionType": "dev",
    "name": "aro",
    "path": "<path to go SRC>/github.com/Azure/ARO-RP/python/az/aro",
    "preview": true,
    "version": "1.0.1"
  }
]
```

Follow [tutorial-create-cluster](https://docs.microsoft.com/en-us/azure/openshift/tutorial-create-cluster) but don't run the `az aro create` command, instead proceed as follows:

  - set additional env variables
```bash
# use a unique prefix for Azure resources when it is set, otherwise use your user's name
export AZURE_PREFIX="${AZURE_PREFIX:-$USER}"
export KEYVAULT_NAME=$AZURE_PREFIX-enckv
export KEYVAULT_KEY_NAME=$AZURE_PREFIX-key
export DISK_ENCRYPTION_SET_NAME=$RESOURCEGROUP-disk-encryption-set
```
  - create the KeyVault and Key
```bash
az keyvault create -n $KEYVAULT_NAME \
                   -g $RESOURCEGROUP \
                   -l $LOCATION \
                   --enable-purge-protection true

az keyvault key create --vault-name $KEYVAULT_NAME \
                       -n $KEYVAULT_KEY_NAME \
                       --protection software

KEYVAULT_ID=$(az keyvault show --name $KEYVAULT_NAME -g $RESOURCEGROUP --query "[id]" -o tsv)

KEYVAULT_KEY_URL=$(az keyvault key show --vault-name $KEYVAULT_NAME \
                                        --name $KEYVAULT_KEY_NAME \
                                        --query "[key.kid]" -o tsv)
```
  - create the DES and add permissions to use the KeyVault
```bash
az disk-encryption-set create -n $DISK_ENCRYPTION_SET_NAME \
                              -l $LOCATION \
                              -g $RESOURCEGROUP \
                              --source-vault $KEYVAULT_ID \
                              --key-url $KEYVAULT_KEY_URL

DES_IDENTITY=$(az disk-encryption-set show -n $DISK_ENCRYPTION_SET_NAME \
                                           -g $RESOURCEGROUP \
                                           --query "[identity.principalId]" \
                                           -o tsv)

DES_ID=$(az disk-encryption-set show -n $DISK_ENCRYPTION_SET_NAME -g $RESOURCEGROUP --query 'id' -o tsv)                                    

az keyvault set-policy -n $KEYVAULT_NAME \
                       -g $RESOURCEGROUP \
                       --object-id $DES_IDENTITY \
                       --key-permissions wrapkey unwrapkey get

```
  - run the az aro create command
```bash
az aro create --resource-group $RESOURCEGROUP \
              --name $CLUSTER  \
              --vnet aro-vnet  \
              --master-subnet master-subnet \
              --worker-subnet worker-subnet \
              --disk-encryption-set $DES_ID
```

After creating the cluster all VMs should have the customer controlled Disk Encryption Set.
>Remember to delete the disk-encryption-set and keyvault when done.
