#!/bin/bash -e
######## Helper file to automate the Shared RP Development Environment creation ########
# Automate https://github.com/Azure/ARO-RP/blob/master/docs/prepare-a-shared-rp-development-environment.md

prerequisites() {
    # Prerequisites
    err_str="Usage $0 <SECRET_SA_ACCOUNT_NAME> <PREFIX> <LOCATION>  <PARENT_DOMAIN_RESOURCEGROUP>. Please try again"
    local secret_sa_account_name=${1?$err_str}
    local prefix=${2?$err_str}
    local location=${3?$err_str}
    local parent_domain_resourcegroup=${4?$err_str}

    echo -e "#### Prerequisites ####"
    global_resourcegroup=global-infra-${prefix}
    echo -e "\n#### Create global resourcGroup $global_resourcegroup ####"
    az group create -n $global_resourcegroup --location ${location}
    export PARENT_DOMAIN_NAME=${prefix}.osadev.cloud

    echo -e "\n#### Create global dns zone $PARENT_DOMAIN_NAME ####"
    az network dns zone create \
        --name $PARENT_DOMAIN_NAME \
        -g ${parent_domain_resourcegroup}
    export SECRET_SA_ACCOUNT_NAME=${secret_sa_account_name}
    echo -e "\n#### Create deployment e2esecretstorage ####"
    # ./hack/devtools/deploy-shared-env-storage.sh
    if check_deployment ${parent_domain_resourcegroup} e2esecretstorage; then
        log "⏩📋 e2esecretstorage deployment was skipped"
    else
        az deployment group create \
            --name e2esecretstorage \
            --resource-group ${parent_domain_resourcegroup} \
            --parameters storageAccounts_e2earosecrets_name=${secret_sa_account_name} \
            --template-file pkg/deploy/assets/e2e-secret-storage.json
        log "e2esecretstorage has been deployed"
    fi

    # export ADMIN_OBJECT_ID="$(az ad group show -g aro-engineering --query id -o tsv)"
    # export PULL_SECRET='dummy'
    # export AZURE_TENANT_ID=$(az account show --query tenantId -o tsv)
    # export AZURE_SUBSCRIPTION_ID=$(az account show --query id -o tsv)
    mkdir -p secrets
}

