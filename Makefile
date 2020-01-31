COMMIT = $(shell git rev-parse --short HEAD)$(shell [[ $$(git status --porcelain) = "" ]] || echo -dirty)

aro: generate
	go build -ldflags "-X main.gitCommit=$(COMMIT)" ./cmd/aro

az:
	cd python/az/aro && python ./setup.py bdist_egg

clean:
	rm -rf python/az/aro/{aro.egg-info,build,dist} aro
	find python -type f -name '*.pyc' -delete
	find python -type d -name __pycache__ -delete

client: generate
	rm -rf pkg/client python/client
	mkdir pkg/client python/client
	sha256sum swagger/redhatopenshift/resource-manager/Microsoft.RedHatOpenShift/preview/2019-12-31-preview/redhatopenshift.json >.sha256sum

	sudo docker run \
		--rm \
		-v $(PWD)/pkg/client:/github.com/Azure/ARO-RP/pkg/client:z \
		-v $(PWD)/swagger:/swagger:z \
		azuresdk/autorest \
		--go \
		--license-header=MICROSOFT_APACHE_NO_VERSION \
		--namespace=redhatopenshift \
		--input-file=/swagger/redhatopenshift/resource-manager/Microsoft.RedHatOpenShift/preview/2019-12-31-preview/redhatopenshift.json \
		--output-folder=/github.com/Azure/ARO-RP/pkg/client/services/preview/redhatopenshift/mgmt/2019-12-31-preview/redhatopenshift

	sudo docker run \
		--rm \
		-v $(PWD)/python/client:/python/client:z \
		-v $(PWD)/swagger:/swagger:z \
		azuresdk/autorest \
		--use=@microsoft.azure/autorest.python@4.0.70 \
		--python \
		--azure-arm \
		--license-header=MICROSOFT_APACHE_NO_VERSION \
		--namespace=azure.mgmt.redhatopenshift.v2019_12_31_preview \
		--input-file=/swagger/redhatopenshift/resource-manager/Microsoft.RedHatOpenShift/preview/2019-12-31-preview/redhatopenshift.json \
		--output-folder=/python/client

	sudo chown -R $$(id -un):$$(id -gn) pkg/client python/client
	sed -i -e 's|azure/aro-rp|Azure/ARO-RP|g' pkg/client/services/preview/redhatopenshift/mgmt/2019-12-31-preview/redhatopenshift/models.go pkg/client/services/preview/redhatopenshift/mgmt/2019-12-31-preview/redhatopenshift/redhatopenshiftapi/interfaces.go
	rm -rf python/client/azure/mgmt/redhatopenshift/v2019_12_31_preview/aio
	>python/client/__init__.py

	go run ./vendor/golang.org/x/tools/cmd/goimports -w -local=github.com/Azure/ARO-RP pkg/client

generate:
	go generate ./...

image-aro: aro
	docker pull registry.access.redhat.com/ubi8/ubi-minimal
	docker build -f Dockerfile.aro -t arosvc.azurecr.io/aro:$(COMMIT) .

image-mdm:
	docker build --build-arg VERSION=2.2019.801.1228-66cac1-~bionic_amd64 \
	  -f Dockerfile.mdm -t arosvc.azurecr.io/mdm:2019.801.1228-66cac1 .

image-proxy: proxy
	docker pull registry.access.redhat.com/ubi8/ubi-minimal
	docker build -f Dockerfile.proxy -t arosvc.azurecr.io/proxy:latest .

proxy:
	go build -ldflags "-X main.gitCommit=$(COMMIT)" ./hack/proxy

pyenv${PYTHON_VERSION}:
	virtualenv --python=/usr/bin/python${PYTHON_VERSION} pyenv${PYTHON_VERSION}
	. pyenv${PYTHON_VERSION}/bin/activate && \
		pip install azdev && \
		azdev setup -r . && \
		sed -i -e "s|^dev_sources = $(PWD)$$|dev_sources = $(PWD)/python|" ~/.azure/config

secrets:
	rm -rf secrets
	mkdir secrets
	oc extract -n azure secret/aro-v4-dev --to=secrets

secrets-update:
	oc create secret generic aro-v4-dev --from-file=secrets --dry-run -o yaml | oc apply -f -

e2e:
	go test ./test/e2e -timeout 60m -v -ginkgo.v -tags e2e

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
	go test -v ./... -coverprofile cover.out | tee uts.txt

test-python: generate pyenv${PYTHON_VERSION}
	. pyenv${PYTHON_VERSION}/bin/activate && \
		$(MAKE) az && \
		azdev linter && \
		azdev style

.PHONY: aro az clean client generate image-aro proxy secrets secrets-update test-go test-python
