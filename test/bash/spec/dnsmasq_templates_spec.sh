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
    The output should start with '#!/bin/sh'
    The output should include 'logger -i "$0" -t '"'"'99-DNSMASQ-RESTART SCRIPT'"'"''
    The output should include 'systemctl try-restart dnsmasq --wait'
    The output should include '[[ $interface == eth* && $action == "up" ]]'
    The output should include '[[ $interface == enP* && $action == "down" ]]'
  End
End
