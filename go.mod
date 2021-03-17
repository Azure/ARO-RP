module github.com/Azure/ARO-RP

go 1.14

require (
	github.com/AlekSi/gocov-xml v0.0.0-20190121064608-3a14fb1c4737
	github.com/Azure/azure-sdk-for-go v52.4.0+incompatible
	github.com/Azure/go-autorest/autorest v0.11.18
	github.com/Azure/go-autorest/autorest/adal v0.9.13
	github.com/Azure/go-autorest/autorest/azure/auth v0.5.7
	github.com/Azure/go-autorest/autorest/date v0.3.0
	github.com/Azure/go-autorest/autorest/to v0.4.0
	github.com/Azure/go-autorest/autorest/validation v0.3.1
	github.com/Azure/go-autorest/tracing v0.6.0
	github.com/MakeNowJust/heredoc v1.0.0 // indirect
	github.com/alvaroloes/enumer v1.1.2
	github.com/apparentlymart/go-cidr v1.1.0
	github.com/aws/aws-sdk-go v1.37.32 // indirect
	github.com/axw/gocov v1.0.0
	github.com/clarketm/json v1.15.7 // indirect
	github.com/codahale/etm v0.0.0-20141003032925-c00c9e6fb4c9
	github.com/containers/image/v5 v5.10.5
	github.com/containers/libtrust v0.0.0-20200511145503-9c3a6c22cd9a // indirect
	github.com/containers/ocicrypt v1.1.0 // indirect
	github.com/containers/storage v1.28.0 // indirect
	github.com/coreos/go-oidc v2.2.1+incompatible
	github.com/coreos/go-systemd/v22 v22.2.0
	github.com/coreos/ignition/v2 v2.9.0 // indirect
	github.com/docker/docker v20.10.5+incompatible // indirect
	github.com/docker/spdystream v0.2.0 // indirect
	github.com/emicklei/go-restful v2.15.0+incompatible // indirect
	github.com/exponent-io/jsonpath v0.0.0-20201116121440-e84ac1befdf8 // indirect
	github.com/form3tech-oss/jwt-go v3.2.2+incompatible
	github.com/ghodss/yaml v1.0.1-0.20190212211648-25d852aebe32
	github.com/go-bindata/go-bindata v3.1.2+incompatible
	github.com/go-logr/logr v0.4.0
	github.com/go-openapi/jsonpointer v0.19.5 // indirect
	github.com/go-openapi/jsonreference v0.19.5 // indirect
	github.com/go-openapi/spec v0.20.3 // indirect
	github.com/go-openapi/swag v0.19.14 // indirect
	github.com/go-playground/validator/v10 v10.4.1 // indirect
	github.com/go-test/deep v1.0.7
	github.com/gofrs/uuid v4.0.0+incompatible
	github.com/golang/mock v1.5.0
	github.com/golangci/golangci-lint v1.32.2
	github.com/google/btree v1.0.1 // indirect
	github.com/google/go-cmp v0.5.5
	github.com/google/gofuzz v1.2.0 // indirect
	github.com/google/uuid v1.2.0 // indirect
	github.com/googleapis/gnostic v0.5.4
	github.com/gophercloud/gophercloud v0.16.0 // indirect
	github.com/gophercloud/utils v0.0.0-20210216074907-f6de111f2eae // indirect
	github.com/gorilla/csrf v1.7.0
	github.com/gorilla/mux v1.8.0
	github.com/gorilla/securecookie v1.1.1
	github.com/gorilla/sessions v1.2.1
	github.com/gregjones/httpcache v0.0.0-20190611155906-901d90724c79 // indirect
	github.com/h2non/filetype v1.1.1 // indirect
	github.com/imdario/mergo v0.3.12 // indirect
	github.com/jim-minter/go-cosmosdb v0.0.0-20201119201311-b37af9b82812
	github.com/jstemmer/go-junit-report v0.9.1
	github.com/leodido/go-urn v1.2.1 // indirect
	github.com/libvirt/libvirt-go v7.0.0+incompatible // indirect
	github.com/mailru/easyjson v0.7.7 // indirect
	github.com/mattn/go-runewidth v0.0.10 // indirect
	github.com/metal3-io/cluster-api-provider-baremetal v0.2.2 // indirect
	github.com/mgutz/ansi v0.0.0-20200706080929-d51e80ef957d // indirect
	github.com/mitchellh/go-wordwrap v1.0.1 // indirect
	github.com/moby/term v0.0.0-20201216013528-df9cb8a40635 // indirect
	github.com/onsi/ginkgo v1.15.2
	github.com/onsi/gomega v1.11.0
	github.com/openshift/api v3.9.1-0.20191111211345-a27ff30ebf09+incompatible
	github.com/openshift/client-go v0.0.0-20200827190008-3062137373b5
	github.com/openshift/cloud-credential-operator v0.0.0-20210304224921-06e29217a80c // indirect
	github.com/openshift/cluster-api-provider-baremetal v0.0.0-20210315131540-73c1ffba6ac8 // indirect
	github.com/openshift/console-operator v0.0.0-20210315120042-056de2ed4825
	github.com/openshift/installer v0.16.1
	github.com/openshift/machine-api-operator v0.2.1-0.20210212025836-cb508cd8777d
	github.com/openshift/machine-config-operator v4.2.0-alpha.0.0.20190917115525-033375cbe820+incompatible
	github.com/operator-framework/operator-sdk v1.5.0
	github.com/ovirt/go-ovirt v0.0.0-20210308100159-ac0bcbc88d7c // indirect
	github.com/pborman/uuid v1.2.1 // indirect
	github.com/pquerna/cachecontrol v0.0.0-20201205024021-ac21108117ac // indirect
	github.com/prometheus/client_golang v1.9.0 // indirect
	github.com/prometheus/common v0.19.0
	github.com/prometheus/procfs v0.6.0 // indirect
	github.com/rivo/uniseg v0.2.0 // indirect
	github.com/russross/blackfriday v1.6.0 // indirect
	github.com/sirupsen/logrus v1.8.1
	github.com/spf13/cobra v1.1.3 // indirect
	github.com/ugorji/go/codec v1.2.5-0.20210226122216-b5e6aedbb7b6
	github.com/ulikunitz/xz v0.5.10 // indirect
	github.com/vmware/govmomi v0.24.0 // indirect
	golang.org/x/crypto v0.0.0-20210314154223-e6e6c4f2bb5b
	golang.org/x/mod v0.4.2 // indirect
	golang.org/x/net v0.0.0-20210316092652-d523dce5a7f4
	golang.org/x/oauth2 v0.0.0-20210313182246-cd4f82c27b84
	golang.org/x/sync v0.0.0-20210220032951-036812b2e83c
	golang.org/x/sys v0.0.0-20210317091845-390168757d9c // indirect
	golang.org/x/term v0.0.0-20210220032956-6a3ed077a48d // indirect
	golang.org/x/time v0.0.0-20210220033141-f8bda1e9f3ba // indirect
	golang.org/x/tools v0.1.0
	gomodules.xyz/jsonpatch/v2 v2.1.0 // indirect
	google.golang.org/api v0.42.0 // indirect
	google.golang.org/genproto v0.0.0-20210315173758-2651cd453018 // indirect
	gopkg.in/ini.v1 v1.62.0 // indirect
	gopkg.in/yaml.v3 v3.0.0-20210107192922-496545a6307b // indirect
	k8s.io/api v0.20.4
	k8s.io/apiextensions-apiserver v0.20.4
	k8s.io/apimachinery v0.20.4
	k8s.io/apiserver v0.20.4 // indirect
	k8s.io/cli-runtime v0.20.4 // indirect
	k8s.io/client-go v12.0.0+incompatible
	k8s.io/code-generator v0.19.4
	k8s.io/component-base v0.20.4 // indirect
	k8s.io/klog/v2 v2.8.0 // indirect
	k8s.io/kube-openapi v0.0.0-20210305164622-f622666832c1 // indirect
	k8s.io/kubectl v0.20.4
	k8s.io/kubernetes v1.20.4
	k8s.io/utils v0.0.0-20210305010621-2afb4311ab10 // indirect
	sigs.k8s.io/cluster-api-provider-aws v0.6.4 // indirect
	sigs.k8s.io/cluster-api-provider-azure v0.4.13
	sigs.k8s.io/cluster-api-provider-openstack v0.3.4 // indirect
	sigs.k8s.io/controller-runtime v0.8.3
	sigs.k8s.io/controller-tools v0.4.1
)

