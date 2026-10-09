REPO_ROOT="${REPO_ROOT:-$(cd "${SHELLSPEC_HELPERDIR}/../../.." && pwd)}"
export REPO_ROOT

ensure_test_workspace() {
    if [ -n "${TEST_ROOT:-}" ] && [ -d "${TEST_ROOT}" ]; then
        return 0
    fi

    TEST_ROOT="$(mktemp -d "${TMPDIR:-/tmp}/aro-bash-tests.XXXXXX")"
    SCRIPT_FIXTURE_DIR="${TEST_ROOT}/scripts"
    MOCK_BIN="${TEST_ROOT}/bin"
    CALL_LOG="${TEST_ROOT}/calls.log"
    ORIGINAL_PATH="${ORIGINAL_PATH:-$PATH}"

    mkdir -p "${SCRIPT_FIXTURE_DIR}" "${MOCK_BIN}"
    mkdir -p /etc/ssh /etc/sysconfig /etc/systemd/system /etc/cron.weekly /etc/cron.daily
    mkdir -p /etc/NetworkManager/conf.d /etc/kubernetes/manifests /var/lib /host/usr/bin /host/usr/lib64 /usr/lib64
    : > "${CALL_LOG}"

    PATH="${MOCK_BIN}:${ORIGINAL_PATH}"
    export TEST_ROOT SCRIPT_FIXTURE_DIR MOCK_BIN CALL_LOG ORIGINAL_PATH PATH
}

reset_absolute_test_state() {
    rm -rf /etc/proxy
    rm -rf /root/.docker
    rm -rf /var/lib/etcd
    rm -rf /var/lib/etcd-backup-*
    rm -f /etc/sysconfig/aro-gateway
    rm -f /etc/sysconfig/aro-rp
    rm -f /etc/sysconfig/proxy
    rm -f /etc/systemd/system/aro-gateway.service
    rm -f /etc/systemd/system/aro-rp.service
    rm -f /etc/systemd/system/proxy.service
    rm -f /etc/cron.weekly/pull-image
    rm -f /etc/cron.weekly/yumupdate
    rm -f /etc/cron.daily/restart-proxy
    rm -f /etc/kubernetes/manifests/etcd-pod.yaml
    rm -f /etc/resolv.conf.dnsmasq
    rm -f /etc/NetworkManager/conf.d/aro-dns.conf
    rm -f /host/usr/bin/jq
    rm -f /host/usr/bin/oc

    local lib
    for lib in /usr/lib64/libjq.so.1 /usr/lib64/libonig.so.5; do
        if [ -L "${lib}" ] && [ "$(readlink "${lib}")" = "/host${lib}" ]; then
            rm -f -- "${lib}"
        fi
    done

copy_fixture() {
    local source_path="$1"
    local destination_name="$2"

    ensure_test_workspace
    cp "${source_path}" "${SCRIPT_FIXTURE_DIR}/${destination_name}"
    chmod +x "${SCRIPT_FIXTURE_DIR}/${destination_name}"
    echo "${SCRIPT_FIXTURE_DIR}/${destination_name}"
}

write_file_contents() {
    local target_path="$1"
    shift

    ensure_test_workspace
    mkdir -p "$(dirname "${target_path}")"
    cat > "${target_path}" << EOF
$*
EOF
}

append_mock_command() {
    local name="$1"
    shift

    ensure_test_workspace
    cat > "${MOCK_BIN}/${name}" << EOF
#!/bin/bash
set -eu
$*
EOF
    chmod +x "${MOCK_BIN}/${name}"
}

write_vmss_util_stub() {
    ensure_test_workspace
    cat > "${SCRIPT_FIXTURE_DIR}/util.sh" << 'EOF'
#!/bin/bash
set -eu

declare -r role_gateway="gateway"
declare -r role_rp="rp"
declare -r role_devproxy="devproxy"

_log() {
  printf '%s\n' "$*" >> "${CALL_LOG}"
}

_log_block() {
  local label="$1"
  local contents="$2"
  {
    printf '%s<<EOF\n' "${label}"
    printf '%s\n' "${contents}"
    printf 'EOF\n'
  } >> "${CALL_LOG}"
}

create_required_dirs() {
  _log "create_required_dirs"
}

configure_sshd() {
  _log "configure_sshd"
}

configure_rpm_repos() {
  local -n wait_ref="$1"
  _log "configure_rpm_repos wait=${wait_ref} retries=${2:-}"
}

dnf_update_pkgs() {
  local -n excludes_ref="$1"
  local -n wait_ref="$2"
  _log "dnf_update_pkgs excludes=${excludes_ref[*]} wait=${wait_ref} retries=${3:-}"
}

dnf_install_pkgs() {
  local -n packages_ref="$1"
  local -n wait_ref="$2"
  _log "dnf_install_pkgs packages=${packages_ref[*]} wait=${wait_ref} retries=${3:-}"
}

fips_configure() {
  _log "fips_configure"
}

host_mem_mib() {
  printf '%s\n' "16384"
}

configure_logrotate() {
  _log "configure_logrotate"
}

pull_container_images() {
  local -n images_ref="$1"
  _log "pull_container_images keys=${!images_ref[*]}"
}

firewalld_configure() {
  local -n ports_ref="$1"
  _log "firewalld_configure ports=${ports_ref[*]}"
}

configure_vmss_aro_services() {
  local -n role_ref="$1"
  local -n images_ref="$2"
  local -n configs_ref="$3"
  _log "configure_vmss_aro_services role=${role_ref} images=${!images_ref[*]} configs=${!configs_ref[*]}"

  local config_name
  for config_name in "${!configs_ref[@]}"; do
    case "${config_name}" in
      rp_config|gateway_config|gateway_otel_collector|azuremonitor_tenant)
        local -n config_ref="${configs_ref[$config_name]}"
        _log_block "${config_name}" "${config_ref}"
        ;;
    esac
  done
}

enable_services() {
  local -n services_ref="$1"
  _log "enable_services services=${services_ref[*]}"
}

reboot_vm() {
  _log "reboot_vm"
}
EOF
    chmod +x "${SCRIPT_FIXTURE_DIR}/util.sh"
}

