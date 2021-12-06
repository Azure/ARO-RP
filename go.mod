module github.com/Azure/ARO-RP

go 1.14

require (
	cloud.google.com/go v0.94.1 // indirect
	github.com/AlecAivazis/survey/v2 v2.3.1 // indirect
	github.com/AlekSi/gocov-xml v0.0.0-20190121064608-3a14fb1c4737
	github.com/Azure/azure-sdk-for-go v57.1.0+incompatible
	github.com/Azure/go-autorest/autorest v0.11.20
	github.com/Azure/go-autorest/autorest/adal v0.9.15
	github.com/Azure/go-autorest/autorest/azure/auth v0.5.8
	github.com/Azure/go-autorest/autorest/azure/cli v0.4.3 // indirect
	github.com/Azure/go-autorest/autorest/date v0.3.0
	github.com/Azure/go-autorest/autorest/to v0.4.0
	github.com/Azure/go-autorest/autorest/validation v0.3.1
	github.com/Azure/go-autorest/tracing v0.6.0
	github.com/alvaroloes/enumer v1.1.2
	github.com/apparentlymart/go-cidr v1.1.0
	github.com/aws/aws-sdk-go v1.40.37 // indirect
	github.com/axw/gocov v1.0.0
	github.com/cespare/xxhash/v2 v2.1.2 // indirect
	github.com/clarketm/json v1.15.7 // indirect
	github.com/codahale/etm v0.0.0-20141003032925-c00c9e6fb4c9
	github.com/containers/image/v5 v5.16.0
	github.com/containers/libtrust v0.0.0-20200511145503-9c3a6c22cd9a // indirect
	github.com/coreos/go-oidc v2.2.1+incompatible
	github.com/coreos/go-systemd v0.0.0-20191104093116-d3cd4ed1dbcf // indirect
	github.com/coreos/go-systemd/v22 v22.3.2
	github.com/coreos/ignition/v2 v2.12.0
	github.com/coreos/stream-metadata-go v0.1.1
	github.com/coreos/vcontext v0.0.0-20210903173952-c22998be8e20 // indirect
	github.com/davecgh/go-spew v1.1.1
	github.com/emicklei/go-restful v2.15.0+incompatible // indirect
	github.com/exponent-io/jsonpath v0.0.0-20210407135951-1de76d718b3f // indirect
	github.com/form3tech-oss/jwt-go v3.2.5+incompatible
	github.com/fsnotify/fsnotify v1.5.1 // indirect
	github.com/ghodss/yaml v1.0.1-0.20190212211648-25d852aebe32
	github.com/go-bindata/go-bindata v3.1.2+incompatible
	github.com/go-errors/errors v1.4.0 // indirect
	github.com/go-logr/logr v1.1.0
	github.com/go-openapi/jsonpointer v0.19.5 // indirect
	github.com/go-openapi/jsonreference v0.19.6 // indirect
	github.com/go-openapi/spec v0.20.3 // indirect
	github.com/go-openapi/swag v0.19.15 // indirect
	github.com/go-playground/validator/v10 v10.9.0 // indirect
	github.com/go-test/deep v1.0.7
	github.com/gofrs/uuid v4.0.0+incompatible
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da // indirect
	github.com/golang/mock v1.6.0
	github.com/golangci/golangci-lint v1.32.2
	github.com/google/btree v1.0.1 // indirect
	github.com/google/go-cmp v0.5.6
	github.com/google/uuid v1.3.0 // indirect
	github.com/googleapis/gnostic v0.5.5
	github.com/gophercloud/utils v0.0.0-20210823151123-bfd010397530 // indirect
	github.com/gorilla/csrf v1.7.1
	github.com/gorilla/mux v1.8.0
	github.com/gorilla/securecookie v1.1.1
	github.com/gorilla/sessions v1.2.1
	github.com/h2non/filetype v1.1.1 // indirect
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/jewzaam/go-cosmosdb v0.0.0-20211018134417-8d1494e7862f
	github.com/jstemmer/go-junit-report v0.9.1
	github.com/k8snetworkplumbingwg/network-attachment-definition-client v1.1.0 // indirect
	github.com/klauspost/compress v1.13.5 // indirect
	github.com/libvirt/libvirt-go v7.4.0+incompatible // indirect
	github.com/mailru/easyjson v0.7.7 // indirect
	github.com/mattn/go-isatty v0.0.13 // indirect
	github.com/metal3-io/baremetal-operator v0.0.0-20210903180935-ae74cdcb3142 // indirect
	github.com/metal3-io/cluster-api-provider-baremetal v0.2.2 // indirect
	github.com/mgutz/ansi v0.0.0-20200706080929-d51e80ef957d // indirect
	github.com/mitchellh/go-wordwrap v1.0.1 // indirect
	github.com/moby/term v0.0.0-20210619224110-3f7ff695adc6 // indirect
	github.com/onsi/ginkgo v1.16.4
	github.com/onsi/gomega v1.16.0
	github.com/openshift/api v3.9.1-0.20191111211345-a27ff30ebf09+incompatible
	github.com/openshift/client-go v3.9.0+incompatible
	github.com/openshift/cloud-credential-operator v0.0.0-20210906074537-c3316bb35a5a // indirect
	github.com/openshift/cluster-api-provider-baremetal v0.0.0-20210823144712-1c81cab6cc3a // indirect
	github.com/openshift/cluster-api-provider-kubevirt v0.0.0-20210719100556-9b8bc3666720 // indirect
	github.com/openshift/console-operator v0.0.0-20210905022429-b8058325fabe
	github.com/openshift/custom-resource-status v1.1.0 // indirect
	github.com/openshift/installer v0.16.1
	github.com/openshift/library-go v0.0.0-20210906100234-6754cfd64cb5
	github.com/openshift/machine-api-operator v0.2.1-0.20210820103535-d50698c302f5
	github.com/openshift/machine-config-operator v4.2.0-alpha.0.0.20190917115525-033375cbe820+incompatible
	github.com/pborman/uuid v1.2.1 // indirect
	github.com/pires/go-proxyproto v0.6.0
	github.com/pkg/errors v0.9.1
	github.com/pquerna/cachecontrol v0.1.0 // indirect
	github.com/prometheus/client_golang v1.11.0
	github.com/prometheus/common v0.30.0
	github.com/prometheus/procfs v0.7.3 // indirect
	github.com/russross/blackfriday v1.6.0 // indirect
	github.com/sirupsen/logrus v1.8.1
	github.com/spf13/cobra v1.2.1 // indirect
	github.com/ugorji/go/codec v1.2.6
	github.com/vmware/govmomi v0.26.1 // indirect
	github.com/xlab/treeprint v1.1.0 // indirect
	go.mozilla.org/pkcs7 v0.0.0-20210826202110-33d05740a352 // indirect
	go.starlark.net v0.0.0-20210901212718-87f333178d59 // indirect
	golang.org/x/crypto v0.0.0-20210817164053-32db794688a5
	golang.org/x/mod v0.5.0 // indirect
	golang.org/x/net v0.0.0-20210903162142-ad29c8ab022f
	golang.org/x/oauth2 v0.0.0-20210819190943-2bc19b11175f
	golang.org/x/sync v0.0.0-20210220032951-036812b2e83c
	golang.org/x/sys v0.0.0-20210903071746-97244b99971b // indirect
	golang.org/x/term v0.0.0-20210615171337-6886f2dfbf5b // indirect
	golang.org/x/text v0.3.7 // indirect
	golang.org/x/time v0.0.0-20210723032227-1f47c861a9ac // indirect
	golang.org/x/tools v0.1.5
	google.golang.org/api v0.56.0 // indirect
	google.golang.org/genproto v0.0.0-20210903162649-d08c68adba83 // indirect
	gopkg.in/ini.v1 v1.62.1 // indirect
	gopkg.in/square/go-jose.v2 v2.6.0 // indirect
	gotest.tools/gotestsum v1.6.4
	k8s.io/api v0.22.1
	k8s.io/apiextensions-apiserver v0.22.1
	k8s.io/apimachinery v0.22.1
	k8s.io/cli-runtime v0.22.1 // indirect
	k8s.io/client-go v12.0.0+incompatible
	k8s.io/code-generator v0.21.4
	k8s.io/component-base v0.22.1 // indirect
	k8s.io/klog/v2 v2.20.0 // indirect
	k8s.io/kube-openapi v0.0.0-20210817084001-7fbd8d59e5b8 // indirect
	k8s.io/kubectl v0.22.1
	k8s.io/kubernetes v1.21.4
	k8s.io/utils v0.0.0-20210820185131-d34e5cb4466e // indirect
	kubevirt.io/client-go v0.44.1 // indirect
	kubevirt.io/containerized-data-importer v1.39.0 // indirect
	kubevirt.io/controller-lifecycle-operator-sdk v0.2.1 // indirect
	sigs.k8s.io/cluster-api-provider-aws v0.7.0 // indirect
	sigs.k8s.io/cluster-api-provider-azure v0.5.2
	sigs.k8s.io/cluster-api-provider-openstack v0.4.0 // indirect
	sigs.k8s.io/controller-runtime v0.10.0
	sigs.k8s.io/controller-tools v0.6.2
	sigs.k8s.io/kustomize/api v0.9.0 // indirect
	sigs.k8s.io/kustomize/kyaml v0.11.1 // indirect
)

