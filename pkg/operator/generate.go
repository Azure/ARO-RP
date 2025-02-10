package operator

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

// build the Kubernetes objects
//go:generate controller-gen object paths=./apis/...

// build the operator's CRD (based on the apis)
// for master deployment
//go:generate controller-gen crd paths="./apis/..." output:crd:dir=deploy/staticresources
// for worker deployment - less privileges as it only runs the internetchecker
// rbac (based on in-code tags - search for "+kubebuilder:rbac")
//go:generate controller-gen rbac:roleName=aro-operator-worker paths="./controllers/checkers/internetchecker/..." output:dir=deploy/staticresources/worker
