#Adding retry logic to yum commands in order to avoid stalling out on resource locks
echo "running RHUI fix"
for attempt in {1..60}; do
  yum update -y --disablerepo='*' --enablerepo='rhui-microsoft-azure*' && break
  if [[ ${attempt} -lt 60 ]]; then sleep 30; else exit 1; fi
done

echo "running yum update"
for attempt in {1..60}; do
  yum -y -x WALinuxAgent -x WALinuxAgent-udev update --allowerasing && break
  if [[ ${attempt} -lt 60 ]]; then sleep 30; else exit 1; fi
done

echo "installing podman-docker"
for attempt in {1..60}; do
  yum -y install podman-docker && break
  if [[ ${attempt} -lt 60 ]]; then sleep 30; else exit 1; fi
done

firewall-cmd --add-port=443/tcp --permanent

mkdir /root/.docker
cat >/root/.docker/config.json <<EOF
{
	"auths": {
		"${PROXYIMAGE%%/*}": {
			"auth": "$PROXYIMAGEAUTH"
		}
	}
}
EOF

mkdir -p /etc/containers/
touch /etc/containers/nodocker

docker pull "$PROXYIMAGE"

mkdir /etc/proxy
base64 -d <<<"$PROXYCERT" >/etc/proxy/proxy.crt
base64 -d <<<"$PROXYKEY" >/etc/proxy/proxy.key
base64 -d <<<"$PROXYCLIENTCERT" >/etc/proxy/proxy-client.crt
chown -R 1000:1000 /etc/proxy
chmod 0600 /etc/proxy/proxy.key

cat >/etc/sysconfig/proxy <<EOF
PROXY_IMAGE='$PROXYIMAGE'
EOF

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

systemctl enable proxy.service

cat >/etc/cron.weekly/pull-image <<'EOF'
#!/bin/bash

docker pull $PROXYIMAGE
systemctl restart proxy.service
EOF
chmod +x /etc/cron.weekly/pull-image

cat >/etc/cron.weekly/yumupdate <<'EOF'
#!/bin/bash

yum update -y
EOF
chmod +x /etc/cron.weekly/yumupdate

cat >/etc/cron.daily/restart-proxy <<'EOF'
#!/bin/bash

systemctl restart proxy.service
EOF
chmod +x /etc/cron.daily/restart-proxy

(
	sleep 30
	reboot
) &
