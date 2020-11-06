#!/bin/bash -ex

for x in vendor/github.com/openshift/*; do
	if [[ "$x" = "vendor/github.com/openshift/installer" ]]; then
		continue
	fi
	go mod edit -replace ${x##vendor/}=$(go list -mod=mod -m ${x##vendor/}@release-4.5 | sed -e 's/ /@/')
done

for x in aws azure gcp openstack; do
	go mod edit -replace sigs.k8s.io/cluster-api-provider-$x=$(go list -mod=mod -m github.com/openshift/cluster-api-provider-$x@release-4.5 | sed -e 's/ /@/')
done

go mod edit -replace github.com/metal3-io/baremetal-operator=$(go list -mod=mod -m github.com/openshift/baremetal-operator@release-4.5 | sed -e 's/ /@/')

go mod edit -replace github.com/openshift/installer=$(go list -mod=mod -m github.com/jim-minter/installer@release-4.5-azure | sed -e 's/ /@/')

go mod edit -replace k8s.io/kube-openapi=$(go list -mod=mod -m k8s.io/kube-openapi@release-1.18 | sed -e 's/ /@/')

go get -u ./...

go mod tidy
go mod vendor
