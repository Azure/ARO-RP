module github.com/Azure/ARO-RP

go 1.14

require (
	cloud.google.com/go v0.82.0 // indirect
	github.com/AlekSi/gocov-xml v0.0.0-20190121064608-3a14fb1c4737
	github.com/Azure/azure-sdk-for-go v56.1.0+incompatible
	github.com/Azure/go-autorest/autorest v0.11.19
	github.com/Azure/go-autorest/autorest/adal v0.9.14
	github.com/Azure/go-autorest/autorest/azure/auth v0.5.7
	github.com/Azure/go-autorest/autorest/date v0.3.0
	github.com/Azure/go-autorest/autorest/to v0.4.0
	github.com/Azure/go-autorest/autorest/validation v0.3.1
	github.com/Azure/go-autorest/tracing v0.6.0
	github.com/alvaroloes/enumer v1.1.2
	github.com/apparentlymart/go-cidr v1.1.0
	github.com/aws/aws-sdk-go v1.38.46 // indirect
	github.com/axw/gocov v1.0.0
	github.com/clarketm/json v1.15.7 // indirect
	github.com/codahale/etm v0.0.0-20141003032925-c00c9e6fb4c9
	github.com/containers/image/v5 v5.12.0
	github.com/containers/libtrust v0.0.0-20200511145503-9c3a6c22cd9a // indirect
	github.com/containers/storage v1.31.2 // indirect
	github.com/coreos/go-oidc v2.2.1+incompatible
	github.com/coreos/go-systemd/v22 v22.3.2
	github.com/coreos/ignition/v2 v2.10.1
	github.com/coreos/vcontext v0.0.0-20210511154938-8fbad538d364 // indirect
	github.com/davecgh/go-spew v1.1.1
	github.com/docker/docker v20.10.6+incompatible // indirect
	github.com/docker/spdystream v0.2.0 // indirect
	github.com/emicklei/go-restful v2.15.0+incompatible // indirect
	github.com/exponent-io/jsonpath v0.0.0-20210407135951-1de76d718b3f // indirect
	github.com/form3tech-oss/jwt-go v3.2.3+incompatible
	github.com/ghodss/yaml v1.0.1-0.20190212211648-25d852aebe32
	github.com/go-bindata/go-bindata v3.1.2+incompatible
	github.com/go-logr/logr v0.4.0
	github.com/go-openapi/jsonpointer v0.19.5 // indirect
	github.com/go-openapi/jsonreference v0.19.5 // indirect
	github.com/go-openapi/spec v0.20.3 // indirect
	github.com/go-openapi/swag v0.19.15 // indirect
	github.com/go-playground/validator/v10 v10.6.1 // indirect
	github.com/go-test/deep v1.0.7
	github.com/gofrs/uuid v4.0.0+incompatible
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da // indirect
	github.com/golang/mock v1.5.0
	github.com/golangci/golangci-lint v1.32.2
	github.com/google/btree v1.0.1 // indirect
	github.com/google/go-cmp v0.5.6
	github.com/googleapis/gnostic v0.5.5
	github.com/gophercloud/gophercloud v0.17.0 // indirect
	github.com/gorilla/csrf v1.7.0
	github.com/gorilla/mux v1.8.0
	github.com/gorilla/securecookie v1.1.1
	github.com/gorilla/sessions v1.2.1
	github.com/h2non/filetype v1.1.1 // indirect
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/imdario/mergo v0.3.12 // indirect
	github.com/jstemmer/go-junit-report v0.9.1
	github.com/k8snetworkplumbingwg/network-attachment-definition-client v1.1.0 // indirect
	github.com/klauspost/compress v1.12.3 // indirect
	github.com/leodido/go-urn v1.2.1 // indirect
	github.com/libvirt/libvirt-go v7.3.0+incompatible // indirect
	github.com/mailru/easyjson v0.7.7 // indirect
	github.com/mattn/go-runewidth v0.0.13 // indirect
	github.com/metal3-io/cluster-api-provider-baremetal v0.2.2 // indirect
	github.com/mgutz/ansi v0.0.0-20200706080929-d51e80ef957d // indirect
	github.com/mitchellh/go-wordwrap v1.0.1 // indirect
	github.com/mjudeikis/go-cosmosdb v0.0.0-20210518104404-b205b3cefd36
	github.com/moby/term v0.0.0-20201216013528-df9cb8a40635 // indirect
	github.com/onsi/ginkgo v1.16.2
	github.com/onsi/gomega v1.12.0
	github.com/openshift/api v3.9.1-0.20191111211345-a27ff30ebf09+incompatible
	github.com/openshift/client-go v3.9.0+incompatible
	github.com/openshift/cloud-credential-operator v0.0.0-20210525141023-02cc6303cd10 // indirect
	github.com/openshift/cluster-api-provider-baremetal v0.0.0-20210518162658-a60d493e45aa // indirect
	github.com/openshift/cluster-api-provider-kubevirt v0.0.0-20210515110917-b0e15d7907de // indirect
	github.com/openshift/console-operator v0.0.0-20210518192856-2d7be8eea682
	github.com/openshift/custom-resource-status v1.1.0 // indirect
	github.com/openshift/installer v0.16.1
	github.com/openshift/library-go v0.0.0-20210521084623-7392ea9b02ca
	github.com/openshift/machine-api-operator v0.2.1-0.20210516083017-bb9e0b5c1170
	github.com/openshift/machine-config-operator v4.2.0-alpha.0.0.20190917115525-033375cbe820+incompatible
	github.com/operator-framework/operator-sdk v1.7.2
	github.com/pborman/uuid v1.2.1 // indirect
	github.com/pires/go-proxyproto v0.5.0
	github.com/pkg/errors v0.9.1
	github.com/pquerna/cachecontrol v0.1.0 // indirect
	github.com/prometheus/client_golang v1.10.0
	github.com/prometheus/common v0.25.0
	github.com/russross/blackfriday v1.6.0 // indirect
	github.com/sirupsen/logrus v1.8.1
	github.com/spf13/cobra v1.1.3 // indirect
	github.com/ugorji/go/codec v1.2.6
	github.com/vbauerster/mpb/v6 v6.0.4 // indirect
	github.com/vmware/govmomi v0.25.0 // indirect
	golang.org/x/crypto v0.0.0-20210513164829-c07d793c2f9a
	golang.org/x/mod v0.4.2 // indirect
	golang.org/x/net v0.0.0-20210525063256-abc453219eb5
	golang.org/x/oauth2 v0.0.0-20210514164344-f6687ab2804c
	golang.org/x/sync v0.0.0-20210220032951-036812b2e83c
	golang.org/x/sys v0.0.0-20210525143221-35b2ab0089ea // indirect
	golang.org/x/term v0.0.0-20210503060354-a79de5458b56 // indirect
	golang.org/x/time v0.0.0-20210220033141-f8bda1e9f3ba // indirect
	golang.org/x/tools v0.1.1
	gomodules.xyz/jsonpatch/v2 v2.2.0 // indirect
	google.golang.org/api v0.47.0 // indirect
	google.golang.org/genproto v0.0.0-20210524171403-669157292da3 // indirect
	google.golang.org/grpc v1.38.0 // indirect
	gopkg.in/ini.v1 v1.62.0 // indirect
	gopkg.in/yaml.v3 v3.0.0-20210107192922-496545a6307b // indirect
	gotest.tools/gotestsum v1.6.4
	k8s.io/api v0.21.1
	k8s.io/apiextensions-apiserver v0.21.1
	k8s.io/apimachinery v0.21.1
	k8s.io/apiserver v0.21.1 // indirect
	k8s.io/cli-runtime v0.21.1 // indirect
	k8s.io/client-go v12.0.0+incompatible
	k8s.io/code-generator v0.20.6
	k8s.io/component-base v0.21.1 // indirect
	k8s.io/kube-openapi v0.0.0-20210524163139-412c2b45c7d3 // indirect
	k8s.io/kubectl v0.21.1
	k8s.io/kubernetes v1.21.1
	k8s.io/utils v0.0.0-20210521133846-da695404a2bc // indirect
	kubevirt.io/client-go v0.41.0 // indirect
	kubevirt.io/containerized-data-importer v1.34.0 // indirect
	sigs.k8s.io/cluster-api-provider-aws v0.6.6 // indirect
	sigs.k8s.io/cluster-api-provider-azure v0.4.15
	sigs.k8s.io/cluster-api-provider-openstack v0.3.4 // indirect
	sigs.k8s.io/controller-runtime v0.9.0-beta.1.0.20210512131817-ce2f0c92d77e
	sigs.k8s.io/controller-tools v0.5.0
	sigs.k8s.io/structured-merge-diff/v3 v3.0.1 // indirect
)

