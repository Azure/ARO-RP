package RP

// this file is purely run generate commands
// that need to be run from the top level of the project.

// build the operator's rbac based on in-code tags (search for "+kubebuilder:rbac")
//go:generate go run ./vendor/sigs.k8s.io/controller-tools/cmd/controller-gen "crd:trivialVersions=true" rbac:roleName=manager-role webhook paths="./pkg/controllers/..." output:crd:artifacts:config=operator/config/crd/bases output:dir=operator/config/resources
// build the operator's static resources
//go_generate go run ./vendor/sigs.k8s.io/kustomize/kustomize build operator/config/default > operator/deploy/resources.yaml
// build the operator's custom client (doesn't seem to work within a generate header)
//go:generate go run ./vendor/k8s.io/code-generator/cmd/deepcopy-gen --input-dirs github.com/Azure/ARO-RP/operator/apis/aro.openshift.io/v1alpha1 -O zz_generated.deepcopy --bounding-dirs github.com/Azure/ARO-RP/operator/apis --go-header-file hack/licenses/boilerplate.go.txt
//go:generate go run ./vendor/k8s.io/code-generator/cmd/client-gen --clientset-name versioned --input-base github.com/Azure/ARO-RP --input operator/apis/aro.openshift.io/v1alpha1 --output-package github.com/Azure/ARO-RP/pkg/util/aro-operator-client/clientset --go-header-file hack/licenses/boilerplate.go.txt
//go:generate gofmt -s -w pkg/util/aro-operator-client
//go:generate go run ./vendor/golang.org/x/tools/cmd/goimports -w -local=github.com/Azure/ARO-RP pkg/util/aro-operator-client
