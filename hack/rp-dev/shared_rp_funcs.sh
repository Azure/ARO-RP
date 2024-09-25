#!/bin/bash -e
######## Helper file to automate the Shared RP Development Environment creation ########
# Automate https://github.com/Azure/ARO-RP/blob/master/docs/prepare-a-shared-rp-development-environment.md

set -o errexit \
    -o nounset \
    -o monitor

prerequisites() {
    # Prerequisites
    err_str="Usage $0 <SECRET_SA_ACCOUNT_NAME> <PREFIX> <LOCATION>  <PARENT_DOMAIN_RESOURCEGROUP>. Please try again"
    local secret_sa_account_name=${1?$err_str}
    local prefix=${2?$err_str}
    local location=${3?$err_str}
    local parent_domain_resourcegroup=${4?$err_str}

    # global_resourcegroup=global-infra-${prefix}
    # log "Create global resourcGroup $global_resourcegroup"
    # az group create -n "$global_resourcegroup" --location "${location}"
    log "Create global resourcGroup $parent_domain_resourcegroup"
    az group create --name "$parent_domain_resourcegroup" --location "${location}"
    export PARENT_DOMAIN_NAME=${prefix}.osadev.cloud

    log "Create global dns zone $PARENT_DOMAIN_NAME"
    az network dns zone create \
        --name "$PARENT_DOMAIN_NAME" \
        --resource-group "${parent_domain_resourcegroup}"
    export SECRET_SA_ACCOUNT_NAME=${secret_sa_account_name}
    local secret_storage_resourcegroup="secretstorage-${prefix}"
    log "Create deployment secretstorage under resource group ${secret_storage_resourcegroup}"
    # ./hack/devtools/deploy-shared-env-storage.sh
    if check_deployment "${parent_domain_resourcegroup}" secretstorage; then
        log "⏩📋 secretstorage deployment was skipped"
    else
        az group create --name "${secret_storage_resourcegroup}" --location "${location}"
        az deployment group create \
            --name secretstorage \
            --resource-group "${secret_storage_resourcegroup}" \
            --parameters storageAccounts_arosecrets_name="${SECRET_SA_ACCOUNT_NAME}" \
            --template-file pkg/deploy/assets/shared-rp-secret-storage.json 
        log "secretstorage has been deployed"
    fi

    # Generate new secrets directory
    local secrets_dir="secrets"
    if [ -d "$secrets_dir" ]; then
        rm -R $secrets_dir
    fi
    mkdir -p $secrets_dir
}

