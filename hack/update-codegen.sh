#!/bin/bash -e

# https://www.openshift.com/blog/kubernetes-deep-dive-code-generation-customresources
#
# note this requires:
#
# go get k8s.io/code-generator/cmd/...
# cd $GOPATH/src/k8s.io/code-generator
# git checkout -b kubernetes-1.16.0-beta.0 kubernetes-1.16.0-beta.0

$GOPATH/src/k8s.io/code-generator/generate-groups.sh deepcopy,client \
github.com/Azure/ARO-RP/pkg/util/aro-operator-client \
github.com/Azure/ARO-RP/operator/apis aro.openshift.io:v1alpha1

