package operator

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

// build the kubenetes client
//go:generate go run ../../vendor/sigs.k8s.io/controller-tools/cmd/controller-gen object paths=./apis/...
//go:generate go run ../../vendor/k8s.io/code-generator/cmd/client-gen --clientset-name versioned --input-base ./apis --input aro.openshift.io/v1alpha1,preview.aro.openshift.io/v1alpha1 --output-package ./clientset --go-header-file ../../hack/licenses/boilerplate.go.txt
//go:generate gofmt -s -w ./clientset
//go:generate go run ../../vendor/golang.org/x/tools/cmd/goimports -local=github.com/Azure/ARO-RP -e -w ./clientset ./apis

// build the operator's CRD (based on the apis)
// for master deployment
//go:generate go run ../../vendor/sigs.k8s.io/controller-tools/cmd/controller-gen "crd:trivialVersions=true" paths="./apis/..." output:crd:dir=deploy/staticresources
// for worker deployment - less privileges as it only runs the internetchecker
// rbac (based on in-code tags - search for "+kubebuilder:rbac")
//go:generate go run ../../vendor/sigs.k8s.io/controller-tools/cmd/controller-gen rbac:roleName=aro-operator-worker paths="controllers/internetchecker/..." output:dir=deploy/staticresources/worker

// bindata for the above yaml files
//go:generate go run ../../vendor/github.com/go-bindata/go-bindata/go-bindata -nometadata -pkg deploy -prefix deploy/staticresources/ -o deploy/bindata.go deploy/staticresources/...
//go:generate gofmt -s -l -w deploy/bindata.go