aad_applications() {
    # AAD applications
    err_str="Usage $0 <PREFIX> <LOCATION>. Please try again"
    local prefix=${1?$err_str}
    local location=${2?$err_str}
    local endless_date="2299-12-31T11:59:59+00:00"

    log "(1) Fake up the ARM layer"
    go run ./hack/genkey -client arm
    mv arm.* secrets
    local arm_client_info="$(az ad app list --display-name "${AAD_PREFIX}-arm-shared" 2>/dev/null)"
    if [ "${arm_client_info}"  == "[]" ]; then
        log "(1) Create the fake up ARM layer"
        export AZURE_ARM_CLIENT_ID="$(az ad app create \
            --display-name "${AAD_PREFIX}-arm-shared" \
            --query appId \
            -o tsv)"
        az ad app credential reset \
            --id "$AZURE_ARM_CLIENT_ID" \
            --cert "$(base64 -w0 <secrets/arm.crt)" >/dev/null
        az ad sp create --id "$AZURE_ARM_CLIENT_ID" >/dev/null
        
    else
        export AZURE_ARM_CLIENT_ID="$(az ad app list \
             --filter "displayname eq '${AAD_PREFIX}-arm-shared'" \
             --query '[].appId' \
             -o tsv)"
        log "(1) ⏩🔑 Skip the fake up ARM layer with application id $AZURE_ARM_CLIENT_ID"
    fi

    log "(2) Fake up the first party application"
    go run ./hack/genkey -client firstparty
    mv firstparty.* secrets
    local fp_client_info="$(az ad app list --display-name "${AAD_PREFIX}-fp-shared" 2>/dev/null)"
    if [ "${fp_client_info}"  == "[]" ]; then
        log "(2) Create the fake up first party application"
        export AZURE_FP_CLIENT_ID="$(az ad app create \
            --display-name "${AAD_PREFIX}-fp-shared" \
            --query appId \
            -o tsv)"
        az ad app credential reset \
            --id "$AZURE_FP_CLIENT_ID" \
            --cert "$(base64 -w0 <secrets/firstparty.crt)" >/dev/null
        az ad sp create --id "$AZURE_FP_CLIENT_ID" >/dev/null
    else
        export AZURE_FP_CLIENT_ID="$(az ad app list \
             --filter "displayname eq '${AAD_PREFIX}-fp-shared'" \
             --query '[].appId' \
             -o tsv)"
        log "(2) ⏩🔑 Skip the fake up first party application with application id $AZURE_FP_CLIENT_ID"
    fi

    export AZURE_RP_CLIENT_SECRET="$(openssl rand -base64 32)"
    log "(3) Fake up the RP identity with secret $AZURE_RP_CLIENT_SECRET"
    local rp_identity_info="$(az ad app list --display-name "${AAD_PREFIX}-rp-shared" 2>/dev/null)"
    if [ "${rp_identity_info}" == "[]" ]; then
        log "(3) Create the fake RP identity"
        export AZURE_RP_CLIENT_ID="$(az ad app create \
            --display-name "${AAD_PREFIX}-rp-shared" \
            --start-date "$(date -Iseconds)" \
            --end-date $endless_date \
            --key-type Symmetric \
            --key-usage Sign \
            --key-value "$AZURE_RP_CLIENT_SECRET" \
            --query appId \
            -o tsv)"
        az ad sp create --id "$AZURE_RP_CLIENT_ID" >/dev/null
    else
        export AZURE_RP_CLIENT_ID="$(az ad app list \
             --filter "displayname eq '${AAD_PREFIX}-rp-shared'" \
             --query '[].appId' \
             -o tsv)"
        log "(3) ⏩🔑 Skip the fake RP identity with application id $AZURE_RP_CLIENT_ID"
    fi

    export AZURE_GATEWAY_CLIENT_SECRET="$(openssl rand -base64 32)"
    log "(4) Fake up the GWY identity with secret $AZURE_GATEWAY_CLIENT_SECRET"
    local gwy_identity_info="$(az ad app list --display-name "${AAD_PREFIX}-gateway-shared" 2>/dev/null)"
    if [ "${gwy_identity_info}" == "[]" ]; then
        log "(4) Create the fake GWY identity"
        export AZURE_GATEWAY_CLIENT_ID="$(az ad app create \
            --display-name "${AAD_PREFIX}-gateway-shared" \
            --start-date "$(date -Iseconds)" \
            --end-date $endless_date \
            --key-type Symmetric \
            --key-usage Sign \
            --key-value "$AZURE_GATEWAY_CLIENT_SECRET" \
            --query appId \
            -o tsv)"   
        az ad sp create --id "$AZURE_GATEWAY_CLIENT_ID" >/dev/null
    else
        export AZURE_GATEWAY_CLIENT_ID="$(az ad app list \
             --filter "displayname eq '${AAD_PREFIX}-gateway-shared'" \
             --query '[].appId' \
             -o tsv)"
        log "(4) ⏩🔑 Skip the fake GWY identity with application id $AZURE_GATEWAY_CLIENT_ID"
    fi

    export AZURE_CLIENT_SECRET="$(openssl rand -base64 32)"
    log "(5) E2E and tooling client with secret $AZURE_CLIENT_SECRET"
    local client_identity_info="$(az ad app list --display-name "${AAD_PREFIX}-tooling-shared" 2>/dev/null)"
    if [ "${client_identity_info}" == "[]" ]; then
        log "(5) Create the E2E and tooling client"
        export AZURE_CLIENT_ID="$(az ad app create \
            --display-name "${AAD_PREFIX}-tooling-shared" \
            --start-date "$(date -Iseconds)" \
            --end-date $endless_date \
            --key-type Symmetric \
            --key-usage Sign \
            --key-value "$AZURE_CLIENT_SECRET" \
            --query appId \
            -o tsv)"
        az ad sp create --id "$AZURE_CLIENT_ID" >/dev/null
    else
        export AZURE_CLIENT_ID="$(az ad app list \
             --filter "displayname eq '${AAD_PREFIX}-tooling-shared'" \
             --query '[].appId' \
             -o tsv)"
        log "(5) ⏩🔑 Skip the E2E and tooling client with application id $AZURE_CLIENT_ID"
    fi

    log "(6) Add Microsoft.Graph/Application.ReadWrite.OwnedBy permission to E2E and tooling client $AZURE_CLIENT_ID####"
    local ms_graph_sp_api_id="00000003-0000-0000-c000-000000000000"
    local permission_id="$(az ad sp show \
         --id $ms_graph_sp_api_id \
         --query "appRoles" \
         -o jsonc | jq -r '.[] | select(.value=="Application.ReadWrite.OwnedBy") | .id')"
    log "(6) Add premission $permission_id"
    local app_premission_info="$(az ad app permission list --id fb194a8e-da8a-4b15-8c1e-ef49b98987dc 2>/dev/null)" 
    if [ "${app_premission_info}" == "[]" ]; then
        az ad app permission add \
            --id "$AZURE_CLIENT_ID" \
            --api $ms_graph_sp_api_id \
            --api-permissions "$permission_id=Role"
        log "Grant premission $permission_id"
        az ad app permission grant \
            --id "$AZURE_CLIENT_ID" \
            --api $ms_graph_sp_api_id
        
        log "Admin-consent premission $permission_id"
        log -n "Are you admin? (y/N): "
		read answer
		if [[ "$answer" == "y" ]]; then
            az ad app permission admin-consent \
                --id "$AZURE_CLIENT_ID"
        else
            log "User is not an admin, only an admin can consent the premission"
		fi
    else
         log "(6) ⏩🔑 Skip adding Microsoft.Graph/Application.ReadWrite.OwnedBy permission"
    fi

    # Check if the subscription deployment RBAC_DEV_DEPLOYMENT_NAME exists
    log "(7) Set up the RP role definitions and subscription role assignments at ${location} with subscription deployment ${RBAC_DEV_DEPLOYMENT_NAME}"
    sub_dep_state="$(az deployment sub list \
        --query "[?name=='${RBAC_DEV_DEPLOYMENT_NAME}'].properties.{provision:provisioningState}" \
        -o tsv)"
    if [[ "${sub_dep_state}" == "Succeeded" ]]; then
        log "🟢📦 Deployment '${RBAC_DEV_DEPLOYMENT_NAME}' in the subscription has been provisioned successfully."
        log "(7) ⏩🔑 Skip subscription deployment creation"
    elif [[ "${sub_dep_state}" == "Failed" ]]; then
        log "⏩🔑 skip deployment '${RBAC_DEV_DEPLOYMENT_NAME}' in the subscription, since the deployment state is $sub_dep_state"
    else
        log "Create deployment '${RBAC_DEV_DEPLOYMENT_NAME}' in the subscription, since the deployment is missing"
        az deployment sub create \
            --location "${location}" \
            --name "${RBAC_DEV_DEPLOYMENT_NAME}" \
            --template-file pkg/deploy/assets/rbac-development.json \
            --parameters \
                "armServicePrincipalId=$(az ad sp list --filter "appId eq '$AZURE_ARM_CLIENT_ID'" --query '[].id' -o tsv)" \
                "fpServicePrincipalId=$(az ad sp list --filter "appId eq '$AZURE_FP_CLIENT_ID'" --query '[].id' -o tsv)" \
                "fpRoleDefinitionId"="$(uuidgen)" \
                "devServicePrincipalId=$(az ad sp list --filter "appId eq '$AZURE_CLIENT_ID'" --query '[].id' -o tsv)" \
            >/dev/null
    fi  

    log "(8) Fake up the portal client ${AAD_PREFIX}-portal-shared"
    local portal_client_info="$(az ad app list --display-name "${AAD_PREFIX}-portal-shared" 2>/dev/null)"
    if [ "${portal_client_info}" == "[]" ]; then
        log "(8) Fake up the portal client"
        export AZURE_PORTAL_CLIENT_ID="$(az ad app create \
            --display-name "${AAD_PREFIX}-portal-shared" \
            --query appId \
            -o tsv)"

        obj_id="$(az ad app show --id "$AZURE_PORTAL_CLIENT_ID" --query id -o tsv)"

        az rest --method PATCH \
            --uri "https://graph.microsoft.com/v1.0/applications/$obj_id" \
            --headers 'Content-Type=application/json' \
            --body '{"web":{"redirectUris":["https://locahlost:8444/callback"]}}'

        az ad app credential reset \
            --id "$AZURE_PORTAL_CLIENT_ID" \
            --cert "$(base64 -w0 <secrets/portal-client.crt)" >/dev/null
    else
        export AZURE_PORTAL_CLIENT_ID="$(az ad app list \
             --filter "displayname eq '${AAD_PREFIX}-portal-shared'" \
             --query '[].appId' \
             -o tsv)"
        log "(8) ⏩🔑 Skip the portal client with application id $AZURE_PORTAL_CLIENT_ID"
    fi
    log "Finish aad_applications"
}

