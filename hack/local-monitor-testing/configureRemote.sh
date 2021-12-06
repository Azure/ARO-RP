# Setup the VM
rpm --import https://dl.fedoraproject.org/pub/epel/RPM-GPG-KEY-EPEL-7
rpm --import https://packages.microsoft.com/keys/microsoft.asc

yum -y install https://dl.fedoraproject.org/pub/epel/epel-release-latest-7.noarch.rpm

cat >/etc/yum.repos.d/azure.repo <<'EOF'
[azure-cli]
name=azure-cli
baseurl=https://packages.microsoft.com/yumrepos/azure-cli
enabled=yes
gpgcheck=yes
EOF

yum --enablerepo=rhui-rhel-7-server-rhui-optional-rpms -y install \
   azure-cli \
   docker \
   jq \
   gcc \
   rh-git29 \
   rh-python36 \
   tmpwatch \
   lttng-usr \
   gpgme-devel \
   libassuan-devel \
   socat


sed -i -e 's/^OPTIONS='\''/OPTIONS='\''-G cloud-user /' /etc/sysconfig/docker

systemctl enable docker
systemctl restart docker




