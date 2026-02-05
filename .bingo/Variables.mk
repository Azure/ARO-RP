# Auto generated binary variables helper managed by https://github.com/bwplotka/bingo v0.9. DO NOT EDIT.
# All tools are designed to be build inside $GOBIN.
BINGO_DIR := $(dir $(lastword $(MAKEFILE_LIST)))
GOPATH ?= $(shell go env GOPATH)
GOBIN  ?= $(firstword $(subst :, ,${GOPATH}))/bin
GO     ?= $(shell which go)

# Below generated variables ensure that every time a tool under each variable is invoked, the correct version
# will be used; reinstalling only if needed.
# For example for bingo variable:
#
# In your main Makefile (for non array binaries):
#
#include .bingo/Variables.mk # Assuming -dir was set to .bingo .
#
#command: $(BINGO)
#	@echo "Running bingo"
#	@$(BINGO) <flags/args..>
#
BINGO := $(GOBIN)/bingo-v0.9.0
$(BINGO): $(BINGO_DIR)/bingo.mod
	@# Install binary/ries using Go 1.14+ build command. This is using bwplotka/bingo-controlled, separate go module with pinned dependencies.
	@echo "(re)installing $(GOBIN)/bingo-v0.9.0"
	@cd $(BINGO_DIR) && GOWORK=off $(GO) build -mod=mod -modfile=bingo.mod -o=$(GOBIN)/bingo-v0.9.0 "github.com/bwplotka/bingo"

CLIENT_GEN := $(GOBIN)/client-gen-v0.25.16
$(CLIENT_GEN): $(BINGO_DIR)/client-gen.mod
	@# Install binary/ries using Go 1.14+ build command. This is using bwplotka/bingo-controlled, separate go module with pinned dependencies.
	@echo "(re)installing $(GOBIN)/client-gen-v0.25.16"
	@cd $(BINGO_DIR) && GOWORK=off $(GO) build -mod=mod -modfile=client-gen.mod -o=$(GOBIN)/client-gen-v0.25.16 "k8s.io/code-generator/cmd/client-gen"

CONTROLLER_GEN := $(GOBIN)/controller-gen-v0.16.5
$(CONTROLLER_GEN): $(BINGO_DIR)/controller-gen.mod
	@# Install binary/ries using Go 1.14+ build command. This is using bwplotka/bingo-controlled, separate go module with pinned dependencies.
	@echo "(re)installing $(GOBIN)/controller-gen-v0.16.5"
	@cd $(BINGO_DIR) && GOWORK=off $(GO) build -mod=mod -modfile=controller-gen.mod -o=$(GOBIN)/controller-gen-v0.16.5 "sigs.k8s.io/controller-tools/cmd/controller-gen"

ENUMER := $(GOBIN)/enumer-v1.5.11
$(ENUMER): $(BINGO_DIR)/enumer.mod
	@# Install binary/ries using Go 1.14+ build command. This is using bwplotka/bingo-controlled, separate go module with pinned dependencies.
	@echo "(re)installing $(GOBIN)/enumer-v1.5.11"
	@cd $(BINGO_DIR) && GOWORK=off $(GO) build -mod=mod -modfile=enumer.mod -o=$(GOBIN)/enumer-v1.5.11 "github.com/dmarkham/enumer"

FIPS_DETECT := $(GOBIN)/fips-detect-v0.0.0-20230309083406-7157dae5bafd
$(FIPS_DETECT): $(BINGO_DIR)/fips-detect.mod
	@# Install binary/ries using Go 1.14+ build command. This is using bwplotka/bingo-controlled, separate go module with pinned dependencies.
	@echo "(re)installing $(GOBIN)/fips-detect-v0.0.0-20230309083406-7157dae5bafd"
	@cd $(BINGO_DIR) && GOWORK=off $(GO) build -mod=mod -modfile=fips-detect.mod -o=$(GOBIN)/fips-detect-v0.0.0-20230309083406-7157dae5bafd "github.com/acardace/fips-detect"

GENCOSMOSDB := $(GOBIN)/gencosmosdb-v0.0.0-20251002164525-db0ff9dc7ecc
$(GENCOSMOSDB): $(BINGO_DIR)/gencosmosdb.mod
	@# Install binary/ries using Go 1.14+ build command. This is using bwplotka/bingo-controlled, separate go module with pinned dependencies.
	@echo "(re)installing $(GOBIN)/gencosmosdb-v0.0.0-20251002164525-db0ff9dc7ecc"
	@cd $(BINGO_DIR) && GOWORK=off $(GO) build -mod=mod -modfile=gencosmosdb.mod -o=$(GOBIN)/gencosmosdb-v0.0.0-20251002164525-db0ff9dc7ecc "github.com/bennerv/go-cosmosdb/cmd/gencosmosdb"

GO_BINDATA := $(GOBIN)/go-bindata-v3.1.2+incompatible
$(GO_BINDATA): $(BINGO_DIR)/go-bindata.mod
	@# Install binary/ries using Go 1.14+ build command. This is using bwplotka/bingo-controlled, separate go module with pinned dependencies.
	@echo "(re)installing $(GOBIN)/go-bindata-v3.1.2+incompatible"
	@cd $(BINGO_DIR) && GOWORK=off $(GO) build -mod=mod -modfile=go-bindata.mod -o=$(GOBIN)/go-bindata-v3.1.2+incompatible "github.com/go-bindata/go-bindata/go-bindata"

