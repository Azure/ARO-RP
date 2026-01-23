SHELL = /bin/bash
TAG ?= $(shell git describe --exact-match 2>/dev/null)
COMMIT = $(shell git rev-parse --short=7 HEAD)$(shell [[ $$(git status --porcelain) = "" ]] || echo -dirty)
ARO_IMAGE_BASE = ${RP_IMAGE_ACR}.azurecr.io/aro
E2E_FLAGS ?= -test.v --ginkgo.vv --ginkgo.timeout 180m --ginkgo.flake-attempts=2 --ginkgo.junit-report=e2e-report.xml
E2E_LABEL ?= !smoke&&!regressiontest
GO_FLAGS ?= -tags=containers_image_openpgp,exclude_graphdriver_btrfs,exclude_graphdriver_devicemapper
OC ?= oc

export GOFLAGS=$(GO_FLAGS)

# fluentbit version must also be updated in RP code, see pkg/util/version/const.go
MARINER_VERSION = 20250701
FLUENTBIT_VERSION = 4.0.4
FLUENTBIT_IMAGE ?= ${RP_IMAGE_ACR}.azurecr.io/fluentbit:$(FLUENTBIT_VERSION)-cm$(MARINER_VERSION)
AUTOREST_VERSION = 3.7.2
AUTOREST_IMAGE = arointsvc.azurecr.io/autorest:${AUTOREST_VERSION}
GATEKEEPER_VERSION = v3.19.2

# Golang version go mod tidy compatibility
GOLANG_VERSION ?= $(shell go mod edit -json | jq --raw-output .Go)

include .bingo/Variables.mk

ifneq ($(shell uname -s),Darwin)
    export CGO_CFLAGS=-Dgpgme_off_t=off_t
endif

ifeq ($(TAG),)
	VERSION = $(COMMIT)
else
	VERSION = $(TAG)
endif

# REGISTRY, BUILDER_REGISTRY, and FEDORA_REGISTRY are set conditionally below based on RP_IMAGE_ACR
# default to registry.access.redhat.com for build images on local builds and CI builds without $RP_IMAGE_ACR set.
ifeq ($(RP_IMAGE_ACR),arointsvc)
	REGISTRY = arointsvc.azurecr.io
	BUILDER_REGISTRY = arointsvc.azurecr.io
	FEDORA_REGISTRY = $(REGISTRY)
else ifeq ($(RP_IMAGE_ACR),arosvc)
	REGISTRY = arosvc.azurecr.io
	BUILDER_REGISTRY = arosvc.azurecr.io
	FEDORA_REGISTRY = $(REGISTRY)
else ifeq ($(RP_IMAGE_ACR),)
	REGISTRY ?= registry.access.redhat.com
	BUILDER_REGISTRY ?= quay.io/openshift-release-dev
	FEDORA_REGISTRY ?= arointsvc.azurecr.io
else
	REGISTRY = $(RP_IMAGE_ACR)
	BUILDER_REGISTRY = quay.io/openshift-release-dev
	FEDORA_REGISTRY = $(REGISTRY)
endif

# prod images
ARO_IMAGE ?= $(ARO_IMAGE_BASE):$(VERSION)
GATEKEEPER_IMAGE ?= ${REGISTRY}/gatekeeper:$(GATEKEEPER_VERSION)


help:  ## Show help message
	@awk 'BEGIN {FS = ": .*##"; printf "\nUsage:\n  make \033[36m\033[0m\n"} /^[$$()% 0-9a-zA-Z._-]+(\\:[$$()% 0-9a-zA-Z._-]+)*:.*?##/ { gsub(/\\:/,":", $$1); printf "  \033[36m%-25s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

.PHONY: check-release
check-release: ## Check that VERSION is a valid tag when building an official release (when RELEASE=true)
ifeq ($(RELEASE), true)
ifeq ($(TAG), $(VERSION))
	@echo Building release version $(VERSION)