set_vmss_environment() {
    export AZURECLOUDNAME="AzurePublicCloud"
    export RPIMAGE="example.azurecr.io/aro:latest"
    export MDMIMAGE="example.azurecr.io/mdm:latest"
    export MISEIMAGE="example.azurecr.io/mise:latest"
    export OTELIMAGE="example.azurecr.io/otel:latest"
    export FLUENTBITIMAGE="example.azurecr.io/fluentbit:latest"
    export GATEWAYOTELCOLLECTORIMAGE="example.azurecr.io/gateway-otel-collector:latest"
    export ACRRESOURCEID="/subscriptions/test/resourceGroups/example/providers/Microsoft.ContainerRegistry/registries/example"
    export DATABASEACCOUNTNAME="db-account"
    export RPMDMACCOUNT="rp-mdm-account"
    export GATEWAYDOMAINS="gateway.example.com"
    export GATEWAYFEATURES="alpha,beta"
    export GATEWAYMDSDCONFIGVERSION="1"
    export GATEWAYOTELKUSTOINGESTIONENDPOINT="https://example.kusto.windows.net"
    export RPLOGLEVEL="debug"
    export GATEWAYLOGLEVEL="info"
    export RPMDSDCONFIGVERSION="2"
    export ADMINAPICLIENTCERTCOMMONNAME="admin"
    export ARMAPICLIENTCERTCOMMONNAME="arm"
    export ARMCLIENTID="arm-client"
    export FPCLIENTID="fp-client"
    export FPSERVICEPRINCIPALID="fp-sp"
    export GATEWAYCLIENTID="gateway-client"
    export CLUSTERMDMACCOUNT="cluster-mdm"
    export CLUSTERMDSDACCOUNT="cluster-mdsd"
    export CLUSTERMDSDCONFIGVERSION="3"
    export OTELCLUSTERMDSDCONFIGVERSION="4"
    export CLUSTERMDSDNAMESPACE="cluster-ns"
    export LOCATION="westeurope"
    export CLUSTERPARENTDOMAINNAME="aroapp.io"
    export GATEWAYRESOURCEGROUPNAME="gateway-rg"
    export GATEWAYUSERASSIGNEDIDENTITYRESOURCEID="/subscriptions/test/resourceGroups/example/providers/Microsoft.ManagedIdentity/userAssignedIdentities/gateway"
    export KEYVAULTPREFIX="kv-prefix"
    export MDSDENVIRONMENT="prod"
    export RPFEATURES="feature-a"
    export CLUSTERSINSTALLVIAHIVE="false"
    export CLUSTERDEFAULTINSTALLERPULLSPEC="installer:latest"
    export CLUSTERSADOPTBYHIVE="false"
    export RPPARENTDOMAINNAME="rp-parent.example.com"
    export OIDCSTORAGEACCOUNTNAME="oidcstorage"
    export OTELAUDITQUEUESIZE="10"
    export MSIRPENDPOINT="https://msi.example.com"
    export ENVIRONMENT="test"
    export PROXYIMAGE="example.azurecr.io/proxy:latest"
    export PROXYIMAGEAUTH="cHJveHk6YXV0aA=="
    export PROXYCERT="Y2VydA=="
    export PROXYKEY="a2V5"
    export PROXYCLIENTCERT="Y2xpZW50"
}

extract_template_body() {
    local line
    local start end i
    local -a lines=()

    while IFS= read -r line || [ -n "${line}" ]; do
        if [[ "${line}" =~ ^[[:space:]]*\{\{[[:space:]]*define[[:space:]].*\}\}[[:space:]]*$ ]]; then
            continue
        fi

        if [[ "${line}" =~ ^[[:space:]]*\{\{[[:space:]]*end[[:space:]]*\}\}[[:space:]]*$ ]]; then
            continue
        fi

        lines+=("${line}")
    done < "$1"

    start=0
    while [ "${start}" -lt "${#lines[@]}" ] && [[ "${lines[$start]}" =~ ^[[:space:]]*$ ]]; do
        start=$((start + 1))
    done

    end=$((${#lines[@]} - 1))
    while [ "${end}" -ge "${start}" ] && [[ "${lines[$end]}" =~ ^[[:space:]]*$ ]]; do
        end=$((end - 1))
    done

    for ((i = start; i <= end; i++)); do
        printf '%s\n' "${lines[$i]}"
    done
}
