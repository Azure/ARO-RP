FROM golang:1.25.3 AS build

WORKDIR /src
COPY go.mod go.sum /src/
RUN go mod download
COPY hack/devtools/otlp-fake-kusto-bridge/main.go /src/main.go
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/otlp-fake-kusto-bridge /src/main.go

FROM registry.access.redhat.com/ubi9/ubi-micro:9.6
COPY --from=build /out/otlp-fake-kusto-bridge /usr/local/bin/otlp-fake-kusto-bridge
ENTRYPOINT ["/usr/local/bin/otlp-fake-kusto-bridge"]
