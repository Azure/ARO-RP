# Deploy production RP

Work in progress.

```
COSMOSDB_ACCOUNT=$RESOURCEGROUP
DOMAIN=$RESOURCEGROUP.osadev.cloud
KEYVAULT_NAME=$RESOURCEGROUP
RP_IMAGE=
RP_IMAGE_AUTH=

az group create -g "$RESOURCEGROUP" -l "$LOCATION"

az group deployment create -g "$RESOURCEGROUP" \
  --template-file deploy/rp-production-nsg.json

# use rpServicePrincipalId from this step in the next step

az group deployment create -g "$RESOURCEGROUP" \
  --template-file deploy/rp-production.json \
  --parameters \
    "databaseAccountName=$COSMOSDB_ACCOUNT" \
    "domainName=$DOMAIN" \
    "keyvaultName=$KEYVAULT_NAME" \
    "pullSecret=$PULL_SECRET" \
    "rpImage=$RP_IMAGE" \
    "rpImageAuth=$RP_IMAGE_AUTH" \
    "rpServicePrincipalId=$RP_SERVICEPRINCIPAL_ID" \
    "sshPublicKey=$(cat ~/.ssh/id_rsa.pub)"

# Load certificate into key vault

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
