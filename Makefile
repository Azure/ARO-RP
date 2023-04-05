SHELL = /bin/bash
TAG ?= $(shell git describe --exact-match 2>/dev/null)
COMMIT = $(shell git rev-parse --short=7 HEAD)$(shell [[ $$(git status --porcelain) = "" ]] || echo -dirty)
ARO_IMAGE_BASE = ${RP_IMAGE_ACR}.azurecr.io/aro
E2E_FLAGS ?= -test.v --ginkgo.v --ginkgo.timeout 180m --ginkgo.flake-attempts=2 --ginkgo.junit-report=e2e-report.xml

# fluentbit version must also be updated in RP code, see pkg/util/version/const.go
MARINER_VERSION = 20230321
FLUENTBIT_VERSION = 1.9.10
FLUENTBIT_IMAGE ?= ${RP_IMAGE_ACR}.azurecr.io/fluentbit:$(FLUENTBIT_VERSION)-cm$(MARINER_VERSION)
AUTOREST_VERSION = 3.6.2
AUTOREST_IMAGE = quay.io/openshift-on-azure/autorest:${AUTOREST_VERSION}

ifneq ($(shell uname -s),Darwin)
    export CGO_CFLAGS=-Dgpgme_off_t=off_t
endif

ifeq ($(TAG),)
	VERSION = $(COMMIT)
else
	VERSION = $(TAG)
endif

# default to registry.access.redhat.com for build images on local builds and CI builds without $RP_IMAGE_ACR set.
ifeq ($(RP_IMAGE_ACR),arointsvc)
	REGISTRY = arointsvc.azurecr.io
else ifeq ($(RP_IMAGE_ACR),arosvc)
	REGISTRY = arosvc.azurecr.io
else ifeq ($(RP_IMAGE_ACR),)
	REGISTRY = registry.access.redhat.com
else
	REGISTRY = $(RP_IMAGE_ACR)
endif

ARO_IMAGE ?= $(ARO_IMAGE_BASE):$(VERSION)

check-release:
# Check that VERSION is a valid tag when building an official release (when RELEASE=true).
ifeq ($(RELEASE), true)
ifeq ($(TAG), $(VERSION))
	@echo Building release version $(VERSION)
else
	$(error $(shell git describe --exact-match) Ensure there is an annotated tag (git tag -a) for git commit $(COMMIT))
endif
endif

build-all:
	go build -tags aro,containers_image_openpgp ./...

aro: check-release generate
	go build -tags aro,containers_image_openpgp,codec.safe -ldflags "-X github.com/Azure/ARO-RP/pkg/util/version.GitCommit=$(VERSION)" ./cmd/aro

runlocal-rp:
	go run -tags aro,containers_image_openpgp -ldflags "-X github.com/Azure/ARO-RP/pkg/util/version.GitCommit=$(VERSION)" ./cmd/aro rp

az: pyenv
	. pyenv/bin/activate && \
	cd python/az/aro && \
	python3 ./setup.py bdist_egg && \
	python3 ./setup.py bdist_wheel || true && \
	rm -f ~/.azure/commandIndex.json # https://github.com/Azure/azure-cli/issues/14997

clean:
	rm -rf python/az/aro/{aro.egg-info,build,dist} aro
	find python -type f -name '*.pyc' -delete
	find python -type d -name __pycache__ -delete
	find -type d -name 'gomock_reflect_[0-9]*' -exec rm -rf {} \+ 2>/dev/null

client: generate
	hack/build-client.sh "${AUTOREST_IMAGE}" 2020-04-30 2021-09-01-preview 2022-04-01 2022-09-04 2023-04-01

# TODO: hard coding dev-config.yaml is clunky; it is also probably convenient to
# override COMMIT.
deploy:
	go run -tags aro,containers_image_openpgp -ldflags "-X github.com/Azure/ARO-RP/pkg/util/version.GitCommit=$(VERSION)" ./cmd/aro deploy dev-config.yaml ${LOCATION}

dev-config.yaml:
	go run ./hack/gendevconfig >dev-config.yaml

discoverycache:
	$(MAKE) admin.kubeconfig
	KUBECONFIG=admin.kubeconfig go run ./hack/gendiscoverycache
	$(MAKE) generate

generate:
	go generate ./...

image-aro: aro e2e.test
	docker pull $(REGISTRY)/ubi8/ubi-minimal
	docker build --platform=linux/amd64 --network=host --no-cache -f Dockerfile.aro -t $(ARO_IMAGE) --build-arg REGISTRY=$(REGISTRY) .