exclude (
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
	github.com/operator-framework/operator-sdk => github.com/operator-framework/operator-sdk v0.19.4
	// https://www.whitesourcesoftware.com/vulnerability-database/WS-2018-0594
	github.com/satori/go.uuid => github.com/satori/go.uuid v1.2.1-0.20181028125025-b2ce2384e17b
	github.com/satori/uuid => github.com/satori/uuid v1.2.1-0.20181028125025-b2ce2384e17b
	github.com/terraform-providers/terraform-provider-aws => github.com/openshift/terraform-provider-aws v1.60.1-0.20200630224953-76d1fb4e5699
	github.com/terraform-providers/terraform-provider-azurerm => github.com/openshift/terraform-provider-azurerm v1.40.1-0.20200707062554-97ea089cc12a
	github.com/terraform-providers/terraform-provider-ignition/v2 => github.com/community-terraform-providers/terraform-provider-ignition/v2 v2.1.0
	golang.org/x/tools => golang.org/x/tools v0.1.0 // We are still using Go 1.14, but >=v0.1.1 depends on standard library from Go 1.16
	k8s.io/api => k8s.io/api v0.19.0-rc.2
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.19.0-rc.2
	k8s.io/apimachinery => k8s.io/apimachinery v0.19.0-rc.2
	k8s.io/apiserver => k8s.io/apiserver v0.19.0-rc.2
	k8s.io/cli-runtime => k8s.io/cli-runtime v0.19.0-rc.2
	k8s.io/client-go => k8s.io/client-go v0.19.0-rc.2
	k8s.io/cloud-provider => k8s.io/cloud-provider v0.19.0-rc.2
	k8s.io/cluster-bootstrap => k8s.io/cluster-bootstrap v0.19.0-rc.2
	k8s.io/code-generator => k8s.io/code-generator v0.19.0-rc.2
	k8s.io/component-base => k8s.io/component-base v0.19.0-rc.2
	k8s.io/cri-api => k8s.io/cri-api v0.19.0-rc.2
	k8s.io/csi-translation-lib => k8s.io/csi-translation-lib v0.19.0-rc.2
	k8s.io/kube-aggregator => k8s.io/kube-aggregator v0.19.0-rc.2
	k8s.io/kube-controller-manager => k8s.io/kube-controller-manager v0.19.0-rc.2
	k8s.io/kube-openapi => k8s.io/kube-openapi v0.0.0-20200805222855-6aeccd4b50c6
	k8s.io/kube-proxy => k8s.io/kube-proxy v0.19.0-rc.2
	k8s.io/kube-scheduler => k8s.io/kube-scheduler v0.19.0-rc.2
	k8s.io/kubectl => k8s.io/kubectl v0.19.0-rc.2
	k8s.io/kubelet => k8s.io/kubelet v0.19.0-rc.2
	k8s.io/kubernetes => k8s.io/kubernetes v1.19.0-rc.2
	k8s.io/legacy-cloud-providers => k8s.io/legacy-cloud-providers v0.19.0-rc.2
	k8s.io/metrics => k8s.io/metrics v0.19.0-rc.2
	k8s.io/sample-apiserver => k8s.io/sample-apiserver v0.19.0-rc.2
	sigs.k8s.io/controller-runtime => sigs.k8s.io/controller-runtime v0.7.0
	sigs.k8s.io/controller-tools => sigs.k8s.io/controller-tools v0.5.0
)

