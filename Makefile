SHELL = /bin/bash
COMMIT = $(shell git rev-parse --short HEAD)$(shell [[ $$(git status --porcelain) = "" ]] || echo -dirty)
ARO_IMAGE ?= ${RP_IMAGE_ACR}.azurecr.io/aro:$(COMMIT)

export CGO_CFLAGS=-Dgpgme_off_t=off_t

aro: generate
	go build -ldflags "-X github.com/Azure/ARO-RP/pkg/util/version.GitCommit=$(COMMIT)" ./cmd/aro

az: pyenv
	. pyenv/bin/activate && \
	cd python/az/aro && \
	python ./setup.py bdist_egg && \
	python ./setup.py bdist_wheel || true && \
	rm -f ~/.azure/commandIndex.json # https://github.com/Azure/azure-cli/issues/1499

clean:
	rm -rf python/az/aro/{aro.egg-info,build,dist} aro
	find python -type f -name '*.pyc' -delete
	find python -type d -name __pycache__ -delete

client: generate
	hack/build-client.sh 2020-10-31-preview

generate:
	go generate ./...

image-aro: aro e2e.test
	docker pull registry.access.redhat.com/ubi8/ubi-minimal
	docker build --no-cache -f Dockerfile.aro -t $(ARO_IMAGE) .

image-fluentbit:
	docker build --no-cache --build-arg VERSION=1.3.9-1 \
	  -f Dockerfile.fluentbit -t ${RP_IMAGE_ACR}.azurecr.io/fluentbit:1.3.9-1 .

image-proxy: proxy
	docker pull registry.access.redhat.com/ubi8/ubi-minimal
	docker build --no-cache -f Dockerfile.proxy -t ${RP_IMAGE_ACR}.azurecr.io/proxy:latest .

image-routefix:
	docker pull registry.access.redhat.com/ubi8/ubi
	docker build --no-cache -f Dockerfile.routefix -t ${RP_IMAGE_ACR}.azurecr.io/routefix:$(COMMIT) .

publish-image-aro: image-aro
	docker push $(ARO_IMAGE)
ifeq ("${RP_IMAGE_ACR}-$(BRANCH)","arointsvc-master")
		docker tag $(ARO_IMAGE) arointsvc.azurecr.io/aro:latest
		docker push arointsvc.azurecr.io/aro:latest
endif

publish-image-fluentbit: image-fluentbit
	docker push ${RP_IMAGE_ACR}.azurecr.io/fluentbit:1.3.9-1

publish-image-proxy: image-proxy
	docker push ${RP_IMAGE_ACR}.azurecr.io/proxy:latest

publish-image-routefix: image-routefix
	docker push ${RP_IMAGE_ACR}.azurecr.io/routefix:$(COMMIT)

proxy:
	go build -ldflags "-X github.com/Azure/ARO-RP/pkg/util/version.GitCommit=$(COMMIT)" ./hack/proxy

pyenv:
	virtualenv pyenv
	. pyenv/bin/activate && \
		pip install autopep8 azdev azure-mgmt-loganalytics==0.2.0 ruamel.yaml wheel && \
		azdev setup -r . && \
		sed -i -e "s|^dev_sources = $(PWD)$$|dev_sources = $(PWD)/python|" ~/.azure/config

secrets:
	@[ "${SECRET_SA_ACCOUNT_NAME}" ] || ( echo ">> SECRET_SA_ACCOUNT_NAME is not set"; exit 1 )
	rm -rf secrets
	az storage blob download --auth-mode login -n secrets.tar.gz -c secrets -f secrets.tar.gz --account-name ${SECRET_SA_ACCOUNT_NAME} >/dev/null
	tar -xzf secrets.tar.gz
	rm secrets.tar.gz

secrets-update:
	@[ "${SECRET_SA_ACCOUNT_NAME}" ] || ( echo ">> SECRET_SA_ACCOUNT_NAME is not set"; exit 1 )
	tar -czf secrets.tar.gz secrets
	az storage blob upload -n secrets.tar.gz -c secrets -f secrets.tar.gz --account-name ${SECRET_SA_ACCOUNT_NAME} >/dev/null
	rm secrets.tar.gz

e2e.test:
	go test ./test/e2e -tags e2e -c -o e2e.test

test-e2e: e2e.test
	./e2e.test -test.timeout 180m -test.v -ginkgo.v

test-go: generate
	go build ./...

	gofmt -s -w cmd hack pkg test
	go run ./vendor/golang.org/x/tools/cmd/goimports -w -local=github.com/Azure/ARO-RP cmd hack pkg test
	go run ./hack/validate-imports cmd hack pkg test
	go run ./hack/licenses
	@[ -z "$$(ls pkg/util/*.go 2>/dev/null)" ] || (echo error: go files are not allowed in pkg/util, use a subpackage; exit 1)
	@[ -z "$$(find -name "*:*")" ] || (echo error: filenames with colons are not allowed on Windows, please rename; exit 1)
	@sha256sum --quiet -c .sha256sum || (echo error: client library is stale, please run make client; exit 1)
	go test -tags e2e -run ^$$ ./test/e2e/...

	go vet ./...
	set -o pipefail && go test -v ./... -coverprofile cover.out | tee uts.txt

lint-go: generate
	golangci-lint run

test-python: generate pyenv az
	. pyenv/bin/activate && \
		azdev linter && \
		azdev style && \
		hack/format-yaml/format-yaml.py .pipelines

admin.kubeconfig:
	hack/get-admin-kubeconfig.sh /subscriptions/${AZURE_SUBSCRIPTION_ID}/resourceGroups/${RESOURCEGROUP}/providers/Microsoft.RedHatOpenShift/openShiftClusters/${CLUSTER} >admin.kubeconfig

.PHONY: admin.kubeconfig aro az clean client generate image-aro image-fluentbit image-proxy image-routefix proxy publish-image-aro publish-image-fluentbit publish-image-proxy publish-image-routefix secrets secrets-update e2e.test test-e2e test-go test-python
