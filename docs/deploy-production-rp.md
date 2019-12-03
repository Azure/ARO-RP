# Deploy production RP

Work in progress.

```
ADMIN_OBJECT_ID=$(az ad group list --query "[?displayName=='Engineering'].objectId" -o tsv)
COSMOSDB_ACCOUNT=$RESOURCEGROUP
DOMAIN=$RESOURCEGROUP.osadev.cloud
DOMAIN_NAME_LABEL=$RESOURCEGROUP
KEYVAULT_NAME=$RESOURCEGROUP

az group create -g "$RESOURCEGROUP" -l "$LOCATION"

az group deployment create -g "$RESOURCEGROUP" \
  --mode complete \
  --template-file deploy/rp-production-debug.json \
  --parameters \
    "adminObjectId=$ADMIN_OBJECT_ID" \
    "databaseAccountName=$COSMOSDB_ACCOUNT" \
    "domainName=$DOMAIN" \
    "domainNameLabel=$DOMAIN_NAME_LABEL" \
    "keyvaultName=$KEYVAULT_NAME" \
    "location=$LOCATION" \
    "sshPublicKey=$(cat ~/.ssh/id_rsa.pub)"
```
