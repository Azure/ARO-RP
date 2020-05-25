package deploy

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

// build the operator's rbac based on in-code tags (search for "+kubebuilder:rbac")
//go:generate go run ../../vendor/sigs.k8s.io/controller-tools/cmd/controller-gen "crd:trivialVersions=true" rbac:roleName=manager-role paths="../../pkg/controllers/..." paths="../apis/..."  output:dir=staticresources
//go:generate go get github.com/go-bindata/go-bindata/go-bindata
//go:generate go-bindata -nometadata -pkg $GOPACKAGE staticresources/
//go:generate gofmt -s -l -w bindata.go
