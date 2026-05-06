#!/bin/bash
#Adding retry logic to yum commands in order to avoid stalling out on resource locks

# Set up logging
CSE_LOG_FILE="/var/log/azure/aro-vmss-setup.log"
mkdir -p "$(dirname "$CSE_LOG_FILE")"
exec > >(tee -a "$CSE_LOG_FILE") 2>&1

log() {
    echo "[$(date -u '+%Y-%m-%d %H:%M:%S UTC')] $*"
}

log "=== Starting DevProxy VMSS setup ==="
log "Hostname: $(hostname)"

log "installing moby-engine (docker)"
for attempt in {1..60}; do
	log "Attempting to install moby-engine moby-cli (attempt $attempt/60)"
	tdnf install -y moby-engine moby-cli && break
	if [[ ${attempt} -lt 60 ]]; then sleep 30; else log "ERROR: Failed to install moby-engine after 60 attempts"; exit 1; fi
done

log "Enabling and starting docker"
systemctl enable docker
systemctl start docker

log "Creating /root/.docker directory"
mkdir -p /root/.docker

log "Writing docker config.json"
cat >/root/.docker/config.json <<EOF
{
	"auths": {
		"${PROXYIMAGE%%/*}": {
			"auth": "$PROXYIMAGEAUTH"
		}
	}
}
EOF

log "Pulling proxy image: $PROXYIMAGE"
docker pull "$PROXYIMAGE"

log "Creating /etc/proxy directory"
mkdir -p /etc/proxy

log "Decoding and writing proxy certificates"
base64 -d <<<"$PROXYCERT" >/etc/proxy/proxy.crt
base64 -d <<<"$PROXYKEY" >/etc/proxy/proxy.key
base64 -d <<<"$PROXYCLIENTCERT" >/etc/proxy/proxy-client.crt
chown -R 1000:1000 /etc/proxy
chmod 0600 /etc/proxy/proxy.key

log "Writing /etc/sysconfig/proxy"
cat >/etc/sysconfig/proxy <<EOF
PROXY_IMAGE='$PROXYIMAGE'
EOF

log "Writing /etc/systemd/system/proxy.service"
cat >/etc/systemd/system/proxy.service <<'EOF'
[Unit]
After=network-online.target
Wants=network-online.target

[Service]
EnvironmentFile=/etc/sysconfig/proxy
ExecStartPre=-/usr/bin/docker rm -f %n
ExecStart=/usr/bin/docker run --rm --name %n -p 443:8443 -v /etc/proxy:/secrets $PROXY_IMAGE
ExecStop=/usr/bin/docker stop %n
Restart=always
RestartSec=1
StartLimitInterval=0

[Install]
WantedBy=multi-user.target
EOF

log "Enabling proxy.service"
systemctl enable proxy.service

log "Writing /etc/cron.weekly/pull-image"
cat >/etc/cron.weekly/pull-image <<'EOF'
#!/bin/bash

docker pull $PROXYIMAGE
systemctl restart proxy.service
EOF
chmod +x /etc/cron.weekly/pull-image

log "Writing /etc/cron.weekly/yumupdate"
cat >/etc/cron.weekly/yumupdate <<'EOF'
#!/bin/bash

yum update -y
EOF
chmod +x /etc/cron.weekly/yumupdate

log "Writing /etc/cron.daily/restart-proxy"
cat >/etc/cron.daily/restart-proxy <<'EOF'
#!/bin/bash

systemctl restart proxy.service
EOF
chmod +x /etc/cron.daily/restart-proxy

log "=== DevProxy VMSS setup completed successfully ==="
log "Scheduling reboot in 30 seconds"

(
	sleep 30
	reboot
) &
