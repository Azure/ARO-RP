package dnsmasq

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"bytes"
	"text/template"
)

const restartTemplate = `{{ define "99-dnsmasq-restart" }}
#!/bin/sh
# This is a NetworkManager dispatcher script to restart dnsmasq
# in the event of a network interface change (e. g. host servicing event https://learn.microsoft.com/en-us/azure/developer/intro/hosting-apps-on-azure)
# this will restart dnsmasq, reapplying our /etc/resolv.conf file and overwriting any modifications made by NetworkManager

interface=$1
action=$2

log() {
    logger -i "$0" -t '99-DNSMASQ-RESTART SCRIPT' "$@"
}

# log dns configuration information relevant to SRE while troubleshooting
# The line break used here is important for formatting
check_dns_files() {
    log "/etc/resolv.conf contents

    $(cat /etc/resolv.conf)"

    log "$(echo -n \"/etc/resolv.conf file metadata: \") $(ls -lZ /etc/resolv.conf)"

    log "/etc/resolv.conf.dnsmasq contents

    $(cat /etc/resolv.conf.dnsmasq)"

    log "$(echo -n "/etc/resolv.conf.dnsmasq file metadata: ") $(ls -lZ /etc/resolv.conf.dnsmasq)"
}

if [[ $interface == eth* && $action == "up" ]] || [[ $interface == eth* && $action == "down" ]] || [[ $interface == enP* && $action == "up" ]] || [[ $interface == enP* && $action == "down" ]]; then
    log "$action happened on $interface, connection state is now $CONNECTIVITY_STATE"
    log "Pre dnsmasq restart file information"
    check_dns_files
    log "restarting dnsmasq now"
    if systemctl try-restart dnsmasq --wait; then
        log "dnsmasq successfully restarted"
        log "Post dnsmasq restart file information"
        check_dns_files
    else
        log "failed to restart dnsmasq"
    fi
fi

exit 0
{{ end }}
`

func restartDnsmasqTemplate() *template.Template {
	return template.Must(template.New("").Parse(restartTemplate))
}

func nmDispatcherRestartDnsmasq() ([]byte, error) {
	buf := &bytes.Buffer{}

	err := restartDnsmasqTemplate().ExecuteTemplate(buf, "99-dnsmasq-restart", nil)
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
