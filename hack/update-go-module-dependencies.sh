#!/bin/bash -ex

# Background: https://groups.google.com/forum/#!topic/golang-nuts/51-D_YFC78k
#
# TLDR: OCP consists of many repos where for each release a new release branch gets created (release-X.Y).
# When we update vendors we want to get latest changes from the release branch for all of the dependencies.
# With Go modules we can't easily do it, but there is a workaround which consists of multiple steps:
# 	1. Get the latest commit from the branch using `go list -mod=mod -m MODULE@release-x.y`.
# 	2. Using `sed`, transform output of the above command into format accepted by `go mod edit -replace`.
#	3. Modify `go.mod` by calling `go mod edit -replace`.
#
# This needs to happen for each module that uses this branching strategy: all these repos
# live under github.com/openshift organisation.
#
# There are however, some exceptions:
# 	* Some repos under github.com/openshift do not use this strategy.
#     We should skip them in this script and manage directly with `go mod`.
# 	* Some dependencies pin their own dependencies to older commits.
#     For example, dependency Foo from release-4.7 branch requires
#	  dependency Bar at older commit which is
#     not compatible with Bar@release-4.7.
#
# Note that github.com/openshift org also contains forks of K8s upstream repos and we
# use these forks (indirectly in most cases). This means that
# we also must take care of replacing modules such as sigs.k8s.io/cluster-api-provider-azure
# with github.com/openshift/cluster-api-provider-azure (just an example, there are more).

for x in vendor/github.com/openshift/*; do
	case $x in
		# Review the list of special cases on each release.

		# Do not update Hive: it is not part of OCP
		vendor/github.com/openshift/hive)
			;;

		# Replace the installer with our own fork below in this script.
		vendor/github.com/openshift/installer)
			;;

		# Inconsistent imports: some of our dependencies import it as github.com/metal3-io/cluster-api-provider-baremetal
		# but in some places directly from the openshift fork.
		# Replace github.com/metal3-io/cluster-api-provider-baremetal with an openshift fork in go.mod
		vendor/github.com/openshift/cluster-api-provider-baremetal)
			;;

		# It is only used indirectly and intermediate dependencies pin to different incompatible commits.
		# We force a specific commit here to make all dependencies happy.
		vendor/github.com/openshift/cloud-credential-operator)
			go mod edit -replace github.com/openshift/cloud-credential-operator=github.com/openshift/cloud-credential-operator@v0.0.0-20200316201045-d10080b52c9e
			;;

		# This repo doesn't follow release-x.y branching strategy
		# We skip it and let go mod resolve it
		vendor/github.com/openshift/custom-resource-status)
			;;

		*)
			go mod edit -replace ${x##vendor/}=$(go list -mod=mod -m ${x##vendor/}@release-4.10 | sed -e 's/ /@/')
			;;
	esac
done

for x in aws azure openstack; do
	go mod edit -replace sigs.k8s.io/cluster-api-provider-$x=$(go list -mod=mod -m github.com/openshift/cluster-api-provider-$x@release-4.10 | sed -e 's/ /@/')
done

go mod edit -replace github.com/openshift/installer=$(go list -mod=mod -m github.com/jewzaam/installer-aro@release-4.10-azure | sed -e 's/ /@/')

go get -u ./...

go mod tidy -compat=1.18
go mod vendor
