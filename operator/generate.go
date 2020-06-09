package operator

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

// build the kubenetes client
//go:generate go run ../vendor/k8s.io/code-generator/cmd/client-gen --clientset-name versioned --input-base github.com/Azure/ARO-RP --input operator/apis/aro.openshift.io/v1alpha1 --output-package github.com/Azure/ARO-RP/pkg/util/aro-operator-client/clientset --go-header-file ../hack/licenses/boilerplate.go.txt
//go:generate gofmt -s -w ../pkg/util/aro-operator-client
//go:generate go run ../vendor/golang.org/x/tools/cmd/goimports -w -local=github.com/Azure/ARO-RP ../pkg/util/aro-operator-client

// build the operator's CRD (based on the apis) and rbac (based on in-code tags - search for "+kubebuilder:rbac")
// for master deployment
//go:generate go run ../vendor/sigs.k8s.io/controller-tools/cmd/controller-gen "crd:trivialVersions=true" rbac:roleName=aro-operator-master paths="../pkg/controllers/..." paths="./apis/..." output:crd:dir=deploy/staticresources output:rbac:dir=deploy/staticresources/master
// for worker deployment - less privledges as it only runs the internetchecker
//go:generate go run ../vendor/sigs.k8s.io/controller-tools/cmd/controller-gen rbac:roleName=aro-operator-worker paths="../pkg/controllers/internetchecker_controller.go" output:dir=deploy/staticresources/worker

// bindata for the above yaml files
//go:generate go run ../vendor/github.com/go-bindata/go-bindata/go-bindata -nometadata -pkg deploy -prefix deploy/staticresources/ -o deploy/bindata.go deploy/staticresources/...
//go:generate gofmt -s -l -w deploy/bindata.go