certificates(){
    # Certificates
    log "Generate key/certificate file using an helper utility"
    # TODO- consider checking whether each key/certificate already exists under secrets/

    log "(1) VPN CA key/certificate"
    go run ./hack/genkey -ca vpn-ca
    mv vpn-ca.* secrets

    log "(2) VPN client key/certificate"
    go run ./hack/genkey -client -keyFile secrets/vpn-ca.key -certFile secrets/vpn-ca.crt vpn-client
    mv vpn-client.* secrets

    log "(3) proxy serving key/certificate"
    go run ./hack/genkey proxy
    mv proxy.* secrets

    log "(4) proxy client key/certificate"
    go run ./hack/genkey -client proxy-client
    mv proxy-client.* secrets

    log "(5) proxy ssh key/certificate"
    ssh-keygen -f secrets/proxy_id_rsa -N ''

    log "(6) RP serving key/certificate"
    go run ./hack/genkey localhost
    mv localhost.* secrets

    log "(7)) CA key/certificate"
    go run ./hack/genkey -ca dev-ca
    mv dev-ca.* secrets

    log "(8) dev client key/certificate"
    go run ./hack/genkey -client -keyFile secrets/dev-ca.key -certFile secrets/dev-ca.crt dev-client
    mv dev-client.* secrets

    log "(9) CA key/certificate"
    go run ./hack/genkey cluster-mdsd
    mv cluster-mdsd.* secrets

    log "Finish certificates"
}