exclude (
	// exclude github.com/golang/protobuf < 1.3.2 https://nvd.nist.gov/vuln/detail/CVE-2021-3121
	github.com/golang/protobuf v1.0.0
	github.com/golang/protobuf v1.1.1
	github.com/golang/protobuf v1.2.0
	github.com/golang/protobuf v1.2.1
	github.com/golang/protobuf v1.3.0
	github.com/golang/protobuf v1.3.1
	// exclude github.com/hashicorp/vault < v1.5.1: https://nvd.nist.gov/vuln/detail/CVE-2020-16251
	github.com/hashicorp/vault v0.10.4
	github.com/hashicorp/vault v0.11.0
	github.com/hashicorp/vault v0.11.0-beta1
	github.com/hashicorp/vault v0.11.1
	github.com/hashicorp/vault v0.11.2
	github.com/hashicorp/vault v0.11.3
	github.com/hashicorp/vault v0.11.4
	github.com/hashicorp/vault v0.11.5
	github.com/hashicorp/vault v0.11.6
	github.com/hashicorp/vault v0.11.7
	github.com/hashicorp/vault v0.11.8
	github.com/hashicorp/vault v1.0.0
	github.com/hashicorp/vault v1.0.0-beta1
	github.com/hashicorp/vault v1.0.0-beta2
	github.com/hashicorp/vault v1.0.0-rc1
	github.com/hashicorp/vault v1.0.1
	github.com/hashicorp/vault v1.0.2
	github.com/hashicorp/vault v1.0.3
	github.com/hashicorp/vault v1.1.0
	github.com/hashicorp/vault v1.1.0-beta1
	github.com/hashicorp/vault v1.1.0-beta2
	github.com/hashicorp/vault v1.1.1
	github.com/hashicorp/vault v1.1.2
	github.com/hashicorp/vault v1.1.3
	github.com/hashicorp/vault v1.1.4
	github.com/hashicorp/vault v1.1.5
	github.com/hashicorp/vault v1.2.0
	github.com/hashicorp/vault v1.2.0-beta1
	github.com/hashicorp/vault v1.2.0-beta2
	github.com/hashicorp/vault v1.2.0-rc1
	github.com/hashicorp/vault v1.2.1
	github.com/hashicorp/vault v1.2.2
	github.com/hashicorp/vault v1.2.3
	github.com/hashicorp/vault v1.2.4
	github.com/hashicorp/vault v1.3.0
	github.com/hashicorp/vault v1.3.1
	github.com/hashicorp/vault v1.3.2
	github.com/hashicorp/vault v1.3.3
	github.com/hashicorp/vault v1.3.4
	github.com/hashicorp/vault v1.3.5
	github.com/hashicorp/vault v1.3.6
	github.com/hashicorp/vault v1.3.7
	github.com/hashicorp/vault v1.4.0
	github.com/hashicorp/vault v1.4.0-beta1
	github.com/hashicorp/vault v1.4.0-beta2
	github.com/hashicorp/vault v1.4.0-beta3
	github.com/hashicorp/vault v1.4.0-rc1
	github.com/hashicorp/vault v1.4.1
	github.com/hashicorp/vault v1.4.2
	github.com/hashicorp/vault v1.4.3
	github.com/hashicorp/vault v1.5.0
	github.com/hashicorp/vault v1.5.0-beta1
	github.com/hashicorp/vault v1.5.0-beta2
	github.com/hashicorp/vault v1.5.0-rc1
	// https://www.whitesourcesoftware.com/vulnerability-database/WS-2018-0594
	github.com/satori/go.uuid v0.0.0
	github.com/satori/uuid v0.0.0
)