GOCOV_XML := $(GOBIN)/gocov-xml-v1.1.0
$(GOCOV_XML): $(BINGO_DIR)/gocov-xml.mod
	@# Install binary/ries using Go 1.14+ build command. This is using bwplotka/bingo-controlled, separate go module with pinned dependencies.
	@echo "(re)installing $(GOBIN)/gocov-xml-v1.1.0"
	@cd $(BINGO_DIR) && GOWORK=off $(GO) build -mod=mod -modfile=gocov-xml.mod -o=$(GOBIN)/gocov-xml-v1.1.0 "github.com/AlekSi/gocov-xml"

GOCOV := $(GOBIN)/gocov-v1.1.0
$(GOCOV): $(BINGO_DIR)/gocov.mod
	@# Install binary/ries using Go 1.14+ build command. This is using bwplotka/bingo-controlled, separate go module with pinned dependencies.
	@echo "(re)installing $(GOBIN)/gocov-v1.1.0"
	@cd $(BINGO_DIR) && GOWORK=off $(GO) build -mod=mod -modfile=gocov.mod -o=$(GOBIN)/gocov-v1.1.0 "github.com/axw/gocov/gocov"

GOJQ := $(GOBIN)/gojq-v0.12.16
$(GOJQ): $(BINGO_DIR)/gojq.mod
	@# Install binary/ries using Go 1.14+ build command. This is using bwplotka/bingo-controlled, separate go module with pinned dependencies.
	@echo "(re)installing $(GOBIN)/gojq-v0.12.16"
	@cd $(BINGO_DIR) && GOWORK=off $(GO) build -mod=mod -modfile=gojq.mod -o=$(GOBIN)/gojq-v0.12.16 "github.com/itchyny/gojq/cmd/gojq"

GOLANGCI_LINT := $(GOBIN)/golangci-lint-v2.8.0
$(GOLANGCI_LINT): $(BINGO_DIR)/golangci-lint.mod
	@# Install binary/ries using Go 1.14+ build command. This is using bwplotka/bingo-controlled, separate go module with pinned dependencies.
	@echo "(re)installing $(GOBIN)/golangci-lint-v2.8.0"
	@cd $(BINGO_DIR) && GOWORK=off $(GO) build -mod=mod -modfile=golangci-lint.mod -o=$(GOBIN)/golangci-lint-v2.8.0 "github.com/golangci/golangci-lint/v2/cmd/golangci-lint"

GOTESTSUM := $(GOBIN)/gotestsum-v1.13.0
$(GOTESTSUM): $(BINGO_DIR)/gotestsum.mod
	@# Install binary/ries using Go 1.14+ build command. This is using bwplotka/bingo-controlled, separate go module with pinned dependencies.
	@echo "(re)installing $(GOBIN)/gotestsum-v1.13.0"
	@cd $(BINGO_DIR) && GOWORK=off $(GO) build -mod=mod -modfile=gotestsum.mod -o=$(GOBIN)/gotestsum-v1.13.0 "gotest.tools/gotestsum"

GOVULNCHECK := $(GOBIN)/govulncheck-v1.1.4
$(GOVULNCHECK): $(BINGO_DIR)/govulncheck.mod
	@# Install binary/ries using Go 1.14+ build command. This is using bwplotka/bingo-controlled, separate go module with pinned dependencies.
	@echo "(re)installing $(GOBIN)/govulncheck-v1.1.4"
	@cd $(BINGO_DIR) && GOWORK=off $(GO) build -mod=mod -modfile=govulncheck.mod -o=$(GOBIN)/govulncheck-v1.1.4 "golang.org/x/vuln/cmd/govulncheck"

MOCKGEN := $(GOBIN)/mockgen-v0.6.0
$(MOCKGEN): $(BINGO_DIR)/mockgen.mod
	@# Install binary/ries using Go 1.14+ build command. This is using bwplotka/bingo-controlled, separate go module with pinned dependencies.
	@echo "(re)installing $(GOBIN)/mockgen-v0.6.0"
	@cd $(BINGO_DIR) && GOWORK=off $(GO) build -mod=mod -modfile=mockgen.mod -o=$(GOBIN)/mockgen-v0.6.0 "go.uber.org/mock/mockgen"

PINACT := $(GOBIN)/pinact-v1.6.0
$(PINACT): $(BINGO_DIR)/pinact.mod
	@# Install binary/ries using Go 1.14+ build command. This is using bwplotka/bingo-controlled, separate go module with pinned dependencies.
	@echo "(re)installing $(GOBIN)/pinact-v1.6.0"
	@cd $(BINGO_DIR) && GOWORK=off $(GO) build -mod=mod -modfile=pinact.mod -o=$(GOBIN)/pinact-v1.6.0 "github.com/suzuki-shunsuke/pinact/cmd/pinact"

