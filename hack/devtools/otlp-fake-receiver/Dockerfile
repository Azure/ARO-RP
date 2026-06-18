FROM golang:1.25.3 AS build

WORKDIR /src
COPY hack/devtools/otlp-fake-receiver/main.go /src/main.go
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/otlp-fake-receiver /src/main.go

FROM registry.access.redhat.com/ubi9/ubi-micro:9.6
COPY --from=build /out/otlp-fake-receiver /usr/local/bin/otlp-fake-receiver
ENTRYPOINT ["/usr/local/bin/otlp-fake-receiver"]
