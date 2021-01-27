FROM registry.access.redhat.com/devtools/go-toolset-rhel7:1.14
ENV GOOS=linux \
    GOPATH=/go/
USER root
RUN yum update -y && \
    yum --enablerepo=rhel-7-server-optional-rpms install gpgme-devel libassuan-devel openssl -y && \
    yum clean all
