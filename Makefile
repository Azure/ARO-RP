include Makefile.preflight

SHELL = /bin/bash
COMMIT = $(shell git rev-parse --short HEAD)$(shell [[ $$(git status --porcelain) = "" ]] || echo -dirty)
RP_IMAGE_ACR="arosvc"

aro: generate
	go build -ldflags "-X main.gitCommit=$(COMMIT)" ./cmd/aro

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
	docker build -f Dockerfile.aro -t ${RP_IMAGE_ACR}.azurecr.io/aro:$(COMMIT) .

image-fluentbit:
	docker build --build-arg VERSION=1.3.9-1 \
	  -f Dockerfile.fluentbit -t ${RP_IMAGE_ACR}.azurecr.io/fluentbit:1.3.9-1 .

image-proxy: proxy
	docker pull registry.access.redhat.com/ubi8/ubi-minimal
	docker build -f Dockerfile.proxy -t ${RP_IMAGE_ACR}.azurecr.io/proxy:latest .

publish-image-aro: image-aro
	docker push ${RP_IMAGE_ACR}.azurecr.io/aro:$(COMMIT)

publish-image-fluentbit: image-fluentbit
	docker push ${RP_IMAGE_ACR}.azurecr.io/fluentbit:1.3.9-1

publish-image-proxy: image-proxy
	docker push ${RP_IMAGE_ACR}.azurecr.io/proxy:latest

proxy:
	go build -ldflags "-X main.gitCommit=$(COMMIT)" ./hack/proxy

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

test-python: generate pyenv${PYTHON_VERSION}
	. pyenv${PYTHON_VERSION}/bin/activate && \
		$(MAKE) az && \
		azdev linter && \
		azdev style && \
		hack/format-yaml/format-yaml.py .pipelines

admin.kubeconfig:
	hack/get-admin-kubeconfig.sh /subscriptions/${AZURE_SUBSCRIPTION_ID}/resourceGroups/${RESOURCEGROUP}/providers/Microsoft.RedHatOpenShift/openShiftClusters/${CLUSTER} >admin.kubeconfig

.PHONY: aro az clean client generate image-aro proxy secrets secrets-update test-go test-python image-fluentbit publish-image-proxy publish-image-aro publish-image-fluentbit publish-image-proxy admin.kubeconfig

# Image URL to use all building/pushing image targets
ARO_OPERATOR_IMG ?= ${RP_IMAGE_ACR}.azurecr.io/aro-operator:$(COMMIT)
ARO_OPERATOR_DIRS=pkg/util/aro-operator-client operator cmd/operator pkg/controllers

image-operator: operator
	docker pull registry.access.redhat.com/ubi8/ubi-minimal
	docker build -f Dockerfile.operator -t $(ARO_OPERATOR_IMG) .

publish-image-operator: image-operator
	docker push $(ARO_OPERATOR_IMG)

operator-generate:
	go run ./vendor/k8s.io/code-generator/cmd/deepcopy-gen --input-dirs github.com/Azure/ARO-RP/operator/apis/aro.openshift.io/v1alpha1 -O zz_generated.deepcopy --bounding-dirs github.com/Azure/ARO-RP/operator/apis --go-header-file hack/licenses/boilerplate.go.txt
	go run ./vendor/k8s.io/code-generator/cmd/client-gen --clientset-name versioned --input-base '' --input github.com/Azure/ARO-RP/operator/apis/aro.openshift.io/v1alpha1 --output-package github.com/Azure/ARO-RP/pkg/util/aro-operator-client/clientset --go-header-file hack/licenses/boilerplate.go.txt
	gofmt -s -w $(ARO_OPERATOR_DIRS)
	go run ./vendor/golang.org/x/tools/cmd/goimports -w -local=github.com/Azure/ARO-RP $(ARO_OPERATOR_DIRS)
	go run ./hack/validate-imports $(ARO_OPERATOR_DIRS)
	pushd operator ; go run ../hack/licenses ; popd

# Generate manifests e.g. CRD, RBAC etc.
CRD_OPTIONS ?= "crd:trivialVersions=true"
operator-manifests: controller-gen kustomize
	$(CONTROLLER_GEN) $(CRD_OPTIONS) rbac:roleName=manager-role webhook paths="./operator/..." output:crd:artifacts:config=operator/config/crd/bases
	$(KUSTOMIZE) build operator/config/crd > operator/config/output/crd.yaml
	cd operator/config/manager && $(KUSTOMIZE) edit set image controller=${ARO_OPERATOR_IMG}
	$(KUSTOMIZE) build operator/config/default > operator/config/output/resources.yaml

operator-run: operator-generate operator-manifests
	go run ./cmd/operator/main.go

operator: operator-generate
	go build -o aro-operator ./cmd/operator/main.go

# Install CRDs into a cluster
operator-install: operator-manifests
	kubectl apply -f operator/config/output/crd.yaml

# Uninstall CRDs from a cluster
operator-uninstall: operator-manifests
	kubectl delete -f operator/config/output/crd.yaml

# Deploy controller in the configured Kubernetes cluster in ~/.kube/config
operator-deploy: operator-manifests
	kubectl apply -f operator/config/output/resources.yaml

.PHONY: operator operator-generate operator-manifests operator-install operator-uninstall operator-run operator-deploy
