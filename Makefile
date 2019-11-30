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

.PHONY: rp clean image test