aad_applications() {
    # AAD applications
    err_str="Usage $0 <PREFIX> <LOCATION>. Please try again"
    local prefix=${1?$err_str}
    local location=${2?$err_str}

    echo -e "#### AAD applications ####\n"
    aad_prefix=aro-v4-${prefix}
    echo -e "#### (1) Fake up the ARM layer ####"
    go run ./hack/genkey -client arm
    mv arm.* secrets
    arm_client_info=$(az ad app list --display-name ${aad_prefix}-arm-shared 2>/dev/null)
    if [ "${arm_client_info}"  == "[]" ]; then
        echo -e "\n#### (1) Create the fake up ARM layer ####"
        export AZURE_ARM_CLIENT_ID="$(az ad app create \
            --display-name ${aad_prefix}-arm-shared \
            --query appId \
            -o tsv)"
        az ad app credential reset \
            --id "$AZURE_ARM_CLIENT_ID" \
            --cert "$(base64 -w0 <secrets/arm.crt)" >/dev/null
        az ad sp create --id "$AZURE_ARM_CLIENT_ID" >/dev/null
        
    else
        echo -e "\n#### (1) Skip the fake up ARM layer ####"
    fi

    echo -e "\n#### (2) Fake up the first party application ####"
    go run ./hack/genkey -client firstparty
    mv firstparty.* secrets
    fp_client_info=$(az ad app list --display-name ${aad_prefix}-fp-shared 2>/dev/null)
    if [ "${fp_client_info}"  == "[]" ]; then
        echo -e "\n#### (2) Create the fake up first party application ####"
        export AZURE_FP_CLIENT_ID='$(az ad app create \
            --display-name "'${aad_prefix}'-fp-shared" \
            --query appId \
            -o tsv)'
        az ad app credential reset \
            --id "$AZURE_FP_CLIENT_ID" \
            --cert "$(base64 -w0 <secrets/firstparty.crt)" >/dev/null
        az ad sp create --id "$AZURE_FP_CLIENT_ID" >/dev/null
        export AZURE_FP_SERVICE_PRINCIPAL_ID='$(az ad sp list \
            --filter "appId eq '$AZURE_FP_CLIENT_ID'" \
            --query '[].id' \
             -o tsv)'
    else
        echo -e "\n#### (2) Skip the fake up first party application ####"
    fi

    export AZURE_RP_CLIENT_SECRET="$(uuidgen)"
    echo -e "\n#### (3) Fake up the RP identity with secret $AZURE_RP_CLIENT_SECRET ####"
    rp_identity_info=$(az ad app list --display-name ${prefix}-rp-shared 2>/dev/null)
    if [ "${rp_identity_info}" == "[]" ]; then
        echo -e "\n#### (3) Create the fake RP identity ####"
        az ad app create \
            --display-name ${prefix}-rp-shared \
            --end-date '2299-12-31T11:59:59+00:00' \
            --key-type Password \
            --key-value "$AZURE_RP_CLIENT_SECRET" \
            --debug \
            -o tsv
        # export AZURE_RP_CLIENT_ID="$(az ad app create \
        #     --display-name ${prefix}-rp-shared \
        #     --end-date '2299-12-31T11:59:59+00:00' \
        #     --key-type Password \
        #     --key-value "$AZURE_RP_CLIENT_SECRET" \
        #     --query appId \
        #     --debug \
        #     -o tsv)"
        # az ad sp create --id "$AZURE_RP_CLIENT_ID" >/dev/null
    else
        echo -e "\n#### (3) Skip the fake RP identity ####"
    fi

    export AZURE_GATEWAY_CLIENT_SECRET="$(uuidgen)"
    echo -e "\n#### (4) Fake up the GWY identity with secret $AZURE_GATEWAY_CLIENT_SECRET ####"
    gwy_identity_info=$(az ad app list --display-name ${prefix}-gateway-shared 2>/dev/null)
    if [ "${gwy_identity_info}" == "[]" ]; then
        echo -e "\n#### (4) Create the fake GWY identity ####"
        export AZURE_GATEWAY_CLIENT_ID='$(az ad app create \
            --display-name ${prefix}-gateway-shared \
            --end-date '2299-12-31T11:59:59+00:00' \
            --key-type password \
            --password "$AZURE_GATEWAY_CLIENT_SECRET" \
            --query appId \
            -o tsv)'
        az ad sp create --id "$AZURE_GATEWAY_CLIENT_ID" >/dev/null
        export AZURE_GATEWAY_SERVICE_PRINCIPAL_ID='$(az ad sp list \
            --filter "appId eq '$AZURE_GATEWAY_CLIENT_ID'" \
            --query '[].id' \
            -o tsv)'
    else
        echo -e "\n#### (4) Skip the fake GWY identity ####"
    fi

    export AZURE_CLIENT_SECRET="$(uuidgen)"
    echo -e "\n#### (5) E2E and tooling client with secret $AZURE_CLIENT_SECRET ####"
    client_identity_info=$(az ad app list --display-name ${prefix}-tooling-shared 2>/dev/null)
    if [ "${client_identity_info}" == "[]" ]; then
        echo -e "\n#### (5) Create the E2E and tooling client ####"
        export AZURE_CLIENT_ID='$(az ad app create \
            --display-name ${prefix}-tooling-shared \
            --end-date '2299-12-31T11:59:59+00:00' \
            --key-type password \
            --password "$AZURE_CLIENT_SECRET" \
            --query appId \
            -o tsv)'
        az ad sp create --id "$AZURE_CLIENT_ID" >/dev/null
        export AZURE_SERVICE_PRINCIPAL_ID='$(az ad sp list \
            --filter "appId eq '$AZURE_CLIENT_ID'" \
            --query '[].id' \
            -o tsv)'        
    else
        echo -e "\n#### (5) Skip the E2E and tooling client ####"
    fi

    echo -e "\n#### (5-b) Set up for E2E and tooling client with Microsoft.Graph/Application.ReadWrite.OwnedBy permission  (Manuel) ####"

    echo -e "\n#### (6) Set up the RP role definitions and subscription role assignments at ${location} ####"
    echo -e "\n#### (6) No verification ####"
    az deployment sub create \
        -l ${location} \
        --template-file pkg/deploy/assets/rbac-development.json \
        --parameters \
            "armServicePrincipalId=$(az ad sp list --filter "appId eq '$AZURE_ARM_CLIENT_ID'" --query '[].id' -o tsv)" \
            "fpServicePrincipalId=$(az ad sp list --filter "appId eq '$AZURE_FP_CLIENT_ID'" --query '[].id' -o tsv)" \
            "fpRoleDefinitionId"="$(uuidgen)" \
            "devServicePrincipalId=$(az ad sp list --filter "appId eq '$AZURE_CLIENT_ID'" --query '[].id' -o tsv)" \
         >/dev/null
    
    echo -e "\n#### (7) Fake up the portal client ${prefix}-portal-shared ####"
    portal_client_info=$(az ad app list --display-name  ${prefix}-portal-shared 2>/dev/null)
    if [ "${portal_client_info}" == "[]" ]; then
        echo -e "\n#### (7) Fake up the portal client ####"
        export AZURE_PORTAL_CLIENT_ID="$(az ad app create \
            --display-name ${prefix}-portal-shared \
            --query appId \
            -o tsv)"

        OBJ_ID="$(az ad app show --id $AZURE_PORTAL_CLIENT_ID --query id -o tsv)"

        az rest --method PATCH \
            --uri "https://graph.microsoft.com/v1.0/applications/$OBJ_ID" \
            --headers 'Content-Type=application/json' \
            --body '{"web":{"redirectUris":["https://locahlost:8444/callback"]}}'

        az ad app credential reset \
            --id "$AZURE_PORTAL_CLIENT_ID" \
            --cert "$(base64 -w0 <secrets/portal-client.crt)" >/dev/null
    else
        echo -e "\n#### (7) Skip the portal client ####"
    fi
    echo "Finish aad_applications"
}