replace (
	bitbucket.org/ww/goautoneg => github.com/munnerz/goautoneg v0.0.0-20120707110453-a547fc61f48d // 404 on bitbucket.org/ww/goautoneg
	github.com/docker/spdystream => github.com/docker/spdystream v0.1.0
	github.com/go-openapi/spec => github.com/go-openapi/spec v0.19.8
	// Replace old GoGo Protobuf versions https://nvd.nist.gov/vuln/detail/CVE-2021-3121
	github.com/gogo/protobuf => github.com/gogo/protobuf v1.3.2
	github.com/operator-framework/operator-sdk => github.com/operator-framework/operator-sdk v0.19.4
	// https://www.whitesourcesoftware.com/vulnerability-database/WS-2018-0594
	github.com/satori/go.uuid => github.com/satori/go.uuid v1.2.1-0.20181028125025-b2ce2384e17b
	github.com/satori/uuid => github.com/satori/uuid v1.2.1-0.20181028125025-b2ce2384e17b
	github.com/spf13/viper => github.com/spf13/viper v1.7.1
	github.com/terraform-providers/terraform-provider-aws => github.com/openshift/terraform-provider-aws v1.60.1-0.20200630224953-76d1fb4e5699
	github.com/terraform-providers/terraform-provider-azurerm => github.com/openshift/terraform-provider-azurerm v1.40.1-0.20200707062554-97ea089cc12a
	github.com/terraform-providers/terraform-provider-ignition/v2 => github.com/community-terraform-providers/terraform-provider-ignition/v2 v2.1.0
	golang.org/x/tools => golang.org/x/tools v0.1.0 // We are still using Go 1.14, but >=v0.1.1 depends on standard library from Go 1.16
	k8s.io/api => k8s.io/api v0.21.4
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.21.4
	k8s.io/apimachinery => k8s.io/apimachinery v0.21.4
	k8s.io/apiserver => k8s.io/apiserver v0.21.4
	k8s.io/cli-runtime => k8s.io/cli-runtime v0.21.4
	k8s.io/client-go => k8s.io/client-go v0.21.4
	k8s.io/cloud-provider => k8s.io/cloud-provider v0.21.4
	k8s.io/cluster-bootstrap => k8s.io/cluster-bootstrap v0.21.4
	k8s.io/code-generator => k8s.io/code-generator v0.21.4
	k8s.io/component-base => k8s.io/component-base v0.21.4
	k8s.io/component-helpers => k8s.io/component-helpers v0.21.4
	k8s.io/controller-manager => k8s.io/controller-manager v0.21.4
	k8s.io/cri-api => k8s.io/cri-api v0.21.4
	k8s.io/csi-translation-lib => k8s.io/csi-translation-lib v0.21.4
	k8s.io/kube-aggregator => k8s.io/kube-aggregator v0.21.4
	k8s.io/kube-controller-manager => k8s.io/kube-controller-manager v0.21.4
	k8s.io/kube-openapi => k8s.io/kube-openapi v0.0.0-20210305001622-591a79e4bda7
	k8s.io/kube-proxy => k8s.io/kube-proxy v0.21.4
	k8s.io/kube-scheduler => k8s.io/kube-scheduler v0.21.4
	k8s.io/kubectl => k8s.io/kubectl v0.21.4
	k8s.io/kubelet => k8s.io/kubelet v0.21.4
	k8s.io/kubernetes => k8s.io/kubernetes v1.21.4
	k8s.io/legacy-cloud-providers => k8s.io/legacy-cloud-providers v0.21.4
	k8s.io/metrics => k8s.io/metrics v0.21.4
	k8s.io/mount-utils => k8s.io/mount-utils v0.21.4
	k8s.io/sample-apiserver => k8s.io/sample-apiserver v0.21.4
	sigs.k8s.io/controller-runtime => sigs.k8s.io/controller-runtime v0.9.1
	sigs.k8s.io/controller-tools => sigs.k8s.io/controller-tools v0.5.0
)