else
	$(error $(shell git describe --exact-match) Ensure there is an annotated tag (git tag -a) for git commit $(COMMIT))
endif
endif

.PHONY: build-all
build-all: ## Build all Go binaries
	go build ./...

.PHONY: aro
aro: check-release generate ## Build the ARO RP binary
	go build -ldflags "-X github.com/Azure/ARO-RP/pkg/util/version.GitCommit=$(VERSION)" ./cmd/aro

.PHONY: runlocal-rp
runlocal-rp: ## Run RP resource provider locally
	go run -ldflags "-X github.com/Azure/ARO-RP/pkg/util/version.GitCommit=$(VERSION)" ./cmd/aro ${ARO_CMD_ARGS} rp

.PHONY: runlocal-monitor
runlocal-monitor: ## Run the monitor locally
	go run -ldflags "-X github.com/Azure/ARO-RP/pkg/util/version.GitCommit=$(VERSION)" ./cmd/aro ${ARO_CMD_ARGS} monitor

.PHONY: az
az: pyenv ## Building development az aro extension
	. pyenv/bin/activate && \
	cd python/az/aro && \
	python3 ./setup.py bdist_egg && \
	python3 ./setup.py bdist_wheel || true && \
	rm -f ~/.azure/commandIndex.json # https://github.com/Azure/azure-cli/issues/14997

# Don't depend on az as that will reinstall the requirements.txt which makes this pointless.
.PHONY: az-freeze
az-freeze: ## Freeze the dependencies of the current pyenv for hoped-for reproducibility.
	. pyenv/bin/activate && \
	pip freeze > requirements.txt

.PHONY: clean
clean: ## Remove build artifacts
	rm -rf python/az/aro/{aro.egg-info,build,dist} aro
	find python -type f -name '*.pyc' -delete
	find python -type d -name __pycache__ -delete
	find -type d -name 'gomock_reflect_[0-9]*' -exec rm -rf {} \+ 2>/dev/null

.PHONY: client
client: generate client-generate lint-go-fix lint-go

.PHONY: client-generate
client-generate: ## Fix stale client library
	hack/apiclients/generate-swagger-checksum.sh 2020-04-30 2021-09-01-preview 2022-04-01 2022-09-04 2023-04-01 2023-07-01-preview 2023-09-04 2023-11-22 2024-08-12-preview 2025-07-25
# Only generate the clients we use in our dev Python extension or in e2e clients
	hack/apiclients/build-dev-api-clients.sh "${AUTOREST_IMAGE}" 2024-08-12-preview 2025-07-25

# TODO: hard coding dev-config.yaml is clunky; it is also probably convenient to
# override COMMIT.
.PHONY: deploy
deploy: ## Deploy RP resources on Azure
	go run -ldflags "-X github.com/Azure/ARO-RP/pkg/util/version.GitCommit=$(VERSION)" ./cmd/aro deploy dev-config.yaml ${LOCATION}

.PHONY: dev-config.yaml
dev-config.yaml: ## Generate dev-config.yaml file
	go run ./hack/gendevconfig >dev-config.yaml

.PHONY: discoverycache
discoverycache: ## Fix out-of-date discovery cache
	$(MAKE) admin.kubeconfig
	KUBECONFIG=admin.kubeconfig go run ./hack/gendiscoverycache
	$(MAKE) generate

.PHONY: generate
generate: install-tools generate-swagger ## Generate files & content for serving ARO-RP
	go generate ./...
	$(MAKE) imports

.PHONY: generate-swagger
generate-swagger:
	go run ./hack/swagger github.com/Azure/ARO-RP/pkg/api/v20240812preview ./swagger/redhatopenshift/resource-manager/Microsoft.RedHatOpenShift/openshiftclusters/preview/2024-08-12-preview
	go run ./hack/swagger github.com/Azure/ARO-RP/pkg/api/v20250725 ./swagger/redhatopenshift/resource-manager/Microsoft.RedHatOpenShift/openshiftclusters/stable/2025-07-25