image-aro-multistage:
	docker build --platform=linux/amd64 --network=host --no-cache -f Dockerfile.aro-multistage -t $(ARO_IMAGE) --build-arg REGISTRY=$(REGISTRY) .

image-autorest:
	docker build --platform=linux/amd64 --network=host --no-cache --build-arg AUTOREST_VERSION="${AUTOREST_VERSION}" --build-arg REGISTRY=$(REGISTRY) -f Dockerfile.autorest -t ${AUTOREST_IMAGE} .

image-fluentbit:
	docker build --platform=linux/amd64 --network=host --build-arg VERSION=$(FLUENTBIT_VERSION) --build-arg MARINER_VERSION=$(MARINER_VERSION) -f Dockerfile.fluentbit -t $(FLUENTBIT_IMAGE) .

image-proxy:
	docker pull $(REGISTRY)/ubi8/ubi-minimal
	docker build --platform=linux/amd64 --no-cache -f Dockerfile.proxy -t $(REGISTRY)/proxy:latest --build-arg REGISTRY=$(REGISTRY) .

publish-image-aro: image-aro
	docker push $(ARO_IMAGE)
ifeq ("${RP_IMAGE_ACR}-$(BRANCH)","arointsvc-master")
		docker tag $(ARO_IMAGE) arointsvc.azurecr.io/aro:latest
		docker push arointsvc.azurecr.io/aro:latest
endif

publish-image-aro-multistage: image-aro-multistage
	docker push $(ARO_IMAGE)
ifeq ("${RP_IMAGE_ACR}-$(BRANCH)","arointsvc-master")
		docker tag $(ARO_IMAGE) arointsvc.azurecr.io/aro:latest
		docker push arointsvc.azurecr.io/aro:latest
endif

publish-image-autorest: image-autorest
	docker push ${AUTOREST_IMAGE}

publish-image-fluentbit: image-fluentbit
	docker push $(FLUENTBIT_IMAGE)

publish-image-proxy: image-proxy
	docker push ${RP_IMAGE_ACR}.azurecr.io/proxy:latest

image-e2e:
	docker build --platform=linux/amd64 --network=host --no-cache -f Dockerfile.aro-e2e -t $(ARO_IMAGE) --build-arg REGISTRY=$(REGISTRY) .

publish-image-e2e: image-e2e
	docker push $(ARO_IMAGE)

extract-aro-docker:
	hack/ci-utils/extractaro.sh ${ARO_IMAGE}

proxy:
	CGO_ENABLED=0 go build -ldflags "-X github.com/Azure/ARO-RP/pkg/util/version.GitCommit=$(VERSION)" ./hack/proxy

run-portal:
	go run -tags aro,containers_image_openpgp -ldflags "-X github.com/Azure/ARO-RP/pkg/util/version.GitCommit=$(VERSION)" ./cmd/aro portal

build-portal:
	cd portal/v1 && npm install && npm run build && cd ../v2 && npm install && npm run build

pyenv:
	python3 -m venv pyenv
	. pyenv/bin/activate && \
		pip install -U pip && \
		pip install -r requirements.txt && \
		azdev setup -r . && \
		sed -i -e "s|^dev_sources = $(PWD)$$|dev_sources = $(PWD)/python|" ~/.azure/config

secrets:
	@[ "${SECRET_SA_ACCOUNT_NAME}" ] || ( echo ">> SECRET_SA_ACCOUNT_NAME is not set"; exit 1 )
	rm -rf secrets
	az storage blob download -n secrets.tar.gz -c secrets -f secrets.tar.gz --account-name ${SECRET_SA_ACCOUNT_NAME} >/dev/null
	tar -xzf secrets.tar.gz
	rm secrets.tar.gz

secrets-update:
	@[ "${SECRET_SA_ACCOUNT_NAME}" ] || ( echo ">> SECRET_SA_ACCOUNT_NAME is not set"; exit 1 )
	tar -czf secrets.tar.gz secrets
	az storage blob upload -n secrets.tar.gz -c secrets -f secrets.tar.gz --overwrite --account-name ${SECRET_SA_ACCOUNT_NAME} >/dev/null
	rm secrets.tar.gz

tunnel:
	go run ./hack/tunnel $(shell az network public-ip show -g ${RESOURCEGROUP} -n rp-pip --query 'ipAddress')

