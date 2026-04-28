set -eu

spec_helper_precheck() {
  minimum_version "0.28.0"
  if [ "$SHELL_TYPE" != "bash" ]; then
    abort "Only bash is supported."
  fi
}

spec_helper_loaded() {
  REPO_ROOT="$(cd "${SHELLSPEC_HELPERDIR}/../../.." && pwd)"
  export REPO_ROOT
}

spec_helper_configure() {
  before_each 'setup_test_workspace'
  after_each 'cleanup_test_workspace'
}

setup_test_workspace() {
  :
}

cleanup_test_workspace() {
  reset_absolute_test_state
  rm -f /etc/ssh/sshd_config
  rm -rf "${TMPDIR:-/tmp}"/aro-bash-tests.*
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
  rm -f /usr/lib64/libjq.so.1
  rm -f /usr/lib64/libonig.so.5
}

copy_fixture() {
  local source_path="$1"
  local destination_name="$2"

  cp "${source_path}" "${SCRIPT_FIXTURE_DIR}/${destination_name}"
  chmod +x "${SCRIPT_FIXTURE_DIR}/${destination_name}"
  echo "${SCRIPT_FIXTURE_DIR}/${destination_name}"
}

write_file_contents() {
  local target_path="$1"
  shift

  mkdir -p "$(dirname "${target_path}")"
  cat > "${target_path}" <<EOF
$*
EOF
}

append_mock_command() {
  local name="$1"
  shift

  cat > "${MOCK_BIN}/${name}" <<EOF
#!/bin/bash
set -eu
$*
EOF
  chmod +x "${MOCK_BIN}/${name}"
}

write_vmss_util_stub() {
  cat > "${SCRIPT_FIXTURE_DIR}/util.sh" <<'EOF'
#!/bin/bash
set -eu

declare -r role_gateway="gateway"
declare -r role_rp="rp"
declare -r role_devproxy="devproxy"

_log() {
  printf '%s\n' "$*" >> "${CALL_LOG}"
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
  export ACRRESOURCEID="/subscriptions/test/resourceGroups/example/providers/Microsoft.ContainerRegistry/registries/example"
  export DATABASEACCOUNTNAME="db-account"
  export RPMDMACCOUNT="rp-mdm-account"
  export GATEWAYDOMAINS="gateway.example.com"
  export GATEWAYFEATURES="alpha,beta"
  export GATEWAYMDSDCONFIGVERSION="1"
  export RPLOGLEVEL="debug"
  export GATEWAYLOGLEVEL="info"
  export RPMDSDCONFIGVERSION="2"
  export ADMINAPICLIENTCERTCOMMONNAME="admin"
  export ARMAPICLIENTCERTCOMMONNAME="arm"
  export ARMCLIENTID="arm-client"
  export FPCLIENTID="fp-client"
  export FPSERVICEPRINCIPALID="fp-sp"
  export CLUSTERMDMACCOUNT="cluster-mdm"
  export CLUSTERMDSDACCOUNT="cluster-mdsd"
  export CLUSTERMDSDCONFIGVERSION="3"
  export CLUSTERMDSDNAMESPACE="cluster-ns"
  export LOCATION="westeurope"
  export CLUSTERPARENTDOMAINNAME="aroapp.io"
  export GATEWAYRESOURCEGROUPNAME="gateway-rg"
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
  sed '1d;$d' "$1"
}