# TODO: This does not work outside of GOROOT. We should replace all usage of the
# clientset with controller-runtime so we don't need to generate it.
.PHONY: generate-operator-apiclient
generate-operator-apiclient: $(CLIENT_GEN)
	$(CLIENT_GEN) --clientset-name versioned --input-base ./pkg/operator/apis --input aro.openshift.io/v1alpha1,preview.aro.openshift.io/v1alpha1 --output-package ./pkg/operator/clientset --go-header-file ./hack/licenses/boilerplate.go.txt
	gofmt -s -w ./pkg/operator/clientset
	$(MAKE) imports

.PHONY: generate-guardrails
generate-guardrails:
	cd pkg/operator/controllers/guardrails/policies && ./scripts/generate.sh > /dev/null

.PHONY: generate-kiota
generate-kiota:
	kiota generate --clean-output -l go -o ./pkg/util/graph/graphsdk -n "github.com/Azure/ARO-RP/pkg/util/graph/graphsdk" -d hack/graphsdk/openapi.yaml -c GraphBaseServiceClient --additional-data=False --backing-store=True
	find ./pkg/util/graph/graphsdk -type f -name "*.go"  -exec sed -i'' -e 's\github.com/azure/aro-rp\github.com/Azure/ARO-RP\g' {} +
	$(MAKE) imports
	go run ./hack/licenses -dirs ./pkg/util/graph/graphsdk

.PHONY: imports
imports: lint-go-fix

.PHONY: validate-imports
validate-imports: imports
	if ! git diff --quiet HEAD; then \
		git diff; \
		echo "You need to run 'make imports' to update import statements and commit them"; \
		exit 1; \
	fi