e2e.test:
	go test ./test/e2e/... -tags e2e,codec.safe -c -ldflags "-X github.com/Azure/ARO-RP/pkg/util/version.GitCommit=$(VERSION)" -o e2e.test

e2etools:
	CGO_ENABLED=0 go build -ldflags "-X github.com/Azure/ARO-RP/pkg/util/version.GitCommit=$(VERSION)" ./hack/cluster
	CGO_ENABLED=0 go build -ldflags "-X github.com/Azure/ARO-RP/pkg/util/version.GitCommit=$(VERSION)" ./hack/db
	CGO_ENABLED=0 go build -ldflags "-X github.com/Azure/ARO-RP/pkg/util/version.GitCommit=$(VERSION)" ./hack/portalauth

test-e2e: e2e.test
	./e2e.test $(E2E_FLAGS)

test-go: generate build-all validate-go lint-go unit-test-go

validate-go:
	gofmt -s -w cmd hack pkg test
	go run ./vendor/golang.org/x/tools/cmd/goimports -w -local=github.com/Azure/ARO-RP cmd hack pkg test
	go run ./hack/validate-imports cmd hack pkg test
	go run ./hack/licenses
	@[ -z "$$(ls pkg/util/*.go 2>/dev/null)" ] || (echo error: go files are not allowed in pkg/util, use a subpackage; exit 1)
	@[ -z "$$(find -name "*:*")" ] || (echo error: filenames with colons are not allowed on Windows, please rename; exit 1)
	@sha256sum --quiet -c .sha256sum || (echo error: client library is stale, please run make client; exit 1)
	go vet -tags containers_image_openpgp ./...
	go test -tags e2e -run ^$$ ./test/e2e/...

validate-go-action:
	go run ./hack/licenses -validate -ignored-go vendor,pkg/client,.git -ignored-python python/client,vendor,.git
	go run ./hack/validate-imports cmd hack pkg test
	@[ -z "$$(ls pkg/util/*.go 2>/dev/null)" ] || (echo error: go files are not allowed in pkg/util, use a subpackage; exit 1)
	@[ -z "$$(find -name "*:*")" ] || (echo error: filenames with colons are not allowed on Windows, please rename; exit 1)
	@sha256sum --quiet -c .sha256sum || (echo error: client library is stale, please run make client; exit 1)

validate-fips:
	hack/fips/validate-fips.sh

unit-test-go:
	go run gotest.tools/gotestsum@v1.9.0 --format pkgname --junitfile report.xml -- -tags=aro,containers_image_openpgp -coverprofile=cover.out ./...

unit-test-go-coverpkg:
	go run gotest.tools/gotestsum@v1.9.0 --format pkgname --junitfile report.xml -- -tags=aro,containers_image_openpgp -coverpkg=./... -coverprofile=cover_coverpkg.out ./...

lint-go:
	hack/lint-go.sh

lint-admin-portal:
	docker build --platform=linux/amd64 --build-arg REGISTRY=$(REGISTRY) -f Dockerfile.portal_lint . -t linter:latest --no-cache
	docker run --platform=linux/amd64 -t --rm linter:latest

test-python: pyenv az
	. pyenv/bin/activate && \
		azdev linter && \
		azdev style && \
		hack/unit-test-python.sh


shared-cluster-login:
	@oc login ${SHARED_CLUSTER_API} -u kubeadmin -p ${SHARED_CLUSTER_KUBEADMIN_PASSWORD}

unit-test-python:
	hack/unit-test-python.sh

admin.kubeconfig:
	hack/get-admin-kubeconfig.sh /subscriptions/${AZURE_SUBSCRIPTION_ID}/resourceGroups/${RESOURCEGROUP}/providers/Microsoft.RedHatOpenShift/openShiftClusters/${CLUSTER} >admin.kubeconfig

aks.kubeconfig:
	hack/get-admin-aks-kubeconfig.sh

vendor:
	# See comments in the script for background on why we need it
	hack/update-go-module-dependencies.sh

.PHONY: admin.kubeconfig aks.kubeconfig aro az clean client deploy dev-config.yaml discoverycache generate image-aro image-aro-multistage image-fluentbit image-proxy lint-go runlocal-rp proxy publish-image-aro publish-image-aro-multistage publish-image-fluentbit publish-image-proxy secrets secrets-update e2e.test tunnel test-e2e test-go test-python vendor build-all validate-go  unit-test-go coverage-go validate-fips
