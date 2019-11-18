FROM registry.access.redhat.com/ubi8/ubi-minimal
COPY rp /usr/local/bin
ENTRYPOINT ["rp"]
EXPOSE 8080/tcp
USER 1000
