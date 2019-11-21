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
	go test ./...

.PHONY: clean generate image rp test
