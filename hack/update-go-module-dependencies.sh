#!/bin/bash -ex

for x in vendor/github.com/openshift/*; do
	if [[ "$x" = "vendor/github.com/openshift/installer" ]]; then
		continue
	fi
	go mod edit -replace ${x##vendor/}=${x##vendor/}@$(curl https://proxy.golang.org/${x##vendor/}/@v/release-4.6.info | jq -r ."Version")
done

for x in aws azure; do
	go mod edit -replace sigs.k8s.io/cluster-api-provider-$x=github.com/openshift/cluster-api-provider-$x@$(curl https://proxy.golang.org/github.com/openshift/cluster-api-provider-$x/@v/release-4.6.info | jq -r ."Version")
done

go mod edit -replace github.com/openshift/installer=github.com/jim-minter/installer@$(curl https://proxy.golang.org/github.com/jim-minter/installer/@v/release-4.6-azure.info | jq -r ."Version")

go mod edit -replace k8s.io/kube-openapi=k8s.io/kube-openapi@$(curl https://proxy.golang.org/k8s.io/kube-openapi/@v/release-1.19.info | jq -r ."Version")

go get -u ./...

go mod tidy
go mod vendor