certificates(){
    # Certificates
    echo -e "#### Certificates ####\n"
    echo -e "#### Generate key/certificate file using an helper utility ####"
    # TODO- consider checking whether each key/certificate already exists under secrets/

    echo -e "#### (1) VPN CA key/certificate ####\n"
    go run ./hack/genkey -ca vpn-ca
    mv vpn-ca.* secrets

    echo -e "#### (2) VPN client key/certificate ####\n"
    go run ./hack/genkey -client -keyFile secrets/vpn-ca.key -certFile secrets/vpn-ca.crt vpn-client
    mv vpn-client.* secrets

    echo -e "#### (3) proxy serving key/certificate ####\n"
    go run ./hack/genkey proxy
    mv proxy.* secrets

    echo -e "#### (4) proxy client key/certificate ####\n"
    go run ./hack/genkey -client proxy-client
    mv proxy-client.* secrets

    echo -e "#### (5) proxy ssh key/certificate ####\n"
    ssh-keygen -f secrets/proxy_id_rsa -N ''

    echo -e "#### (6) RP serving key/certificate ####\n"
    go run ./hack/genkey localhost
    mv localhost.* secrets

    echo -e "####  (7)) CA key/certificate ####\n"
    go run ./hack/genkey -ca dev-ca
    mv dev-ca.* secrets

    echo -e "#### (8) dev client key/certificate ####\n"
    go run ./hack/genkey -client -keyFile secrets/dev-ca.key -certFile secrets/dev-ca.crt dev-client
    mv dev-client.* secrets

    echo -e "#### (9) CA key/certificate ####\n"
    go run ./hack/genkey cluster-mdsd
    mv cluster-mdsd.* secrets

    echo "Finish certificates"
}