env_file(){
    # Environment file
    err_str="Usage $0 <SECRET_SA_ACCOUNT_NAME> <PARENT_DOMAIN_RESOURCEGROUP> <RESOURCEGROUP_PREFIX> <PROXY_DOMAIN_NAME_LABEL>. Please try again"
    local secret_sa_account_name=${1?$err_str}
    local parent_domain_resourcegroup=${2?$err_str}
    local resourcegroup_prefix=${3?$err_str}
    local proxy_domain_name_label=${4?$err_str}
    
    local admin_object_id="$(az ad group show -g aro-engineering --query id -o tsv)"
    local azure_tenant_id=$(az account show --query tenantId -o tsv)
    local azure_subscription_id=$(az account show --query id -o tsv)

    # TODO: How to get the PULL_SECRET - We can pass it to the container...
    # TODO: How to get the USER_PULL_SECRET

    log "(1) Generate SSH key for VMSS access"
    ssh-keygen -t rsa -N "" -f secrets/full_rp_id_rsa
    log "(2) Create the secrets/env file"
    export RESOURCEGROUP="$resourcegroup_prefix-\$LOCATION"
    local oidc_prefix="${resourcegroup_prefix}${LOCATION}"
    # use a unique prefix for Azure resources when it is set, otherwise use your user's name
    cat >secrets/env <<EOF
   #### Prior to sourcing the file the following env vars must be set:   ####
   #### AZURE_PREFIX, LOCATION, ADMIN_OBJECT_ID, RESOURCEGROUP, and PARENT_DOMAIN_NAME  ####
    export AZURE_PREFIX="${AZURE_PREFIX:-$USER}"
    export ADMIN_OBJECT_ID="$admin_object_id"
    export AZURE_TENANT_ID="$azure_tenant_id"
    export AZURE_SUBSCRIPTION_ID="$azure_subscription_id"
    export AZURE_ARM_CLIENT_ID="$AZURE_ARM_CLIENT_ID"
    export AZURE_FP_CLIENT_ID="$AZURE_FP_CLIENT_ID"
    export AZURE_FP_SERVICE_PRINCIPAL_ID="$(az ad sp list --filter "appId eq '$AZURE_FP_CLIENT_ID'" --query '[].id' -o tsv)"
    export AZURE_PORTAL_CLIENT_ID="$AZURE_PORTAL_CLIENT_ID"
    export AZURE_PORTAL_ACCESS_GROUP_IDS="$admin_object_id"
    export AZURE_PORTAL_ELEVATED_GROUP_IDS="$admin_object_id"
    export AZURE_CLIENT_ID="$AZURE_CLIENT_ID"
    export AZURE_SERVICE_PRINCIPAL_ID="$(az ad sp list --filter "appId eq '$AZURE_CLIENT_ID'" --query '[].id' -o tsv)"
    export AZURE_CLIENT_SECRET="$AZURE_CLIENT_SECRET"
    export AZURE_RP_CLIENT_ID="$AZURE_RP_CLIENT_ID"
    export AZURE_RP_CLIENT_SECRET="$AZURE_RP_CLIENT_SECRET"
    export AZURE_GATEWAY_CLIENT_ID="$AZURE_GATEWAY_CLIENT_ID"
    export AZURE_GATEWAY_SERVICE_PRINCIPAL_ID="$(az ad sp list --filter "appId eq '$AZURE_GATEWAY_CLIENT_ID'" --query '[].id' -o tsv)"
    export AZURE_GATEWAY_CLIENT_SECRET="$AZURE_GATEWAY_CLIENT_SECRET"
    export RESOURCEGROUP="$RESOURCEGROUP"
    export PROXY_HOSTNAME="vm0.$proxy_domain_name_label.\$LOCATION.cloudapp.azure.com"
    export DATABASE_NAME="\$AZURE_PREFIX"
    export RP_MODE="development"
    export PULL_SECRET="PULL_SECRET"
    export USER_PULL_SECRET="USER_PULL_SECRET"
    export SECRET_SA_ACCOUNT_NAME="$secret_sa_account_name"
    export DATABASE_ACCOUNT_NAME="\$RESOURCEGROUP"
    export KEYVAULT_PREFIX="\$RESOURCEGROUP"
    export PARENT_DOMAIN_NAME="$PARENT_DOMAIN_NAME"
    export PARENT_DOMAIN_RESOURCEGROUP="$parent_domain_resourcegroup"
    export DOMAIN_NAME="\$LOCATION.\$PARENT_DOMAIN_NAME"
    export AZURE_ENVIRONMENT="AzurePublicCloud"
    export OIDC_STORAGE_ACCOUNT_NAME="${oidc_prefix}oic"
    export SSH_PRIVATE_KEY="secrets/full_rp_id_rsa"
    export SSH_PUBLIC_KEY="secrets/full_rp_id_rsa.pub"
EOF
    log "(3) Upload the secrets/env file to the storage account"
    make secrets-update
    log "Finish env_file"
}