replace (
	bitbucket.org/ww/goautoneg => github.com/munnerz/goautoneg v0.0.0-20120707110453-a547fc61f48d // 404 on bitbucket.org/ww/goautoneg
	github.com/docker/spdystream => github.com/docker/spdystream v0.1.0
	github.com/go-openapi/spec => github.com/go-openapi/spec v0.19.8
	github.com/metal3-io/baremetal-operator => github.com/openshift/baremetal-operator v0.0.0-20210128152529-b4b10a088a0c
	github.com/metal3-io/cluster-api-provider-baremetal => github.com/openshift/cluster-api-provider-baremetal v0.0.0-20210304234017-16b67b78b538
	github.com/openshift/api => github.com/openshift/api v0.0.0-20210127195806-54e5e88cf848
	github.com/openshift/client-go => github.com/openshift/client-go v0.0.0-20200929181438-91d71ef2122c
	github.com/openshift/cloud-credential-operator => github.com/openshift/cloud-credential-operator v0.0.0-20201202215507-371eb009d9a1
	github.com/openshift/cluster-api => github.com/openshift/cluster-api v0.0.0-20191129101638-b09907ac6668
	github.com/openshift/cluster-api-provider-gcp => github.com/openshift/cluster-api-provider-gcp v0.0.1-0.20210226012455-4480c4f95562
	github.com/openshift/cluster-api-provider-libvirt => github.com/openshift/cluster-api-provider-libvirt v0.2.1-0.20200919090150-1ca52adab176
	github.com/openshift/cluster-api-provider-ovirt => github.com/openshift/cluster-api-provider-ovirt v0.1.1-0.20210210114935-91f12f3f7dee
	github.com/openshift/console-operator => github.com/openshift/console-operator v0.0.0-20210116095614-7fd78a283616
	github.com/openshift/installer => github.com/jim-minter/installer v0.9.0-master.0.20210303004247-74b5bf8dae14
	github.com/openshift/machine-api-operator => github.com/openshift/machine-api-operator v0.2.1-0.20210304062120-b5fb76b27015
	github.com/openshift/machine-config-operator => github.com/openshift/machine-config-operator v0.0.1-0.20210304004022-5ce9351ceb24
	github.com/operator-framework/operator-sdk => github.com/operator-framework/operator-sdk v0.19.4
	github.com/terraform-providers/terraform-provider-aws => github.com/openshift/terraform-provider-aws v1.60.1-0.20200630224953-76d1fb4e5699
	github.com/terraform-providers/terraform-provider-azurerm => github.com/openshift/terraform-provider-azurerm v1.40.1-0.20200707062554-97ea089cc12a
	github.com/terraform-providers/terraform-provider-ignition/v2 => github.com/community-terraform-providers/terraform-provider-ignition/v2 v2.1.0
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
	sigs.k8s.io/cluster-api-provider-aws => github.com/openshift/cluster-api-provider-aws v0.2.1-0.20210225235157-d5dfdd66e309
	sigs.k8s.io/cluster-api-provider-azure => github.com/openshift/cluster-api-provider-azure v0.1.0-alpha.3.0.20210226041523-4663098af22b
	sigs.k8s.io/cluster-api-provider-gcp => github.com/openshift/cluster-api-provider-gcp v0.0.1-0.20210226012455-4480c4f95562
	sigs.k8s.io/cluster-api-provider-openstack => github.com/openshift/cluster-api-provider-openstack v0.0.0-20210226021256-cc9b82526d26
	sigs.k8s.io/controller-runtime => sigs.k8s.io/controller-runtime v0.6.4
	sigs.k8s.io/controller-tools => sigs.k8s.io/controller-tools v0.3.0
)
