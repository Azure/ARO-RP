COMMIT = $(shell git rev-parse --short HEAD)$(shell [[ $$(git status --porcelain) = "" ]] || echo -dirty)

rp:
	go build -ldflags "-X main.gitCommit=$(COMMIT)" ./cmd/rp

clean:
	rm -f rp

image: rp
	docker build -t rp:$(COMMIT) .

test:
	go generate ./...
	go build ./...

	gofmt -s -w cmd hack pkg
	go run ./vendor/golang.org/x/tools/cmd/goimports -w -local=github.com/jim-minter/rp cmd hack pkg
	go run ./hack/validate-imports/validate-imports.go cmd hack pkg
	@[ -z "$$(ls pkg/util/*.go 2>/dev/null)" ] || (echo error: go files are not allowed in pkg/util, use a subpackage; exit 1)
	@[ -z "$$(find -name "*:*")" ] || (echo error: filenames with colons are not allowed on Windows, please rename; exit 1)

	go vet ./...
	go test ./...

generate-sdk:
	go generate ./...
	# Due to orering https://github.com/Azure/autorest.go/blob/3f9bdd60fd7a7c8740bbe3ec986f583f7ffa5fbd/src/Model/CodeModelGo.cs#L120-L141
	# package FQDN are either hardcoded to azure sdk or set wrong. For this we do
	# replace in the end
	podman run --privileged --workdir=/go/src -it -v $(GOPATH):/go --entrypoint autorest \
    azuresdk/autorest ./github.com/jim-minter/rp/rest-api-spec/redhatopenshift/resource-manager/readme.md \
    --go --go-sdks-folder=./github.com/jim-minter/rp/pkg/sdk --multiapi \
    --use=@microsoft.azure/autorest.go@~2.1.137 --use-onever --verbose
	# HACK to align pkg imports when running in non default repository
	find . -type f -name "*.go" -exec sed -i 's/go\/src\/github.com/github.com/g' {} +

.PHONY: clean generate-sdk image rp test

