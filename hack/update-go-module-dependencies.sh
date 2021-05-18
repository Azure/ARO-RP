#!/bin/bash -ex

for x in vendor/github.com/openshift/*; do
	case $x in
		vendor/github.com/openshift/installer|vendor/github.com/openshift/cluster-api-provider-baremetal)
			;;
		vendor/github.com/openshift/cloud-credential-operator)
			go mod edit -replace ${x##vendor/}=$(go list -mod=mod -m ${x##vendor/}@release-4.5 | sed -e 's/ /@/')
			;;
		*)
			go mod edit -replace ${x##vendor/}=$(go list -mod=mod -m ${x##vendor/}@release-4.6 | sed -e 's/ /@/')
			;;
	esac
done

for x in aws azure gcp openstack; do
	go mod edit -replace sigs.k8s.io/cluster-api-provider-$x=$(go list -mod=mod -m github.com/openshift/cluster-api-provider-$x@release-4.6 | sed -e 's/ /@/')
done

go mod edit -replace github.com/metal3-io/baremetal-operator=$(go list -mod=mod -m github.com/openshift/baremetal-operator@release-4.6 | sed -e 's/ /@/')
go mod edit -replace github.com/metal3-io/cluster-api-provider-baremetal=$(go list -mod=mod -m github.com/openshift/cluster-api-provider-baremetal@release-4.6 | sed -e 's/ /@/')

go mod edit -replace github.com/openshift/installer=$(go list -mod=mod -m github.com/mjudeikis/installer@release-4.6-azure | sed -e 's/ /@/')

go mod edit -replace k8s.io/kube-openapi=$(go list -mod=mod -m k8s.io/kube-openapi@release-1.19 | sed -e 's/ /@/')

# We are still using Go 1.14, but >=v0.1.1 depends on standard library from Go 1.16
go mod edit -replace golang.org/x/tools=golang.org/x/tools@v0.1.0

go get -u ./...

go mod tidy
go mod vendor
