SHELL = /bin/bash
COMMIT = $(shell git rev-parse --short HEAD)$(shell [[ $$(git status --porcelain) = "" ]] || echo -dirty)
ARO_IMAGE_TAG ?= $(COMMIT)
ARO_IMAGE ?= ${RP_IMAGE_ACR}.azurecr.io/aro:$(ARO_IMAGE_TAG)

aro: generate
	go build -ldflags "-X github.com/Azure/ARO-RP/pkg/util/version.GitCommit=$(COMMIT)" ./cmd/aro

az:
	cd python/az/aro && python ./setup.py bdist_egg
	cd python/az/aro && python ./setup.py bdist_wheel || true

clean:
	rm -rf python/az/aro/{aro.egg-info,build,dist} aro
	find python -type f -name '*.pyc' -delete
	find python -type d -name __pycache__ -delete

client: generate
	rm -rf pkg/client python/client
	mkdir pkg/client python/client
	sha256sum swagger/redhatopenshift/resource-manager/Microsoft.RedHatOpenShift/stable/2020-04-30/redhatopenshift.json >.sha256sum

	sudo docker run \
		--rm \
		-v $(PWD)/pkg/client:/github.com/Azure/ARO-RP/pkg/client:z \
		-v $(PWD)/swagger:/swagger:z \
		azuresdk/autorest \
		--go \
		--license-header=MICROSOFT_APACHE_NO_VERSION \
		--namespace=redhatopenshift \
		--input-file=/swagger/redhatopenshift/resource-manager/Microsoft.RedHatOpenShift/stable/2020-04-30/redhatopenshift.json \
		--output-folder=/github.com/Azure/ARO-RP/pkg/client/services/redhatopenshift/mgmt/2020-04-30/redhatopenshift

	sudo docker run \
		--rm \
		-v $(PWD)/python/client:/python/client:z \
		-v $(PWD)/swagger:/swagger:z \
		azuresdk/autorest \
		--use=@microsoft.azure/autorest.python@4.0.70 \
		--python \
		--azure-arm \
		--license-header=MICROSOFT_APACHE_NO_VERSION \
		--namespace=azure.mgmt.redhatopenshift.v2020_04_30 \
		--input-file=/swagger/redhatopenshift/resource-manager/Microsoft.RedHatOpenShift/stable/2020-04-30/redhatopenshift.json \
		--output-folder=/python/client

	sudo chown -R $$(id -un):$$(id -gn) pkg/client python/client
	sed -i -e 's|azure/aro-rp|Azure/ARO-RP|g' pkg/client/services/redhatopenshift/mgmt/2020-04-30/redhatopenshift/models.go pkg/client/services/redhatopenshift/mgmt/2020-04-30/redhatopenshift/redhatopenshiftapi/interfaces.go
	rm -rf python/client/azure/mgmt/redhatopenshift/v2020_04_30/aio
	>python/client/__init__.py

	go run ./vendor/golang.org/x/tools/cmd/goimports -w -local=github.com/Azure/ARO-RP pkg/client

generate:
	go generate ./...

image-aro: aro
	docker pull registry.access.redhat.com/ubi8/ubi-minimal
	docker build -f Dockerfile.aro -t $(ARO_IMAGE) .

image-fluentbit:
	docker build --build-arg VERSION=1.3.9-1 \
	  -f Dockerfile.fluentbit -t ${RP_IMAGE_ACR}.azurecr.io/fluentbit:1.3.9-1 .

image-ifreload:
	cd hack/ifreload && $(MAKE) clean ifreload.rhel8

image-proxy: proxy
	docker pull registry.access.redhat.com/ubi8/ubi-minimal
	docker build -f Dockerfile.proxy -t ${RP_IMAGE_ACR}.azurecr.io/proxy:latest .

get-aro-image:
	@echo $(ARO_IMAGE)

publish-image-aro: image-aro
	docker push $(ARO_IMAGE)

publish-image-fluentbit: image-fluentbit
	docker push ${RP_IMAGE_ACR}.azurecr.io/fluentbit:1.3.9-1

publish-image-ifreload: image-ifreload
	docker push ${RP_IMAGE_ACR}.azurecr.io/ifreload:$(COMMIT)

publish-image-proxy: image-proxy
	docker push ${RP_IMAGE_ACR}.azurecr.io/proxy:latest

proxy:
	go build -ldflags "-X github.com/Azure/ARO-RP/pkg/util/version.GitCommit=$(COMMIT)" ./hack/proxy

pyenv${PYTHON_VERSION}:
	virtualenv --python=/usr/bin/python${PYTHON_VERSION} pyenv${PYTHON_VERSION}
	. pyenv${PYTHON_VERSION}/bin/activate && \
		pip install autopep8 azdev azure-mgmt-loganalytics==0.2.0 ruamel.yaml && \
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
	az storage blob upload --auth-mode login -n secrets.tar.gz -c secrets -f secrets.tar.gz --account-name ${SECRET_SA_ACCOUNT_NAME} >/dev/null
	rm secrets.tar.gz

e2e:
	go test ./test/e2e -timeout 60m -v -ginkgo.v -tags e2e

test-go: generate
	go build ./...

	gofmt -s -w cmd hack pkg test operator
	go run ./vendor/golang.org/x/tools/cmd/goimports -w -local=github.com/Azure/ARO-RP cmd hack pkg test operator
	go run ./hack/validate-imports cmd hack pkg test operator
	go run ./hack/licenses
	@[ -z "$$(ls pkg/util/*.go 2>/dev/null)" ] || (echo error: go files are not allowed in pkg/util, use a subpackage; exit 1)
	@[ -z "$$(find -name "*:*")" ] || (echo error: filenames with colons are not allowed on Windows, please rename; exit 1)
	@sha256sum --quiet -c .sha256sum || (echo error: client library is stale, please run make client; exit 1)
	go test -tags e2e -run ^$$ ./test/e2e/...

	go vet ./...
	set -o pipefail && go test -v ./... -coverprofile cover.out | tee uts.txt

test-python: generate pyenv${PYTHON_VERSION}
	. pyenv${PYTHON_VERSION}/bin/activate && \
		$(MAKE) az && \
		azdev linter && \
		azdev style && \
		hack/format-yaml/format-yaml.py .pipelines

admin.kubeconfig:
	hack/get-admin-kubeconfig.sh /subscriptions/${AZURE_SUBSCRIPTION_ID}/resourceGroups/${RESOURCEGROUP}/providers/Microsoft.RedHatOpenShift/openShiftClusters/${CLUSTER} >admin.kubeconfig

.PHONY: aro az clean client generate image-aro image-ifreload proxy secrets secrets-update test-go test-python image-fluentbit publish-image-proxy publish-image-ifreload publish-image-aro publish-image-fluentbit publish-image-proxy admin.kubeconfig