deploy_shared_rp(){
    # Deploy Shared RP Development Environment
    log "(1) Source environment files - Not sure it is needed"
    source env.example

    log "(2) Create AzSecPack managed Identity - Manuel?"
    # This step is required for 'deploy_env_dev' -  https://msazure.visualstudio.com/ASMDocs/_wiki/wikis/ASMDocs.wiki/234249/AzSecPack-AutoConfig-UserAssigned-Managed-Identity
    # curl /subscriptions/fe16a035-e540-4ab7-80d9-373fa9a3d6ae/resourceGroups/AzSecPackAutoConfigRG/providers/Microsoft.ManagedIdentity/userAssignedIdentities/AzSecPackAutoConfigUA-westcentralus

    log "(3) Enable EncryptionAtHost for subscription"
    az feature register --namespace Microsoft.Compute --name EncryptionAtHost
    az provider register -n Microsoft.Compute

    log "(4) Create the resource group and deploy the RP resources"
    source hack/devtools/deploy-shared-env.sh
    # Create the RG
    create_infra_rg
    # # Deploy the predeployment ARM template
    # deploy_rp_dev_predeploy
    # Deploy the infrastructure resources such as Cosmos, KV, Vnet...
    deploy_rp_dev
    # Deploy RP MSI for aks/hive
    deploy_rp_managed_identity
    # Deploy the proxy and VPN
    # deploy_env_dev
    # Deploy AKS resources for Hive
    deploy_aks_dev
    # Deploy the predeployment ARM template
    deploy_rp_dev_predeploy
    # Deploy additional infrastructure required for workload identity clusters
    deploy_miwi_infra_dev

    # If you encounter a "VirtualNetworkGatewayCannotUseStandardPublicIP" error when running the deploy_env_dev command, you have to override two additional parameters. Run this command instead:
    # deploy_env_dev_override
    # If you encounter a "SkuCannotBeChangedOnUpdate" error when running the deploy_env_dev_override command, delete the -pip resource and re-run.

    log "(5) Get the AKS kubeconfig and upload it to the storage account"
    make aks.kubeconfig
    mv aks.kubeconfig secrets/
    make secrets-update
    
    # TODO: Find the aks vpn files that are missing
    log "(6) Install Hive"
    HOME=/usr KUBECONFIG=$(pwd)/secrets/aks.kubeconfig ./hack/hive/hive-dev-install.sh

    log "(7) Load the keys/certificates into the key vault"
    import_certs_secrets

    log "(?) Anymore steps?"

    log "Finish deploy_shared_rp"

}