.PHONY: init-contrib
init-contrib:
	install -v hack/git/hooks/* .git/hooks/

.PHONY: image-aro-multistage
image-aro-multistage:
	docker build --platform=linux/amd64 --network=host --no-cache -f Dockerfile.aro-multistage -t $(ARO_IMAGE) --build-arg REGISTRY=$(REGISTRY) --build-arg BUILDER_REGISTRY=$(BUILDER_REGISTRY) .

.PHONY: image-autorest
image-autorest:
	docker build --platform=linux/amd64 --network=host --no-cache --build-arg AUTOREST_VERSION="${AUTOREST_VERSION}" --build-arg REGISTRY=$(REGISTRY) -f Dockerfile.autorest -t ${AUTOREST_IMAGE} .

.PHONY: image-fluentbit
image-fluentbit:
	docker build --platform=linux/amd64 --network=host --build-arg VERSION=$(FLUENTBIT_VERSION) --build-arg MARINER_VERSION=$(MARINER_VERSION) -f Dockerfile.fluentbit -t $(FLUENTBIT_IMAGE) .

.PHONY: image-proxy
image-proxy:
	docker pull $(REGISTRY)/ubi9/ubi-minimal
	docker build --platform=linux/amd64 --no-cache -f Dockerfile.proxy -t $(REGISTRY)/proxy:latest --build-arg REGISTRY=$(REGISTRY) --build-arg BUILDER_REGISTRY=$(BUILDER_REGISTRY) .

.PHONY: image-gatekeeper
image-gatekeeper:
	docker build --platform=linux/amd64 --network=host --build-arg GATEKEEPER_VERSION=$(GATEKEEPER_VERSION) --build-arg REGISTRY=$(REGISTRY) --build-arg BUILDER_REGISTRY=$(BUILDER_REGISTRY) -f Dockerfile.gatekeeper -t $(GATEKEEPER_IMAGE) .

.PHONY: publish-image-aro-multistage
publish-image-aro-multistage: image-aro-multistage
	docker push $(ARO_IMAGE)
ifeq ("${RP_IMAGE_ACR}-$(BRANCH)","arointsvc-master")
		docker tag $(ARO_IMAGE) arointsvc.azurecr.io/aro:latest
		docker push arointsvc.azurecr.io/aro:latest
endif

.PHONY: publish-image-autorest
publish-image-autorest: image-autorest
	docker push ${AUTOREST_IMAGE}

.PHONY: publish-image-fluentbit
publish-image-fluentbit: image-fluentbit
	docker push $(FLUENTBIT_IMAGE)

.PHONY: publish-image-proxy
publish-image-proxy: image-proxy
	docker push ${RP_IMAGE_ACR}.azurecr.io/proxy:latest

.PHONY: publish-image-gatekeeper
publish-image-gatekeeper: image-gatekeeper
	docker push $(GATEKEEPER_IMAGE)

.PHONY: image-e2e
image-e2e:
	docker build --platform=linux/amd64 --network=host --no-cache -f Dockerfile.aro-e2e -t $(ARO_IMAGE) --build-arg REGISTRY=$(REGISTRY) --build-arg BUILDER_REGISTRY=$(BUILDER_REGISTRY) .

.PHONY: publish-image-e2e
publish-image-e2e: image-e2e
	docker push $(ARO_IMAGE)

.PHONY: extract-aro-docker
extract-aro-docker:
	hack/ci-utils/extractaro.sh ${ARO_IMAGE}

.PHONY: proxy
proxy:
	CGO_ENABLED=0 go build -ldflags "-X github.com/Azure/ARO-RP/pkg/util/version.GitCommit=$(VERSION)" ./hack/proxy

.PHONY: runlocal-portal
runlocal-portal:
	go run -ldflags "-X github.com/Azure/ARO-RP/pkg/util/version.GitCommit=$(VERSION)" ./cmd/aro ${ARO_CMD_ARGS} portal

.PHONY: runlocal-actuator
runlocal-actuator:
	go run -ldflags "-X github.com/Azure/ARO-RP/pkg/util/version.GitCommit=$(VERSION)" ./cmd/aro ${ARO_CMD_ARGS} mimo-actuator

.PHONY: build-portal
build-portal:
	cd portal/v2 && npm install && npm run build

.PHONY: pyenv
pyenv:
	python3 -m venv pyenv
	. pyenv/bin/activate && \
		pip install -U pip && \
		pip install -r requirements.txt && \
		azdev setup -r . && \
		az config set extension.dev_sources=$(PWD)/python

.PHONY: secrets
secrets:
	@[ "${SECRET_SA_ACCOUNT_NAME}" ] || ( echo ">> SECRET_SA_ACCOUNT_NAME is not set"; exit 1 )
	rm -rf secrets
	az storage blob download -n secrets.tar.gz -c secrets -f secrets.tar.gz --account-name ${SECRET_SA_ACCOUNT_NAME} >/dev/null
	tar -xzf secrets.tar.gz
	rm secrets.tar.gz

.PHONY: secrets-update
secrets-update:
	@[ "${SECRET_SA_ACCOUNT_NAME}" ] || ( echo ">> SECRET_SA_ACCOUNT_NAME is not set"; exit 1 )
	tar -czf secrets.tar.gz secrets
	az storage blob upload -n secrets.tar.gz -c secrets -f secrets.tar.gz --overwrite --account-name ${SECRET_SA_ACCOUNT_NAME} >/dev/null
	rm secrets.tar.gz

.PHONY: tunnel
tunnel:
	go run ./hack/tunnel $(shell az network public-ip show -g ${RESOURCEGROUP} -n rp-pip --query 'ipAddress')

.PHONY: e2e.test
e2e.test:
	go test ./test/e2e/... -tags e2e,codec.safe -c -ldflags "-X github.com/Azure/ARO-RP/pkg/util/version.GitCommit=$(VERSION)" -o e2e.test

.PHONY: e2etools
e2etools:
	CGO_ENABLED=0 go build -ldflags "-X github.com/Azure/ARO-RP/pkg/util/version.GitCommit=$(VERSION)" ./hack/cluster
	CGO_ENABLED=0 go build -ldflags "-X github.com/Azure/ARO-RP/pkg/util/version.GitCommit=$(VERSION)" ./hack/db
	CGO_ENABLED=0 go build -ldflags "-X github.com/Azure/ARO-RP/pkg/util/version.GitCommit=$(VERSION)" ./hack/portalauth
	$(BINGO) get -l gojq

.PHONY: test-e2e
test-e2e: e2e.test
	./e2e.test $(E2E_FLAGS) --ginkgo.label-filter="$(E2E_LABEL)"

.PHONY: test-go
test-go: generate build-all validate-go lint-go unit-test-go

.PHONY: validate-go
validate-go: validate-imports
	gofmt -s -w cmd hack pkg test
	go run ./hack/licenses
	@[ -z "$$(ls pkg/util/*.go 2>/dev/null)" ] || (echo error: go files are not allowed in pkg/util, use a subpackage; exit 1)
	@[ -z "$$(find -name "*:*")" ] || (echo error: filenames with colons are not allowed on Windows, please rename; exit 1)
	@sha256sum --quiet -c .sha256sum || (echo error: client library is stale, please run make client; exit 1)
	go test -tags e2e -run ^$$ ./test/e2e/...

.PHONY: validate-go-action
validate-go-action: validate-imports validate-lint-go-fix
	go run ./hack/licenses -validate -ignored-go vendor,pkg/client,.git -ignored-python python/client,python/az/aro/azext_aro/aaz,vendor,.git
	@[ -z "$$(ls pkg/util/*.go 2>/dev/null)" ] || (echo error: go files are not allowed in pkg/util, use a subpackage; exit 1)
	@[ -z "$$(find -name "*:*")" ] || (echo error: filenames with colons are not allowed on Windows, please rename; exit 1)
	@sha256sum --quiet -c .sha256sum || (echo error: client library is stale, please run make client; exit 1)

.PHONY: validate-fips
validate-fips: $(BINGO)
	$(BINGO) get -l fips-detect
	$(BINGO) get -l gojq
	hack/fips/validate-fips.sh ./aro

.PHONY: unit-test-go
unit-test-go: $(GOTESTSUM)
	$(GOTESTSUM) --format pkgname --junitfile report.xml -- -coverprofile=cover.out ./...

.PHONY: unit-test-go-coverpkg
unit-test-go-coverpkg: $(GOTESTSUM)
	$(GOTESTSUM) --format pkgname --junitfile report.xml -- -coverpkg=./... -coverprofile=cover_coverpkg.out ./...

.PHONY: lint-go
lint-go: $(GOLANGCI_LINT)
	$(GOLANGCI_LINT) run --verbose

.PHONY: lint-go-fix
lint-go-fix: $(GOLANGCI_LINT)
	$(GOLANGCI_LINT) run --verbose --fix
	cd pkg/api/ && $(GOLANGCI_LINT) run --verbose --fix ./...

.PHONY: validate-lint-go-fix
validate-lint-go-fix: lint-go-fix
	if ! git diff --quiet HEAD; then \
		git diff; \
		echo "You need to run 'make lint-go-fix' to update the codebase and commit the changes"; \
		exit 1; \
	fi

.PHONY: lint-admin-portal
lint-admin-portal:
	docker build --platform=linux/amd64 --build-arg REGISTRY=$(REGISTRY) --build-arg BUILDER_REGISTRY=$(BUILDER_REGISTRY) -f Dockerfile.portal_lint . -t linter:latest --no-cache
	docker run --platform=linux/amd64 -t --rm linter:latest

.PHONY: test-python
test-python: pyenv az
	. pyenv/bin/activate && \
		azdev linter && \
		azdev style && \
		hack/unit-test-python.sh

.PHONY: test-python-podman
test-python-podman:
	rm -rf pyenv
	docker run --platform=linux/amd64 -t --rm \
	    -v ./:/app:z \
		--user=0 \
	 	$(REGISTRY)/ubi9/python-312:latest \
		bash -c "cd /app && ls && make test-python"

.PHONY: shared-cluster-login
shared-cluster-login:
	@oc login $(shell az aro show -g sre-shared-cluster -n sre-shared-cluster -ojson --query apiserverProfile.url) \
		-u kubeadmin \
		-p $(shell az aro list-credentials -g sre-shared-cluster -n sre-shared-cluster  -ojson --query "kubeadminPassword")

.PHONY: shared-miwi-cluster-login
shared-miwi-cluster-login:
	@oc login $(shell az aro show -g sre-shared-miwi-cluster -n sre-shared-miwi-cluster -ojson --query apiserverProfile.url) \
		-u kubeadmin \
		-p $(shell az aro list-credentials -g sre-shared-miwi-cluster -n sre-shared-miwi-cluster  -ojson --query "kubeadminPassword")

.PHONY: shared-cluster-create
shared-cluster-create:
	./hack/shared-cluster.sh create

.PHONY: shared-cluster-delete
shared-cluster-delete:
	./hack/shared-cluster.sh delete

.PHONY: shared-miwi-cluster-create
shared-miwi-cluster-create:
	./hack/shared-miwi-cluster.sh create

.PHONY: shared-miwi-cluster-delete
shared-miwi-cluster-delete:
	./hack/shared-miwi-cluster.sh delete

.PHONY: unit-test-python
unit-test-python:
	hack/unit-test-python.sh

.PHONY: admin.kubeconfig
admin.kubeconfig: ## Get cluster admin kubeconfig
	hack/get-admin-kubeconfig.sh /subscriptions/${AZURE_SUBSCRIPTION_ID}/resourceGroups/${RESOURCEGROUP}/providers/Microsoft.RedHatOpenShift/openShiftClusters/${CLUSTER} >admin.kubeconfig

.PHONY: aks.kubeconfig
aks.kubeconfig: ## Get AKS admin kubeconfig
	hack/get-admin-aks-kubeconfig.sh

.PHONY: go-tidy
go-tidy: # Run go mod tidy - add missing and remove unused modules.
	echo "tidying main module"
	go mod tidy -compat=${GOLANG_VERSION}
	echo "tidying pkg/api/"
	cd pkg/api/ && go mod tidy -compat=${GOLANG_VERSION}

.PHONY: go-verify
go-verify: go-tidy # Run go mod verify - verify dependencies have expected content
	echo "verifying main module"
	go mod verify
	echo "verifying pkg/api/"
	cd pkg/api/ && go mod verify

.PHONY: xmlcov
xmlcov: $(GOCOV) $(GOCOV_XML)
	$(GOCOV) convert cover.out | $(GOCOV_XML) > coverage.xml

.PHONY: install-tools
install-tools: $(BINGO)
	$(BINGO) get -l
# Fixes https://github.com/uber-go/mock/issues/185 for MacOS users
ifeq ($(shell uname -s),Darwin)
	codesign -f -s - ${GOPATH}/bin/mockgen
endif

###############################################################################
# Containerized CI/CD RP
###############################################################################

###############################################################################
# Config
###############################################################################

# REGISTRY is used from above, and defines what base images are used in the
# build process

# VERSION is used from above, and is used during binary compile time and for
# all image tags.

# Configures use of podman build cache. Likely you want `false` locally. Always
# `true` in CI.
NO_CACHE ?= true

# Useful for CI pipelines where we need to run podman as a service, and specify
# that service as a URL (see .pipelines/ci.yml). This should be invoked on all
# use of `podman` in the Makefile.
PODMAN_REMOTE_ARGS ?=
DOCKER_BUILD_CI_ARGS ?=

# Image names that will be found in the local podman image registry after build
# (tags are always VERSION).
LOCAL_ARO_RP_IMAGE ?= aro
LOCAL_E2E_IMAGE ?= e2e
LOCAL_ARO_AZEXT_IMAGE ?= azext-aro
LOCAL_TUNNEL_IMAGE ?= aro-tunnel

###############################################################################
# Targets
###############################################################################
.PHONY: ci-azext-aro
ci-azext-aro:
	docker build . $(DOCKER_BUILD_CI_ARGS) \
		-f Dockerfile.ci-azext-aro \
		--platform=linux/amd64 \
		--no-cache=$(NO_CACHE) \
		-t $(LOCAL_ARO_AZEXT_IMAGE):$(VERSION)

.PHONY: ci-clean
ci-clean:
	$(shell podman $(PODMAN_REMOTE_ARGS) ps --external --format "{{.Command}} {{.ID}}" | grep buildah | cut -d " " -f 2 | xargs podman $(PODMAN_REMOTE_ARGS) rm -f > /dev/null)
	podman $(PODMAN_REMOTE_ARGS) \
	    image prune --all --filter="label=aro-*=true"

.PHONY: ci-rp
ci-rp:
	docker build . ${DOCKER_BUILD_CI_ARGS} \
		-f Dockerfile.ci-rp \
		--ulimit=nofile=4096:4096 \
		--build-arg REGISTRY=$(REGISTRY) \
		--build-arg BUILDER_REGISTRY=$(BUILDER_REGISTRY) \
		--build-arg ARO_VERSION=$(VERSION) \
		--no-cache=$(NO_CACHE) \
		-t $(LOCAL_ARO_RP_IMAGE):$(VERSION)

	# Extract test coverage files from build to local filesystem
	docker create --name extract_cover_out ${LOCAL_ARO_RP_IMAGE}:${VERSION}; \
	docker cp extract_cover_out:/app/report.xml ./report.xml; \
	docker cp extract_cover_out:/app/coverage.xml ./coverage.xml; \
	docker rm extract_cover_out;

.PHONY: aro-e2e
aro-e2e:
	docker build . ${DOCKER_BUILD_CI_ARGS} \
		-f Dockerfile.aro-e2e \
		--ulimit=nofile=4096:4096 \
		--build-arg REGISTRY=$(REGISTRY) \
		--build-arg BUILDER_REGISTRY=$(BUILDER_REGISTRY) \
		--build-arg ARO_VERSION=$(VERSION) \
		--no-cache=$(NO_CACHE) \
		-t $(LOCAL_E2E_IMAGE):$(VERSION)

.PHONY: ci-tunnel
ci-tunnel:
	podman $(PODMAN_REMOTE_ARGS) \
	    build . \
		-f Dockerfile.ci-tunnel \
		--ulimit=nofile=4096:4096 \
		--build-arg REGISTRY=$(REGISTRY) \
		--build-arg BUILDER_REGISTRY=$(BUILDER_REGISTRY) \
		--build-arg ARO_VERSION=$(VERSION) \
		--no-cache=$(NO_CACHE) \
		-t $(LOCAL_TUNNEL_IMAGE):$(VERSION)

.PHONY: run-portal
run-portal:
	docker compose up portal

.PHONY: run-rp
run-rp: aks.kubeconfig ## Run RP locally as similarly as possible to production, including Hive. Requires a VPN connection.
	docker compose rm -sf rp
	docker compose up rp

.PHONY: acr-login
acr-login: ## Login to arointsvc ACR using PULL_SECRET
	@. hack/devtools/rp_dev_helper.sh && acr_login

.PHONY: dev-env-build
dev-env-build: acr-login ## Build the dev environment container image
	FEDORA_REGISTRY=$(FEDORA_REGISTRY) podman compose build aro-dev-env

.PHONY: dev-env-start
dev-env-start: acr-login ## Start the dev environment RP container
	FEDORA_REGISTRY=$(FEDORA_REGISTRY) podman compose up -d aro-dev-env

.PHONY: dev-env-stop
dev-env-stop: ## Stop the containerized RP
	FEDORA_REGISTRY=$(FEDORA_REGISTRY) podman compose down aro-dev-env

.PHONY: run-selenium
run-selenium:
	docker compose up selenium

.PHONY: validate-roledef
validate-roledef:
	go run ./hack/role -verified-version "$(OCP_VERSION)" -oc-bin=$(OC)
