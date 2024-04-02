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

for x in vendor/github.com/openshift/*; do
	case $x in
		# Review the list of special cases on each release.

		# Do not update Hive: it is not part of OCP
		vendor/github.com/openshift/hive)
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

go get -u ./...
