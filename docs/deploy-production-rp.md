# Deploy production RP

Work in progress.

```
ADMIN_OBJECT_ID=$(az ad group list --query "[?displayName=='Engineering'].objectId" -o tsv)
COSMOSDB_ACCOUNT=$RESOURCEGROUP
DOMAIN=$RESOURCEGROUP.osadev.cloud
DOMAIN_NAME_LABEL=$RESOURCEGROUP
KEYVAULT_NAME=$RESOURCEGROUP
RP_IMAGE=
RP_IMAGE_AUTH=

az group create -g "$RESOURCEGROUP" -l "$LOCATION"

az group deployment create -g "$RESOURCEGROUP" \
  --mode complete \
  --template-file deploy/rp-production-debug.json \
  --parameters \
    "adminObjectId=$ADMIN_OBJECT_ID" \
    "azureFpClientId=$AZURE_FP_CLIENT_ID" \
    "databaseAccountName=$COSMOSDB_ACCOUNT" \
    "domainName=$DOMAIN" \
    "domainNameLabel=$DOMAIN_NAME_LABEL" \
    "keyvaultName=$KEYVAULT_NAME" \
    "location=$LOCATION" \
    "pullSecret=$PULL_SECRET" \
    "rpImage=$RP_IMAGE" \
    "rpImageAuth=$RP_IMAGE_AUTH" \
    "sshPublicKey=$(cat ~/.ssh/id_rsa.pub)"

PARENT_DNS_RESOURCEGROUP=dns

az network dns record-set ns create \
  --resource-group "$PARENT_DNS_RESOURCEGROUP" \
  --zone "$(cut -d. -f2- <<<"$DOMAIN")" \
  --name "$(cut -d. -f1 <<<"$DOMAIN")"

for ns in $(az network dns zone show --resource-group "$RESOURCEGROUP" --name "$DOMAIN" --query nameServers -o tsv); do
  az network dns record-set ns add-record \
    --resource-group "$PARENT_DNS_RESOURCEGROUP" \
    --zone "$(cut -d. -f2- <<<"$DOMAIN")" \
    --record-set-name "$(cut -d. -f1 <<<"$DOMAIN")" \
    --nsdname $ns
done
```