clean_aad_applications() {
    # Cleanup 6 AAD applications and 5 SPs
    app_names=("arm" "fp" "rp" "gateway" "tooling" "portal")
    # shellcheck disable=SC2068
    for app_name in ${app_names[@]}; do
        full_app_name="$AAD_PREFIX-$app_name-shared"
        app_info="$(az ad app list --display-name "$full_app_name" 2>/dev/null)"
        if [ "${app_info}"  != "[]" ]; then
            # app_id="$(az ad app list --display-name $full_app_name --query '[].id' -o tsv)"
            app_id="$(az ad app list --display-name "$full_app_name" --query '[].appId' -o tsv)"
            # TODO do we need to delete SP of the app or just the app for cleanup?
            sp_info="$(az ad sp list --filter "appId eq $app_id" 2>/dev/null)"
            log "sp_info:$sp_info"
            if [[ $app_name != "portal" && "${sp_info}"  != "[]" ]]; then
                sp_id="$(az ad sp list --filter "appId eq $app_id" --query '[].id' -o tsv)"
                log "❌🔑 delete AAD SP id with object ID '$sp_id'"
                az ad sp delete --id "$sp_id"
            fi
            log "❌🔑 delete AAD application with name '$full_app_name' and application ID '$app_id'"
            az ad app delete --id "$app_id"
        else
            # log "⏩🔑 AAD application with name '$full_app_name' is missing so we can't delete it and there is no SP to delete"
            log "⏩🔑 AAD application with name '$full_app_name' is missing"
        fi
    done
    az deployment sub delete --name "${RBAC_DEV_DEPLOYMENT_NAME}"
    log "Finish clean_aad_applications"
}

clean_resource_groups() {
    # Cleanup prerequisites resource groups
    az group delete --resource-group "global-infra-${SHARED_RP_PREFIX}" -y || true
    az group delete --resource-group "global-infra-parent-${SHARED_RP_PREFIX}" -y || true
    az group delete --resource-group "secretstorage-${SHARED_RP_PREFIX}" -y || true

    # Cleanup deploy shared RP resource group
    az group delete --resource-group "${RESOURCEGROUP_PREFIX}-${LOCATION}" -y || true
    log "Finish clean_resource_groups"
}

certificate_rotation(){
    # Certificate Rotation
    log "(1) rotate certificates in dev and INT subscriptions after running aad_applications and certificates"
    source hack/devtools/deploy-shared-env.sh
    log "(2) dev client key/certificate"
    import_certs_secrets

    log "(3) Update the Azure VPN Gateway configuration - 'Manuel'"
    log "(4) OpenVPN configuration file - 'Manuel'"
    log "(5) Update certificates owned by FP Service Principal"
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

    log "(6)  VM needs to be deleted & redeployed - 'Manuel'?"

    log "(7) Upload the secrets to the storage account"
    # [rharosecretsdev|e2earosecrets|e2earoclassicsecrets] make secrets-update
    # SECRET_SA_ACCOUNT_NAME=[rharosecretsdev|e2earosecrets|e2earoclassicsecrets] make secrets-update

    log "Finish certificate_rotation"
}
