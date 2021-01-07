module github.com/Azure/ARO-RP

go 1.14

exclude (
	github.com/terraform-providers/terraform-provider-aws v0.0.0
	github.com/terraform-providers/terraform-provider-azurerm v0.0.0
)

require (
	cloud.google.com/go v0.74.0 // indirect
	github.com/AlekSi/gocov-xml v0.0.0-20190121064608-3a14fb1c4737
	github.com/Azure/azure-sdk-for-go v49.2.0+incompatible
	github.com/Azure/go-autorest/autorest v0.11.15
	github.com/Azure/go-autorest/autorest/adal v0.9.10
	github.com/Azure/go-autorest/autorest/azure/auth v0.5.5
	github.com/Azure/go-autorest/autorest/date v0.3.0
	github.com/Azure/go-autorest/autorest/to v0.4.0
	github.com/Azure/go-autorest/autorest/validation v0.3.1
	github.com/Azure/go-autorest/tracing v0.6.0
	github.com/Microsoft/go-winio v0.4.16 // indirect
	github.com/alvaroloes/enumer v1.1.2
	github.com/apparentlymart/go-cidr v1.1.0
	github.com/aws/aws-sdk-go v1.36.22 // indirect
	github.com/axw/gocov v1.0.0
	github.com/clarketm/json v1.15.0 // indirect
	github.com/codahale/etm v0.0.0-20141003032925-c00c9e6fb4c9
	github.com/containers/image/v5 v5.9.0
	github.com/containers/libtrust v0.0.0-20200511145503-9c3a6c22cd9a // indirect
	github.com/containers/storage v1.24.4 // indirect
	github.com/coreos/go-oidc v2.2.1+incompatible
	github.com/coreos/go-systemd/v22 v22.1.0
	github.com/coreos/ignition/v2 v2.8.1 // indirect
	github.com/dimchansky/utfbom v1.1.1 // indirect
	github.com/docker/docker v20.10.2+incompatible // indirect
	github.com/form3tech-oss/jwt-go v3.2.2+incompatible
	github.com/ghodss/yaml v1.0.1-0.20190212211648-25d852aebe32
	github.com/go-bindata/go-bindata v3.1.2+incompatible
	github.com/go-logr/logr v0.3.0
	github.com/go-playground/validator/v10 v10.4.1 // indirect
	github.com/go-test/deep v1.0.7
	github.com/golang/mock v1.4.4
	github.com/golangci/golangci-lint v1.32.2
	github.com/google/go-cmp v0.5.4
	github.com/google/gofuzz v1.2.0 // indirect
	github.com/google/uuid v1.1.4 // indirect
	github.com/googleapis/gnostic v0.5.3
	github.com/gophercloud/gophercloud v0.15.0 // indirect
	github.com/gophercloud/utils v0.0.0-20201221200157-19f764eec2b7 // indirect
	github.com/gorilla/csrf v1.7.0
	github.com/gorilla/mux v1.8.0
	github.com/gorilla/securecookie v1.1.1
	github.com/gorilla/sessions v1.2.1
	github.com/gregjones/httpcache v0.0.0-20190611155906-901d90724c79 // indirect
	github.com/jim-minter/go-cosmosdb v0.0.0-20201119201311-b37af9b82812
	github.com/jstemmer/go-junit-report v0.9.1
	github.com/leodido/go-urn v1.2.1 // indirect
	github.com/libvirt/libvirt-go v6.10.0+incompatible // indirect
	github.com/mgutz/ansi v0.0.0-20200706080929-d51e80ef957d // indirect
	github.com/nxadm/tail v1.4.6 // indirect
	github.com/onsi/ginkgo v1.14.2
	github.com/onsi/gomega v1.10.4
	github.com/openshift/api v3.9.1-0.20191111211345-a27ff30ebf09+incompatible
	github.com/openshift/client-go v0.0.0-20200827190008-3062137373b5
	github.com/openshift/console-operator v0.0.0-20201215204150-c4f32c549395
	github.com/openshift/installer v0.16.1
	github.com/openshift/machine-api-operator v0.2.1-0.20201111151924-77300d0c997a
	github.com/openshift/machine-config-operator v4.2.0-alpha.0.0.20190917115525-033375cbe820+incompatible
	github.com/operator-framework/operator-sdk v1.3.0
	github.com/pborman/uuid v1.2.1 // indirect
	github.com/pquerna/cachecontrol v0.0.0-20201205024021-ac21108117ac // indirect
	github.com/prometheus/client_golang v1.9.0 // indirect
	github.com/prometheus/common v0.15.0
	github.com/satori/go.uuid v1.2.0
	github.com/shurcooL/vfsgen v0.0.0-20200824052919-0d455de96546 // indirect
	github.com/sirupsen/logrus v1.7.0
	github.com/syndtr/gocapability v0.0.0-20200815063812-42c35b437635 // indirect
	github.com/ugorji/go/codec v1.2.2
	github.com/ulikunitz/xz v0.5.9 // indirect
	github.com/vbauerster/mpb/v5 v5.4.0 // indirect
	github.com/vmware/govmomi v0.24.0 // indirect
	golang.org/x/crypto v0.0.0-20201221181555-eec23a3978ad
	golang.org/x/net v0.0.0-20201224014010-6772e930b67b
	golang.org/x/oauth2 v0.0.0-20201208152858-08078c50e5b5
	golang.org/x/sync v0.0.0-20201207232520-09787c993a3a
	golang.org/x/sys v0.0.0-20210105210732-16f7687f5001 // indirect
	golang.org/x/term v0.0.0-20201210144234-2321bbc49cbf // indirect
	golang.org/x/time v0.0.0-20201208040808-7e3f01d25324 // indirect
	golang.org/x/tools v0.0.0-20210106214847-113979e3529a
	google.golang.org/genproto v0.0.0-20210106152847-07624b53cd92 // indirect
	gopkg.in/ini.v1 v1.62.0 // indirect
	gopkg.in/square/go-jose.v2 v2.5.1 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	gopkg.in/yaml.v3 v3.0.0-20210106172901-c476de37821d // indirect
	k8s.io/api v0.20.1
	k8s.io/apiextensions-apiserver v0.20.1
	k8s.io/apimachinery v0.20.1
	k8s.io/client-go v12.0.0+incompatible
	k8s.io/code-generator v0.19.4
	k8s.io/component-base v0.20.1 // indirect
	k8s.io/kube-openapi v0.0.0-20201113171705-d219536bb9fd // indirect
	sigs.k8s.io/cluster-api-provider-aws v0.6.3 // indirect
	sigs.k8s.io/cluster-api-provider-azure v0.4.10
	sigs.k8s.io/cluster-api-provider-openstack v0.3.3 // indirect
	sigs.k8s.io/controller-runtime v0.7.0
	sigs.k8s.io/controller-tools v0.3.1-0.20200617211605-651903477185
	sigs.k8s.io/structured-merge-diff/v4 v4.0.2 // indirect
	sigs.k8s.io/yaml v1.2.0
)

