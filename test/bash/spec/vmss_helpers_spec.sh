Describe 'VMSS helper scripts'
  Include ./test/bash/spec/support/helpers.sh
  Include ./pkg/deploy/generator/scripts/util-common.sh
  Include ./pkg/deploy/generator/scripts/util-packages.sh
  Include ./pkg/deploy/generator/scripts/util-system.sh
  Include ./pkg/deploy/generator/scripts/util-services.sh

  BeforeEach 'set_vmss_environment'

  Describe 'util-common.sh'
    write_file_example() {
      ensure_test_workspace
      local target_file="${TEST_ROOT}/write-file.txt"
      local initial_text="first"
      local appended_text="second"

      write_file target_file initial_text true
      write_file target_file appended_text false

      cat "${target_file}"
    }

    retry_until_success_example() {
      ensure_test_workspace
      local counter_file="${TEST_ROOT}/retry-count"
      printf '0' > "${counter_file}"
      transient_command() {
        local current_attempt
        current_attempt="$(cat "${counter_file}")"
        current_attempt=$((current_attempt + 1))
        printf '%s' "${current_attempt}" > "${counter_file}"
        [ "${current_attempt}" -ge 3 ]
      }

      local wait_seconds=0
      local -a cmd=(transient_command)
      retry cmd wait_seconds 5
      attempts="$(cat "${counter_file}")"
    }

    verify_invalid_role_example() {
      local role="invalid"
      verify_role role
    }

    It 'writes and appends file contents'
      When call write_file_example
      The status should be success
      The output should include "first"
      The output should include "second"
    End

    It 'retries commands until they succeed'
      When call retry_until_success_example
      The status should be success
      The output should include "attempt #3 - "
      The variable attempts should eq 3
    End

    It 'fails on unknown VMSS roles'
      When run verify_invalid_role_example
      The status should be failure
      The output should include 'failed to verify role'
      The output should include 'invalid'
    End
  End

  Describe 'util-packages.sh'
    configure_repos_example() {
      ensure_test_workspace
      curl() { :; }
      retry() {
        local -n wait_ref="$2"
        printf 'cmd=%s\nwait=%s\nretries=%s\n' "${cmd[*]}" "${wait_ref}" "${3:-}"
      }

      local wait_time=7
      configure_rpm_repos wait_time 9
    }

    install_packages_example() {
      ensure_test_workspace
      retry() {
        local -n wait_ref="$2"
        printf 'cmd=%s\nwait=%s\nretries=%s\n' "${cmd[*]}" "${wait_ref}" "${3:-}"
      }

      local wait_time=5
      local -a packages=(azure-cli podman)
      dnf_install_pkgs packages wait_time 11
    }

    It 'configures the mariner extended repo through retry'
      When call configure_repos_example
      The status should be success
      The output should include 'dnf update -y --enablerepo=cbl-mariner2.0prodextendedx86_64'
      The output should include 'wait=7'
      The output should include 'retries=9'
    End

    It 'builds dnf install commands from package arrays'
      When call install_packages_example
      The status should be success
      The output should include 'dnf -y install azure-cli'
      The output should include 'podman'
      The output should include 'wait=5'
      The output should include 'retries=11'
    End
  End

  Describe 'util-system.sh'
    configure_sshd_example() {
      ensure_test_workspace
      write_file_contents /etc/ssh/sshd_config 'PasswordAuthentication no'
      systemctl() { printf '%s\n' "$*" >> "${CALL_LOG}"; }

      configure_sshd

      cat /etc/ssh/sshd_config
    }

    configure_devproxy_certs_example() {
      ensure_test_workspace
      configure_certs_devproxy
      printf '%s\n' "$(cat /etc/proxy/proxy.crt)"
      printf '%s\n' "$(cat /etc/proxy/proxy.key)"
      printf '%s\n' "$(cat /etc/proxy/proxy-client.crt)"
      stat -c '%a' /etc/proxy/proxy.key
    }

    It 'updates sshd config and reloads sshd'
      When call configure_sshd_example
      The status should be success
      The output should include 'PasswordAuthentication yes'
      The contents of file "${CALL_LOG}" should include 'reload sshd.service'
    End

    It 'writes devproxy certificates with locked-down key permissions'
      When call configure_devproxy_certs_example
      The status should be success
      The output should include 'cert'
      The output should include 'key'
      The output should include 'client'
      The output should end with '600'
      The path /etc/proxy/proxy.key should be file
    End
  End

  Describe 'util-services.sh'
    enable_services_example() {
      ensure_test_workspace
      systemctl() { printf '%s\n' "$*" >> "${CALL_LOG}"; }

      local -a services=(aro-rp fluentbit)
      enable_services services

      cat "${CALL_LOG}"
    }

    configure_rp_service_example() {
      ensure_test_workspace
      local service_image="${RPIMAGE}"
      local service_role="rp"
      local service_conf="CUSTOM='value'"
      local service_ip="10.88.0.2"

      configure_service_aro_rp service_image service_role service_conf service_ip

      cat /etc/sysconfig/aro-rp
      printf '\n--service--\n'
      cat /etc/systemd/system/aro-rp.service
    }

    configure_gateway_service_example() {
      ensure_test_workspace
      local service_image="${RPIMAGE}"
      local service_role="gateway"
      local service_conf="GATEWAY='enabled'"
      local service_ip="10.88.0.3"

      configure_service_aro_gateway service_image service_role service_conf service_ip

      cat /etc/sysconfig/aro-gateway
      printf '\n--service--\n'
      cat /etc/systemd/system/aro-gateway.service
    }

    It 'reloads systemd and enables requested services'
      When call enable_services_example
      The status should be success
      The output should include 'daemon-reload'
      The output should include 'enable --now aro-rp'
      The output should include 'enable --now fluentbit'
    End

    It 'writes the aro-rp systemd unit and env file'
      When call configure_rp_service_example
      The status should be success
      The output should include "IPADDRESS='10.88.0.2'"
      The output should include "ARO_LOG_LEVEL='debug'"
      The output should include 'ExecStart=/usr/bin/podman run'
      The output should include '-p 443:8443'
    End

    It 'writes the aro-gateway systemd unit and env file'
      When call configure_gateway_service_example
      The status should be success
      The output should include "IPADDRESS='10.88.0.3'"
      The output should include "ROLE='gateway'"
      The output should include 'ExecStart=/usr/bin/podman run'
      The output should include '-p 80:8080'
      The output should include '-p 443:8443'
    End
  End
End
