# Rotate Certificates on Dev Subscription

Once every year, we need to rotate certificates on dev subscription. At the time of writing this, new certificates were generated on <u>Jan 5th, 2022</u>. This document goes over the steps taken to rotate the certificates.

1. Generate new certificates

```bash
go run ./hack/genkey -client firstparty
mv firstparty.* secrets

go run ./hack/genkey -ca vpn-ca
mv vpn-ca.* secrets

go run ./hack/genkey -client -keyFile secrets/vpn-ca.key -certFile secrets/vpn-ca.crt vpn-client
mv vpn-client.* secrets

go run ./hack/genkey proxy
mv proxy.* secrets

go run ./hack/genkey -client proxy-client
mv proxy-client.* secrets

ssh-keygen -f secrets/proxy_id_rsa -N ''

go run ./hack/genkey localhost
mv localhost.* secrets
```

2. Run import_certs_secrets. This will import certificates to keyvault and then set the secrets.

```bash
source ./hack/devtools/deploy-shared-env.sh
import_certs_secrets
```

3. The OpenVPN configuration file needs to be updated to enable tunneling to the vpn. To achieve this, edit the `vpn-<region>.ovpn` file in secrets and add the `vpn-client certificate` and `vpn-client certificate private key`.
   
4. At this point, we are done with all the certs except those owned by FP SP. The tenant owner needs to do this manually at the moment.

```bash
# Import firstparty.pem to keyvault 
az keyvault certificate import --vault-name v4-eastus-svc  --name rp-firstparty --file firstparty.pem
# Rotate client certificate credential for SP aro-v4-fp-shared
# Azure Active Directory > Search for app id / name > Verify certs have expired and upload new one

# Similar steps needs to be taken to update the ARM SP as well 
az keyvault certificate import --vault-name v4-eastus-svc  --name dev-arm --file arm.pem
```

4. In a development environment, the RP makes API calls to kubernetes cluster via a proxy vmss agent. To get the updated certs, this vm needs to be redeployed.
Proxy VM is currently deployed by the `deploy_env_dev` function in `deploy-shared-env.sh`. It makes use of `env-development.json`. <mark>The env-development.json also contains definition for CI VMs, Keyvault integration, vnet etc. along with proxy vm. Make sure to only deploy proxy VMs and comment out / remove everything else.</mark>
```bash 
source ./hack/devtools/deploy-shared-env.sh
deploy_env_dev
``` 

5. Finally, run `SECRET_SA_ACCOUNT_NAME=rharosecrets make secrets`