// Installer dependencies. Some of them are being used directly in the RP.
replace (
	github.com/kubevirt/terraform-provider-kubevirt => github.com/nirarg/terraform-provider-kubevirt v0.0.0-20201222125919-101cee051ed3
	github.com/metal3-io/baremetal-operator => github.com/openshift/baremetal-operator v0.0.0-20210517134619-524201a0dbaa
	github.com/metal3-io/cluster-api-provider-baremetal => github.com/openshift/cluster-api-provider-baremetal v0.0.0-20210721192732-726d97e15db7
	github.com/openshift/api => github.com/openshift/api v0.0.0-20210428205234-a8389931bee7
	github.com/openshift/client-go => github.com/openshift/client-go v0.0.0-20210112165513-ebc401615f47
	github.com/openshift/cloud-credential-operator => github.com/openshift/cloud-credential-operator v0.0.0-20200316201045-d10080b52c9e
	github.com/openshift/cluster-api-provider-gcp => github.com/openshift/cluster-api-provider-gcp v0.0.1-0.20210521181620-5f6589d4ef94
	github.com/openshift/cluster-api-provider-kubevirt => github.com/openshift/cluster-api-provider-kubevirt v0.0.0-20210114164510-1f8fc18a50aa
	github.com/openshift/cluster-api-provider-libvirt => github.com/openshift/cluster-api-provider-libvirt v0.2.1-0.20210324200850-033be25ca038
	github.com/openshift/cluster-api-provider-ovirt => github.com/openshift/cluster-api-provider-ovirt v0.1.1-0.20210409185359-01b9bf8368a3
	github.com/openshift/console-operator => github.com/openshift/console-operator v0.0.0-20210526201839-44a0308f894a
	github.com/openshift/installer => github.com/mjudeikis/installer v0.9.0-master.0.20210806071517-1eb41d5665c8
	github.com/openshift/library-go => github.com/openshift/library-go v0.0.0-20210615164315-be4204e96f56
	github.com/openshift/machine-api-operator => github.com/openshift/machine-api-operator v0.2.1-0.20210521181620-e179bb5ce397
	github.com/openshift/machine-config-operator => github.com/openshift/machine-config-operator v0.0.1-0.20210730044813-c9fce5a27bb7
	github.com/ovirt/go-ovirt => github.com/ovirt/go-ovirt v0.0.0-20210112072624-e4d3b104de71
	kubevirt.io/client-go => kubevirt.io/client-go v0.29.0
	sigs.k8s.io/cluster-api-provider-aws => github.com/openshift/cluster-api-provider-aws v0.2.1-0.20210625171553-5368195c02ca
	sigs.k8s.io/cluster-api-provider-azure => github.com/openshift/cluster-api-provider-azure v0.1.0-alpha.3.0.20210805185638-723b7ab15767
	sigs.k8s.io/cluster-api-provider-openstack => github.com/openshift/cluster-api-provider-openstack v0.0.0-20210122093124-471cf3ab636c
	sigs.k8s.io/structured-merge-diff => sigs.k8s.io/structured-merge-diff v1.0.1-0.20191108220359-b1b620dd3f06
)
