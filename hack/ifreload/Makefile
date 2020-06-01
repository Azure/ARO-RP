COMMIT = $(shell git rev-parse --short HEAD)$(shell [[ $$(git status --porcelain) = "" ]] || echo -dirty)

CFLAGS=-I/usr/include/libnl3 -O3 -Wall -W
LDLIBS=-lnl-3 -lnl-route-3 -s

ifreload:

ifreload.rhel8: tools
	docker build --no-cache -v $(PWD):/build -f Dockerfile.build .
	docker build -f Dockerfile.ifreload -t ${RP_IMAGE_ACR}.azurecr.io/ifreload:$(COMMIT) .

tools:
	docker build -f Dockerfile.tools -t tools .

clean:
	rm -f ifreload

.PHONY: ifreload.rhel8 tools clean
