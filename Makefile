SHELL = /bin/bash
TAG ?= $(shell git describe --exact-match 2>/dev/null)
COMMIT = $(shell git rev-parse --short=7 HEAD)$(shell [[ $$(git status --porcelain) = "" ]] || echo -dirty)
ARO_IMAGE_BASE = ${RP_IMAGE_ACR}.azurecr.io/aro
E2E_FLAGS ?= -test.timeout 180m -test.v -ginkgo.v

# fluentbit version must also be updated in RP code, see pkg/util/version/const.go
FLUENTBIT_VERSION = 1.7.8-1
FLUENTBIT_IMAGE ?= ${RP_IMAGE_ACR}.azurecr.io/fluentbit:$(FLUENTBIT_VERSION)
AUTOREST_VERSION = 3.3.2
AUTOREST_IMAGE = "quay.io/openshift-on-azure/autorest:${AUTOREST_VERSION}"

ifneq ($(shell uname -s),Darwin)
    export CGO_CFLAGS=-Dgpgme_off_t=off_t
endif

ifeq ($(TAG),)
	VERSION = $(COMMIT)
else
	VERSION = $(TAG)
endif

ARO_IMAGE ?= $(ARO_IMAGE_BASE):$(VERSION)

build-all:
	go build -tags aro,containers_image_openpgp ./...

aro: generate
	go build -tags aro,containers_image_openpgp,codec.safe -ldflags "-X github.com/Azure/ARO-RP/pkg/util/version.GitCommit=$(VERSION)" ./cmd/aro

runlocal-rp:
	go run -tags aro -ldflags "-X github.com/Azure/ARO-RP/pkg/util/version.GitCommit=$(VERSION)" ./cmd/aro rp

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
	hack/build-client.sh "${AUTOREST_IMAGE}" 2020-04-30 2021-09-01-preview

# TODO: hard coding dev-config.yaml is clunky; it is also probably convenient to
# override COMMIT.
deploy:
	go run -tags aro -ldflags "-X github.com/Azure/ARO-RP/pkg/util/version.GitCommit=$(VERSION)" ./cmd/aro deploy dev-config.yaml ${LOCATION}

dev-config.yaml:
	go run ./hack/gendevconfig >dev-config.yaml

discoverycache:
	$(MAKE) admin.kubeconfig
	KUBECONFIG=admin.kubeconfig go run ./hack/gendiscoverycache
	$(MAKE) generate

generate:
	go generate ./...

image-aro: aro e2e.test
	docker pull registry.access.redhat.com/ubi8/ubi-minimal
	docker build --network=host --no-cache -f Dockerfile.aro -t $(ARO_IMAGE) .

image-aro-multistage:
	docker build --network=host --no-cache -f Dockerfile.aro-multistage -t $(ARO_IMAGE) .

image-autorest:
	docker build --network=host --no-cache --build-arg AUTOREST_VERSION="${AUTOREST_VERSION}" \
	  -f Dockerfile.autorest -t ${AUTOREST_IMAGE} .

image-fluentbit:
	docker build --network=host --no-cache --build-arg VERSION=$(FLUENTBIT_VERSION) \
	  -f Dockerfile.fluentbit -t $(FLUENTBIT_IMAGE) .

image-proxy: proxy
	docker pull registry.access.redhat.com/ubi8/ubi-minimal
	docker build --no-cache -f Dockerfile.proxy -t ${RP_IMAGE_ACR}.azurecr.io/proxy:latest .

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

proxy:
	go build -ldflags "-X github.com/Azure/ARO-RP/pkg/util/version.GitCommit=$(VERSION)" ./hack/proxy

run-portal:
	go run -tags aro -ldflags "-X github.com/Azure/ARO-RP/pkg/util/version.GitCommit=$(VERSION)" ./cmd/aro portal

build-portal:
	cd portal && npm install && npm run build

pyenv:
	python3 -m venv pyenv
	. pyenv/bin/activate && \
		pip install -U pip && \
		pip install autopep8 azdev azure-mgmt-loganalytics==0.2.0 colorama ruamel.yaml wheel && \
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
	az storage blob upload -n secrets.tar.gz -c secrets -f secrets.tar.gz --account-name ${SECRET_SA_ACCOUNT_NAME} >/dev/null
	rm secrets.tar.gz

tunnel:
	go run ./hack/tunnel $(shell az network public-ip show -g ${RESOURCEGROUP} -n rp-pip --query 'ipAddress')

e2e.test:
	go test ./test/e2e -tags e2e,codec.safe -c -ldflags "-X github.com/Azure/ARO-RP/pkg/util/version.GitCommit=$(VERSION)" -o e2e.test

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
	go vet ./...
	go test -tags e2e -run ^$$ ./test/e2e/...

validate-fips:
	hack/fips/validate-fips.sh

unit-test-go:
	go run ./vendor/gotest.tools/gotestsum/main.go --format pkgname --junitfile report.xml -- -tags=aro -coverprofile=cover.out ./...

lint-go:
	go run ./vendor/github.com/golangci/golangci-lint/cmd/golangci-lint run

test-python: pyenv az
	. pyenv/bin/activate && \
		azdev linter && \
		azdev style && \
		hack/format-yaml/format-yaml.py .pipelines

admin.kubeconfig:
	hack/get-admin-kubeconfig.sh /subscriptions/${AZURE_SUBSCRIPTION_ID}/resourceGroups/${RESOURCEGROUP}/providers/Microsoft.RedHatOpenShift/openShiftClusters/${CLUSTER} >admin.kubeconfig

vendor:
	# See comments in the script for background on why we need it
	hack/update-go-module-dependencies.sh

.PHONY: admin.kubeconfig aro az clean client deploy dev-config.yaml discoverycache generate image-aro image-aro-multistage image-fluentbit image-proxy lint-go runlocal-rp proxy publish-image-aro publish-image-aro-multistage publish-image-fluentbit publish-image-proxy secrets secrets-update e2e.test tunnel test-e2e test-go test-python vendor build-all validate-go  unit-test-go coverage-go validate-fips
