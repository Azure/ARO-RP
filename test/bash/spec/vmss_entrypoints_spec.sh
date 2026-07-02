Describe 'VMSS entrypoint scripts'
  Include ./test/bash/spec/support/helpers.sh
  BeforeEach 'set_vmss_environment'

  run_vmss_script() {
    ensure_test_workspace
    local source_path="$1"
    local destination_name="$2"

    copy_fixture "${source_path}" "${destination_name}" >/dev/null
    write_vmss_util_stub

    (
      cd "${SCRIPT_FIXTURE_DIR}"
      bash "./${destination_name}" >/dev/null
    )

    cat "${CALL_LOG}"
  }

  run_devproxy_script() {
    ensure_test_workspace
    copy_fixture "${REPO_ROOT}/pkg/deploy/generator/scripts/devProxyVMSS.sh" "devProxyVMSS.sh" >/dev/null

    append_mock_command tdnf 'printf "tdnf %s\n" "$*" >> "${CALL_LOG}"'
    append_mock_command systemctl 'printf "systemctl %s\n" "$*" >> "${CALL_LOG}"'
    append_mock_command docker 'printf "docker %s\n" "$*" >> "${CALL_LOG}"'
    append_mock_command sleep 'printf "sleep %s\n" "$*" >> "${CALL_LOG}"'
    append_mock_command reboot 'printf "reboot\n" >> "${CALL_LOG}"'

    (
      cd "${SCRIPT_FIXTURE_DIR}"
      bash ./devProxyVMSS.sh >/dev/null
    )

    cat "${CALL_LOG}"
    printf '\n--proxy-env--\n'
    cat /etc/sysconfig/proxy
    printf '\n--proxy-service--\n'
    cat /etc/systemd/system/proxy.service
    printf '\n--pull-image--\n'
    cat /etc/cron.weekly/pull-image
    printf '\n--restart-proxy--\n'
    cat /etc/cron.daily/restart-proxy
  }

  It 'runs rpVMSS through the shared utility layer'
    When call run_vmss_script "${REPO_ROOT}/pkg/deploy/generator/scripts/rpVMSS.sh" "rpVMSS.sh"
    The status should be success
    The output should include 'create_required_dirs'
    The output should include 'configure_sshd'
    The output should include 'dnf_install_pkgs packages=azure-cli azure-mdsd podman podman-docker openssl-perl python3 firewalld grubby dracut-fips'
    The output should include 'configure_vmss_aro_services role=rp'
    The output should include 'enable_services services=aro-mise aro-monitor aro-otel-collector aro-portal aro-mimo-actuator aro-mimo-scheduler aro-rp'
    The output should include 'reboot_vm'
  End

  It 'runs gatewayVMSS through the shared utility layer'
    When call run_vmss_script "${REPO_ROOT}/pkg/deploy/generator/scripts/gatewayVMSS.sh" "gatewayVMSS.sh"
    The status should be success
    The output should include 'create_required_dirs'
    The output should include 'configure_sshd'
    The output should include 'dnf_install_pkgs packages=azure-cli azure-mdsd podman podman-docker openssl-perl python3 firewalld grubby dracut-fips'
    The output should include 'configure_vmss_aro_services role=gateway'
    The output should include 'enable_services services=aro-gateway azsecd mdsd mdm chronyd fluentbit'
    The output should include 'reboot_vm'
  End

  It 'writes the devproxy service and maintenance scripts'
    When call run_devproxy_script
    The status should be success
    The output should include 'tdnf install -y moby-engine moby-cli'
    The output should include 'systemctl enable docker'
    The output should include 'docker pull example.azurecr.io/proxy:latest'
    The output should include "PROXY_IMAGE='example.azurecr.io/proxy:latest'"
    The output should include 'ExecStart=/usr/bin/docker run --rm --name %n -p 443:8443 -v /etc/proxy:/secrets $PROXY_IMAGE'
    The output should include 'docker pull $PROXYIMAGE'
    The output should include 'systemctl restart proxy.service'
  End
End
