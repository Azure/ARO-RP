# Hack - wait on create because the WALinuxAgent sometimes conflicts with the yum update -y below
sleep 60

echo "running RHUI fix"
yum update -y --disablerepo='*' --enablerepo='rhui-microsoft-azure*'

for attempt in {1..5}; do
  yum -y update && break
  if [[ ${attempt} -lt 5 ]]; then sleep 10; else exit 1; fi
done

DEVICE_PARTITION=$(pvs | grep '/dev/' | awk '{print $1}' | grep -oP '[a-z]{3}[0-9]$')
DEVICE=$(echo $DEVICE_PARTITION | grep -oP '^[a-z]{3}')
PARTITION=$(echo $DEVICE_PARTITION | grep -oP '[0-9]$')

# Fix the "GPT PMBR size mismatch (134217727 != 268435455)"
echo "w" | fdisk /dev/${DEVICE}

# Steps from https://access.redhat.com/solutions/5808001
# 1. Delete the LVM partition "d\n2\n"
# 2. Recreate the partition "n\n2\n"
# 3. Accept the default start and end sectors (2 x \n)
# 4. LVM2_member signature remains by default
# 5. Change type to Linux LVM "t\n2\n31\n
# 6. Write new table "w\n"

fdisk /dev/${DEVICE} <<EOF
d
${PARTITION}
n
${PARTITION}


t
${PARTITION}
31
w
EOF

partx -u /dev/${DEVICE}
pvresize /dev/${DEVICE_PARTITION}

lvextend -l +50%FREE /dev/rootvg/homelv
xfs_growfs /home

lvextend -l +50%FREE /dev/rootvg/tmplv
xfs_growfs /tmp

lvextend -l +100%FREE /dev/rootvg/varlv
xfs_growfs /var

rpm --import https://dl.fedoraproject.org/pub/epel/RPM-GPG-KEY-EPEL-8
rpm --import https://packages.microsoft.com/keys/microsoft.asc

yum -y install https://dl.fedoraproject.org/pub/epel/epel-release-latest-8.noarch.rpm

cat >/etc/yum.repos.d/azure.repo <<'EOF'
[azure-cli]
name=azure-cli
baseurl=https://packages.microsoft.com/yumrepos/azure-cli
enabled=yes
gpgcheck=yes
EOF

yum -y install azure-cli podman podman-docker jq gcc gpgme-devel libassuan-devel git make tmpwatch python3-devel htop go-toolset-1.17.12-1.module+el8.6.0+16014+a372c00b openvpn

# Suppress emulation output for podman instead of docker for az acr compatability
mkdir -p /etc/containers/
touch /etc/containers/nodocker

VSTS_AGENT_VERSION=2.206.1
mkdir /home/cloud-user/agent
pushd /home/cloud-user/agent
curl -s https://vstsagentpackage.azureedge.net/agent/${VSTS_AGENT_VERSION}/vsts-agent-linux-x64-${VSTS_AGENT_VERSION}.tar.gz | tar -xz
chown -R cloud-user:cloud-user .

./bin/installdependencies.sh
sudo -u cloud-user ./config.sh --unattended --url https://dev.azure.com/msazure --auth pat --token "$CIAZPTOKEN" --pool "$CIPOOLNAME" --agent "ARO-RHEL-$HOSTNAME" --replace
./svc.sh install cloud-user
popd

cat >/home/cloud-user/agent/.path <<'EOF'
/usr/local/bin:/usr/bin:/usr/local/sbin:/usr/sbin:/home/cloud-user/.local/bin:/home/cloud-user/bin
EOF

# Set the agent's "System capabilities" for tests (go-1.17 and GOLANG_FIPS) in the agent's .env file
# and add a HACK for XDG_RUNTIME_DIR: https://github.com/containers/podman/issues/427
cat >/home/cloud-user/agent/.env <<'EOF'
go-1.17=true
GOLANG_FIPS=1
XDG_RUNTIME_DIR=/run/user/1000
EOF

cat >/etc/cron.weekly/yumupdate <<'EOF'
#!/bin/bash

yum update -y
EOF
chmod +x /etc/cron.weekly/yumupdate

cat >/etc/cron.hourly/tmpwatch <<'EOF'
#!/bin/bash

exec /sbin/tmpwatch 24h /tmp
EOF
chmod +x /etc/cron.hourly/tmpwatch

# HACK - podman doesn't always terminate or clean up it's pause.pid file causing
# 'cannot reexec errors' so attempt to clean it up every minute to keep pipelines running
# smoothly
cat >/usr/local/bin/fix-podman-pause.sh <<'EOF'
#!/bin/bash

PAUSE_FILE='/tmp/podman-run-1000/libpod/tmp/pause.pid'

if [ -f "${PAUSE_FILE}" ]; then
	PID=$(cat ${PAUSE_FILE})
	if ! ps -p $PID > /dev/null; then
		rm $PAUSE_FILE
	fi
fi
EOF
chmod +x /usr/local/bin/fix-podman-pause.sh

# HACK - /tmp will fill up causing build failures
# delete anything not accessed within 2 days
cat >/usr/local/bin/clean-tmp.sh <<'EOF'
#!/bin/bash

find /tmp -type f \( ! -user root \) -atime +2 -delete

EOF
chmod +x /usr/local/bin/clean-tmp.sh

echo "0 0 */1 * * /usr/local/bin/clean-tmp.sh" >> cron
echo "* * * * * /usr/local/bin/fix-podman-pause.sh" >> cron

# HACK - https://github.com/containers/podman/issues/9002
echo "@reboot loginctl enable-linger cloud-user" >> cron

crontab cron
rm cron

(sleep 30; reboot) &
