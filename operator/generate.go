package operator

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

//go:generate go run ../vendor/k8s.io/code-generator/cmd/client-gen --clientset-name versioned --input-base github.com/Azure/ARO-RP --input operator/apis/aro.openshift.io/v1alpha1 --output-package github.com/Azure/ARO-RP/pkg/util/aro-operator-client/clientset --go-header-file ../hack/licenses/boilerplate.go.txt
//go:generate gofmt -s -w ../pkg/util/aro-operator-client
//go:generate go run ../vendor/golang.org/x/tools/cmd/goimports -w -local=github.com/Azure/ARO-RP ../pkg/util/aro-operator-client

// build the operator's rbac based on in-code tags (search for "+kubebuilder:rbac")
//go:generate go run ../vendor/sigs.k8s.io/controller-tools/cmd/controller-gen "crd:trivialVersions=true" rbac:roleName=manager-role paths="../pkg/controllers/..." paths="apis/..."  output:dir=deploy/staticresources
//go:generate go get github.com/go-bindata/go-bindata/go-bindata
//go:generate go-bindata -nometadata -pkg deploy -prefix deploy/staticresources/ -o deploy/bindata.go deploy/staticresources/
//go:generate gofmt -s -l -w deploy/bindata.go