replace (
	bitbucket.org/ww/goautoneg => github.com/munnerz/goautoneg v0.0.0-20120707110453-a547fc61f48d // 404 on bitbucket.org/ww/goautoneg
	github.com/coreos/go-systemd => github.com/coreos/go-systemd/v22 v22.0.0
	github.com/gophercloud/gophercloud => github.com/gophercloud/gophercloud v0.8.0
	github.com/metal3-io/baremetal-operator => github.com/openshift/baremetal-operator v0.0.0-20201023182211-14317000ae07
	github.com/openshift/api => github.com/openshift/api v0.0.0-20200930075302-db52bc4ef99f
	github.com/openshift/client-go => github.com/openshift/client-go v0.0.0-20200929181438-91d71ef2122c
	github.com/openshift/cloud-credential-operator => github.com/openshift/cloud-credential-operator v0.0.0-20201116043024-f6027aa8dce3
	github.com/openshift/cluster-api => github.com/openshift/cluster-api v0.0.0-20191129101638-b09907ac6668
	github.com/openshift/cluster-api-provider-baremetal => github.com/openshift/cluster-api-provider-baremetal v0.0.0-20201105032354-fcd9e769a45c
	github.com/openshift/cluster-api-provider-gcp => github.com/openshift/cluster-api-provider-gcp v0.0.1-0.20201002153134-a0fc9aa4ce81
	github.com/openshift/cluster-api-provider-libvirt => github.com/openshift/cluster-api-provider-libvirt v0.2.1-0.20200919090150-1ca52adab176
	github.com/openshift/cluster-api-provider-ovirt => github.com/openshift/cluster-api-provider-ovirt v0.1.1-0.20201223144549-488f970d6c53
	github.com/openshift/console-operator => github.com/openshift/console-operator v0.0.0-20200930183448-b195cebf52ea
	github.com/openshift/installer => github.com/jim-minter/installer v0.9.0-master.0.20210107032832-25aa614f1e0a
	github.com/openshift/machine-api-operator => github.com/openshift/machine-api-operator v0.2.1-0.20201119210302-9643c3accfda
	github.com/openshift/machine-config-operator => github.com/openshift/machine-config-operator v0.0.1-0.20201222202713-eab9c35dfbeb
	github.com/operator-framework/operator-sdk => github.com/operator-framework/operator-sdk v0.19.4
	k8s.io/api => k8s.io/api v0.19.4
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.19.4
	k8s.io/apimachinery => k8s.io/apimachinery v0.19.4
	k8s.io/client-go => k8s.io/client-go v0.19.4
	k8s.io/code-generator => k8s.io/code-generator v0.19.4
	k8s.io/kube-openapi => k8s.io/kube-openapi v0.0.0-20200805222855-6aeccd4b50c6
	k8s.io/kubectl => k8s.io/kubectl v0.19.4
	sigs.k8s.io/cluster-api-provider-aws => github.com/openshift/cluster-api-provider-aws v0.2.1-0.20201002185235-b1a6ba661ed8
	sigs.k8s.io/cluster-api-provider-azure => github.com/openshift/cluster-api-provider-azure v0.1.0-alpha.3.0.20201119004617-db9109863f2f
	sigs.k8s.io/cluster-api-provider-gcp => github.com/openshift/cluster-api-provider-gcp v0.0.1-0.20200917183408-90e92ed9fd9b
	sigs.k8s.io/cluster-api-provider-openstack => github.com/openshift/cluster-api-provider-openstack v0.0.0-20200917103425-f6733e6dec7b
)