env_file(){
    # Environment file
    echo -e "#### Environment file ####\n"
    err_str="Usage $0 <SECRET_SA_ACCOUNT_NAME> <PARENT_DOMAIN_RESOURCEGROUP> <RESOURCEGROUP_PREFIX> <PROXY_DOMAIN_NAME_LABEL>. Please try again"
    local secret_sa_account_name=${1?$err_str}
    local parent_domain_resourcegroup=${2?$err_str}
    local resourcegroup_prefix=${3?$err_str}
    local proxy_domain_name_label=${4?$err_str}
    
    local admin_object_id="$(az ad group show -g aro-engineering --query id -o tsv)"
    local pull_secret='dummy'
    local azure_tenant_id=$(az account show --query tenantId -o tsv)
    local azure_subscription_id=$(az account show --query id -o tsv)

    # local azure_tenant_id="abcd1"
    # local azure_subscription_id="abcd2"
    local azure_arm_client_id="abcd3"
    local azure_fp_client_id="abcd4"
    local azure_fp_service_principal_id="abcd5"
    local azure_portal_client_id="abcd14"
    # local admin_object_id="abcd6"
    local azure_client_id="abcd7"
    local azure_service_principal_id="abcd8"
    local azure_client_secret="abcd9"
    local azure_rp_client_id="abcd15"
    local azure_rp_client_secret="abcd10"
    local azure_gateway_client_id="abcd11"
    local azure_gateway_service_principal_id="abcd12"
    local azure_gateway_client_secret="abcd13"
    # TODO: How to get the USER_PULL_SECRET 

    echo -e "#### (1) Generate SSH key for VMSS access ####\n"
    ssh-keygen -t rsa -N "" -f secrets/full_rp_id_rsa
    echo -e "#### (2) Create the secrets/env file ####\n"
    # use a unique prefix for Azure resources when it is set, otherwise use your user's name
    cat >secrets/env <<EOF
    export AZURE_PREFIX='${AZURE_PREFIX:-$USER}'
    export ADMIN_OBJECT_ID='$admin_object_id'
    export AZURE_TENANT_ID='$azure_tenant_id'
    export AZURE_SUBSCRIPTION_ID='$azure_subscription_id'
    export AZURE_ARM_CLIENT_ID='$azure_arm_client_id'
    export AZURE_FP_CLIENT_ID='$azure_fp_client_id'
    export AZURE_FP_SERVICE_PRINCIPAL_ID='$azure_fp_service_principal_id'
    export AZURE_PORTAL_CLIENT_ID='$azure_portal_client_id'
    export AZURE_PORTAL_ACCESS_GROUP_IDS='$admin_object_id'
    export AZURE_PORTAL_ELEVATED_GROUP_IDS='$admin_object_id'
    export AZURE_CLIENT_ID='$azure_client_id'
    export AZURE_SERVICE_PRINCIPAL_ID='$azure_service_principal_id'
    export AZURE_CLIENT_SECRET='$azure_client_secret'
    export AZURE_RP_CLIENT_ID='$azure_rp_client_id'
    export AZURE_RP_CLIENT_SECRET='$azure_rp_client_secret'
    export AZURE_GATEWAY_CLIENT_ID='$azure_gateway_client_id'
    export AZURE_GATEWAY_SERVICE_PRINCIPAL_ID='$azure_gateway_service_principal_id'
    export AZURE_GATEWAY_CLIENT_SECRET='$azure_gateway_client_secret'
    export RESOURCEGROUP='$resourcegroup_prefix-\$LOCATION'
    export PROXY_HOSTNAME='vm0.$proxy_domain_name_label.\$LOCATION.cloudapp.azure.com'
    export DATABASE_NAME='\$AZURE_PREFIX'
    export RP_MODE='development'
    export PULL_SECRET='PULL_SECRET'
    export USER_PULL_SECRET='USER_PULL_SECRET'
    export SECRET_SA_ACCOUNT_NAME='${secret_sa_account_name}'
    export DATABASE_ACCOUNT_NAME='\$RESOURCEGROUP'
    export KEYVAULT_PREFIX='\$RESOURCEGROUP'
    export PARENT_DOMAIN_NAME='$PARENT_DOMAIN_NAME'
    export PARENT_DOMAIN_RESOURCEGROUP='${parent_domain_resourcegroup}'
    export DOMAIN_NAME='\$LOCATION.\$PARENT_DOMAIN_NAME'
    export AZURE_ENVIRONMENT='AzurePublicCloud'
    export OIDC_STORAGE_ACCOUNT_NAME='\${RESOURCEGROUP}oic'
    export SSH_PRIVATE_KEY='secrets/full_rp_id_rsa'
    export SSH_PUBLIC_KEY='secrets/full_rp_id_rsa.pub'
EOF
    # export AZURE_GATEWAY_SERVICE_PRINCIPAL_ID='$AZURE_GATEWAY_SERVICE_PRINCIPAL_ID'
    echo -e "#### (4) Upload the secrets/env file to the storage account ####\n"
    make secrets-update
    echo "Finish env_file"
}

