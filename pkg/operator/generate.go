package operator

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

// build the Kubernetes objects
//go:generate go run ../../vendor/sigs.k8s.io/controller-tools/cmd/controller-gen object paths=./apis/...
//go:generate go run ../../vendor/golang.org/x/tools/cmd/goimports -local=github.com/Azure/ARO-RP -e -w ./apis

// build the operator's CRD (based on the apis)
// for master deployment
//go:generate go run ../../vendor/sigs.k8s.io/controller-tools/cmd/controller-gen crd paths="./apis/..." output:crd:dir=deploy/staticresources
// for worker deployment - less privileges as it only runs the internetchecker
// rbac (based on in-code tags - search for "+kubebuilder:rbac")
//go:generate go run ../../vendor/sigs.k8s.io/controller-tools/cmd/controller-gen rbac:roleName=aro-operator-worker paths="./controllers/checkers/internetchecker/..." output:dir=deploy/staticresources/worker
