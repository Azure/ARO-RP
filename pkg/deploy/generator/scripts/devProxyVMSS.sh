echo "running RHUI fix"
yum update -y --disablerepo='*' --enablerepo='rhui-microsoft-azure*'

yum -y -x WALinuxAgent -x WALinuxAgent-udev update
yum -y install docker

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
systemctl start docker.service
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
After=docker.service
Requires=docker.service

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

(sleep 30; reboot) &