deploy_shared_rp(){
    # Deploy Shared RP Development Environment
    echo -e "#### Deploy Shared RP Development Environment ####\n"
    err_str="Usage $0 <SECRET_SA_ACCOUNT_NAME> <PARENT_DOMAIN_RESOURCEGROUP> <RESOURCEGROUP_PREFIX> <PROXY_DOMAIN_NAME_LABEL>. Please try again"
    local secret_sa_account_name=${1?$err_str}
    local parent_domain_resourcegroup=${2?$err_str}
    local resourcegroup_prefix=${3?$err_str}
    local proxy_domain_name_label=${4?$err_str}
    echo -e "#### (1) Tweak and source your environment file - Not sure it is needed ####\n"
    # cp env.example env
    # vi env
    # . ./env

    echo -e "#### (2) Create AzSecPack managed Identity - Manuel? ####\n"
    # This step is required for 'deploy_env_dev' -  https://msazure.visualstudio.com/ASMDocs/_wiki/wikis/ASMDocs.wiki/234249/AzSecPack-AutoConfig-UserAssigned-Managed-Identity
    curl /subscriptions/fe16a035-e540-4ab7-80d9-373fa9a3d6ae/resourceGroups/AzSecPackAutoConfigRG/providers/Microsoft.ManagedIdentity/userAssignedIdentities/AzSecPackAutoConfigUA-westcentralus

    echo -e "#### (3) Enable EncryptionAtHost for subscription ####\n"
    az feature register --namespace Microsoft.Compute --name EncryptionAtHost 

    echo -e "#### (4) Create the resource group and deploy the RP resources ####\n"
    source hack/devtools/deploy-shared-env.sh
    # Create the RG
    create_infra_rg
    # Deploy the predeployment ARM template
    deploy_rp_dev_predeploy
    # Deploy the infrastructure resources such as Cosmos, KV, Vnet...
    deploy_rp_dev
    # Deploy RP MSI for aks/hive
    deploy_rp_managed_identity
    # Deploy the proxy and VPN
    deploy_env_dev
    # Deploy AKS resources for Hive
    deploy_aks_dev
    # Deploy additional infrastructure required for workload identity clusters
    deploy_miwi_infra_dev
    # If you encounter a "VirtualNetworkGatewayCannotUseStandardPublicIP" error when running the deploy_env_dev command, you have to override two additional parameters. Run this command instead:
    #  deploy_env_dev_override
    # If you encounter a "SkuCannotBeChangedOnUpdate" error when running the deploy_env_dev_override command, delete the -pip resource and re-run.

    echo -e "#### (5) Get the AKS kubeconfig and upload it to the storage account ####\n"
    make aks.kubeconfig
    mv aks.kubeconfig secrets/
    make secrets-update
    
    # TODO: Find the aks vpn files that are missing
    echo -e "#### (6) Install Hive ####\n"
    HOME=/usr KUBECONFIG=$(pwd)/secrets/aks.kubeconfig ./hack/hive/hive-dev-install.sh

    echo -e "#### (7) Load the keys/certificates into the key vault ####\n"
    import_certs_secrets

    echo -e "#### (?) Anymore steps? ####\n"

    echo "Finish deploy_shared_rp"

}

certificate_rotation(){
    # Certificate Rotation
    echo -e "#### Certificate Rotation ####\n"
    echo -e "#### (1) rotate certificates in dev and INT subscriptions after running aad_applications and certificates ####"
    source hack/devtools/deploy-shared-env.sh
    echo -e "#### (2) dev client key/certificate ####\n"
    import_certs_secrets

    echo -e "#### (3) Update the Azure VPN Gateway configuration - 'Manuel' ####\n"
    echo -e "#### (4) OpenVPN configuration file - 'Manuel' ####\n"
    echo -e "#### (5) Update certificates owned by FP Service Principal ####\n"
    # Import firstparty.pem to keyvault v4-eastus-svc
    az keyvault certificate import --vault-name <kv_name>  --name rp-firstparty --file firstparty.pem

    # Rotate certificates for SPs ARM, FP, and PORTAL (wherever applicable)
    az ad app credential reset \
        --id "$AZURE_ARM_CLIENT_ID" \
        --cert "$(base64 -w0 <secrets/arm.crt)" >/dev/null

    az ad app credential reset \
        --id "$AZURE_FP_CLIENT_ID" \
        --cert "$(base64 -w0 <secrets/firstparty.crt)" >/dev/null

    az ad app credential reset \
        --id "$AZURE_PORTAL_CLIENT_ID" \
        --cert "$(base64 -w0 <secrets/portal-client.crt)" >/dev/null

    echo -e "#### (6)  VM needs to be deleted & redeployed - 'Manuel'? ####\n"

    echo -e "#### (7) Upload the secrets to the storage account ####\n"
    # [rharosecretsdev|e2earosecrets|e2earoclassicsecrets] make secrets-update
    # SECRET_SA_ACCOUNT_NAME=[rharosecretsdev|e2earosecrets|e2earoclassicsecrets] make secrets-update

    echo "Finish certificate_rotation"
}
