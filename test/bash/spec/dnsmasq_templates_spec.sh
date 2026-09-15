Describe 'dnsmasq shell templates'
Include ./test/bash/spec/support/helpers.sh
pre_template="${REPO_ROOT}/pkg/operator/controllers/dnsmasq/scripts/aro-dnsmasq-pre.sh.gotmpl"
restart_template="${REPO_ROOT}/pkg/operator/controllers/dnsmasq/scripts/99-dnsmasq-restart.gotmpl"

render_pre_template() {
    extract_template_body "${pre_template}"
}

render_restart_template() {
    extract_template_body "${restart_template}"
}

run_restart_template_for_event() {
    ensure_test_workspace
    local interface="$1"
    local action="$2"
    local rendered_script="${TEST_ROOT}/99-dnsmasq-restart.sh"

    extract_template_body "${restart_template}" > "${rendered_script}"
    chmod +x "${rendered_script}"

    write_file_contents /etc/resolv.conf.dnsmasq 'nameserver 10.0.0.2'
    append_mock_command logger 'printf "logger %s\n" "$*" >> "${CALL_LOG}"'
    append_mock_command systemctl 'printf "systemctl %s\n" "$*" >> "${CALL_LOG}"'

    CONNECTIVITY_STATE="full" bash "${rendered_script}" "${interface}" "${action}" > /dev/null

    cat "${CALL_LOG}"
}

run_restart_template_for_irrelevant_event() {
    run_restart_template_for_event "lo" "up"
}

render_template_with_internal_blank_lines() {
    ensure_test_workspace

    local template="${TEST_ROOT}/blank-lines.gotmpl"
    cat > "${template}" << 'EOF'
{{ define "blank-lines" }}

#!/bin/bash

echo first

echo second

{{ end }}

EOF

    extract_template_body "${template}"
}

It 'renders the pre hook with dnsmasq and NetworkManager updates'
When call render_pre_template
The status should be success
The output should start with '#!/bin/bash'
The output should include '/etc/resolv.conf.dnsmasq'
The output should include '/etc/NetworkManager/conf.d/aro-dns.conf'
The output should include '/usr/bin/nmcli general reload conf'
The output should include '/usr/bin/nmcli general reload dns-rc'
End

It 'renders the restart hook with dnsmasq restart logic for interface changes'
When call render_restart_template
The status should be success
The output should start with '#!/bin/bash'
The output should include 'logger -i "$0" -t '"'"'99-DNSMASQ-RESTART SCRIPT'"'"''
The output should include 'systemctl try-restart dnsmasq --wait'
The output should include '[[ $interface == eth* && $action == "up" ]]'
The output should include '[[ $interface == enP* && $action == "down" ]]'
End

It 'restarts dnsmasq for matching interface transitions'
When call run_restart_template_for_event "eth0" "up"
The status should be success
The output should include 'systemctl try-restart dnsmasq --wait'
The output should include 'dnsmasq successfully restarted'
End

It 'skips dnsmasq restart for unrelated interface transitions'
When call run_restart_template_for_irrelevant_event
The status should be success
The output should eq ''
End

It 'preserves internal blank lines while stripping template delimiters'
expected_blank_lines="$(
    cat << 'EOF'
echo first

echo second
EOF
)"

When call render_template_with_internal_blank_lines
The status should be success
The output should start with '#!/bin/bash'
The output should include "${expected_blank_lines}"
The output should end with 'echo second'
End
End
