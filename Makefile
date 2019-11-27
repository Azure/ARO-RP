rp:
	go build -ldflags "-X main.gitCommit=$(shell git rev-parse --short HEAD)$(shell [[ $$(git status --porcelain) = "" ]] || echo -dirty)" ./cmd/rp

clean:
	rm -f rp

generate:
	go generate ./...

image:
	go get github.com/openshift/imagebuilder/cmd/imagebuilder
	imagebuilder -f Dockerfile -t rp:latest .

test: generate
	go vet ./...
	./hack/verify/validate-code-format.sh
	./hack/verify/validate-util.sh
	go run ./hack/validate-imports/validate-imports.go cmd hack pkg
	go test ./...

.PHONY: clean generate image rp test