// Installer dependencies. Some of them are being used directly in the RP.
replace (
	github.com/Azure/go-autorest/autorest/adal => github.com/Azure/go-autorest/autorest/adal v0.9.14
	github.com/BurntSushi/toml => github.com/BurntSushi/toml v0.3.1
	github.com/containers/image => github.com/containers/image v3.0.2+incompatible
	github.com/coreos/stream-metadata-go => github.com/coreos/stream-metadata-go v0.0.0-20210225230131-70edb9eb47b3
	github.com/go-logr/logr => github.com/go-logr/logr v0.4.0
	github.com/kubevirt/terraform-provider-kubevirt => github.com/nirarg/terraform-provider-kubevirt v0.0.0-20201222125919-101cee051ed3
	github.com/metal3-io/baremetal-operator => github.com/openshift/baremetal-operator v0.0.0-20210706141527-5240e42f012a // Use OpenShift fork
	github.com/metal3-io/baremetal-operator/apis => github.com/openshift/baremetal-operator/apis v0.0.0-20210706141527-5240e42f012a // Use OpenShift fork
	github.com/metal3-io/cluster-api-provider-baremetal => github.com/openshift/cluster-api-provider-baremetal v0.0.0-20210721192732-726d97e15db7 // Pin OpenShift fork
	github.com/openshift/api => github.com/openshift/api v0.0.0-20210713130143-be21c6cb1bea
	github.com/openshift/client-go => github.com/openshift/client-go v0.0.0-20210521082421-73d9475a9142
	github.com/openshift/cloud-credential-operator => github.com/openshift/cloud-credential-operator v0.0.0-20200316201045-d10080b52c9e
	github.com/openshift/cluster-api-provider-gcp => github.com/openshift/cluster-api-provider-gcp v0.0.1-0.20210513231632-34db56ebf7a8
	github.com/openshift/cluster-api-provider-kubevirt => github.com/openshift/cluster-api-provider-kubevirt v0.0.0-20210515110917-b0e15d7907de
	github.com/openshift/cluster-api-provider-libvirt => github.com/openshift/cluster-api-provider-libvirt v0.2.1-0.20210812060947-9542e5ac08b7
	github.com/openshift/cluster-api-provider-ovirt => github.com/openshift/cluster-api-provider-ovirt v0.1.1-0.20210811191557-cbf023408f4e
	github.com/openshift/console-operator => github.com/openshift/console-operator v0.0.0-20210729235954-696f4645f37d
	github.com/openshift/installer => github.com/jewzaam/installer-aro v0.9.0-master.0.20210906140350-e0dddfe94b1d
	github.com/openshift/library-go => github.com/openshift/library-go v0.0.0-20210825122301-7f0bf922c345
	github.com/openshift/machine-api-operator => github.com/openshift/machine-api-operator v0.2.1-0.20210811215339-a6349c0280cc
	github.com/openshift/machine-config-operator => github.com/openshift/machine-config-operator v0.0.1-0.20210826190144-a537783ea4a0
	github.com/ovirt/go-ovirt => github.com/ovirt/go-ovirt v0.0.0-20210112072624-e4d3b104de71
	k8s.io/klog/v2 => k8s.io/klog/v2 v2.8.0
	kubevirt.io/client-go => kubevirt.io/client-go v0.29.0
	sigs.k8s.io/cluster-api-provider-aws => github.com/openshift/cluster-api-provider-aws v0.2.1-0.20210819142746-9f0a34faa04c
	sigs.k8s.io/cluster-api-provider-azure => github.com/openshift/cluster-api-provider-azure v0.1.0-alpha.3.0.20210611192943-830107632bf8
	sigs.k8s.io/cluster-api-provider-openstack => github.com/openshift/cluster-api-provider-openstack v0.0.0-20210903123455-eb8656e9dfb4
	sigs.k8s.io/kustomize/api => sigs.k8s.io/kustomize/api v0.8.8
	sigs.k8s.io/kustomize/kyaml => sigs.k8s.io/kustomize/kyaml v0.10.17
	sigs.k8s.io/structured-merge-diff => sigs.k8s.io/structured-merge-diff v1.0.1-0.20191108220359-b1b620dd3f06
)
