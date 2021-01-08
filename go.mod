module github.com/Azure/ARO-RP

go 1.14

exclude (
	github.com/terraform-providers/terraform-provider-aws v0.0.0
	github.com/terraform-providers/terraform-provider-azurerm v0.0.0
)

require (
	cloud.google.com/go v0.72.0 // indirect
	github.com/AlekSi/gocov-xml v0.0.0-20190121064608-3a14fb1c4737
	github.com/Azure/azure-sdk-for-go v49.0.0+incompatible
	github.com/Azure/go-autorest/autorest v0.11.12
	github.com/Azure/go-autorest/autorest/adal v0.9.5
	github.com/Azure/go-autorest/autorest/azure/auth v0.5.3
	github.com/Azure/go-autorest/autorest/date v0.3.0
	github.com/Azure/go-autorest/autorest/to v0.4.0
	github.com/Azure/go-autorest/autorest/validation v0.3.0
	github.com/Azure/go-autorest/tracing v0.6.0
	github.com/ajeddeloh/go-json v0.0.0-20200220154158-5ae607161559 // indirect
	github.com/alvaroloes/enumer v1.1.2
	github.com/apparentlymart/go-cidr v1.1.0
	github.com/aws/aws-sdk-go v1.36.0 // indirect
	github.com/axw/gocov v1.0.0
	github.com/codahale/etm v0.0.0-20141003032925-c00c9e6fb4c9
	github.com/containers/image/v5 v5.8.1
	github.com/containers/libtrust v0.0.0-20200511145503-9c3a6c22cd9a // indirect
	github.com/containers/storage v1.24.1 // indirect
	github.com/coreos/go-oidc v2.1.0+incompatible
	github.com/coreos/go-systemd/v22 v22.1.0
	github.com/dimchansky/utfbom v1.1.1 // indirect
	github.com/emicklei/go-restful v2.15.0+incompatible // indirect
	github.com/form3tech-oss/jwt-go v3.2.2+incompatible
	github.com/ghodss/yaml v1.0.1-0.20190212211648-25d852aebe32
	github.com/go-bindata/go-bindata v3.1.2+incompatible
	github.com/go-logr/logr v0.3.0
	github.com/go-openapi/spec v0.19.15 // indirect
	github.com/go-playground/validator/v10 v10.4.1 // indirect
	github.com/go-test/deep v1.0.7
	github.com/golang/mock v1.4.4
	github.com/golangci/golangci-lint v1.32.2
	github.com/google/go-cmp v0.5.4
	github.com/google/gofuzz v1.2.0 // indirect
	github.com/googleapis/gnostic v0.5.3
	github.com/gophercloud/gophercloud v0.14.0 // indirect
	github.com/gophercloud/utils v0.0.0-20201202205831-7faa8ca08dfe // indirect
	github.com/gorilla/csrf v1.7.0
	github.com/gorilla/mux v1.8.0
	github.com/gorilla/securecookie v1.1.1
	github.com/gorilla/sessions v1.2.1
	github.com/gregjones/httpcache v0.0.0-20190611155906-901d90724c79 // indirect
	github.com/h2non/filetype v1.1.0 // indirect
	github.com/jim-minter/go-cosmosdb v0.0.0-20201119201311-b37af9b82812
	github.com/jstemmer/go-junit-report v0.9.1
	github.com/libvirt/libvirt-go v6.10.0+incompatible // indirect
	github.com/mgutz/ansi v0.0.0-20200706080929-d51e80ef957d // indirect
	github.com/nxadm/tail v1.4.5 // indirect
	github.com/onsi/ginkgo v1.14.2
	github.com/onsi/gomega v1.10.3
	github.com/openshift/api v3.9.1-0.20191111211345-a27ff30ebf09+incompatible
	github.com/openshift/client-go v0.0.0-20200827190008-3062137373b5
	github.com/openshift/cloud-credential-operator v0.0.0-20201120003004-b3f99dcadf5a // indirect
	github.com/openshift/cluster-api-provider-baremetal v0.0.0-20201123200826-dd58d76aef46 // indirect
	github.com/openshift/console-operator v0.0.0-20201203125343-b1238c381caa
	github.com/openshift/installer v0.16.1
	github.com/openshift/machine-api-operator v0.2.1-0.20201002104344-6abfb5440597
	github.com/openshift/machine-config-operator v4.2.0-alpha.0.0.20190917115525-033375cbe820+incompatible
	github.com/operator-framework/operator-sdk v1.2.0
	github.com/ovirt/go-ovirt v0.0.0-20201023070830-77e357c438d5 // indirect
	github.com/pborman/uuid v1.2.1 // indirect
	github.com/prometheus/client_golang v1.8.0 // indirect
	github.com/prometheus/common v0.15.0
	github.com/satori/go.uuid v1.2.0
	github.com/sirupsen/logrus v1.7.0
	github.com/syndtr/gocapability v0.0.0-20200815063812-42c35b437635 // indirect
	github.com/ugorji/go/codec v1.2.0
	github.com/vmware/govmomi v0.23.1 // indirect
	go4.org v0.0.0-20200411211856-f5505b9728dd // indirect
	golang.org/x/crypto v0.0.0-20201124201722-c8d3bf9c5392
	golang.org/x/mod v0.4.0 // indirect
	golang.org/x/net v0.0.0-20201202161906-c7110b5ffcbb
	golang.org/x/oauth2 v0.0.0-20201203001011-0b49973bad19
	golang.org/x/sync v0.0.0-20201020160332-67f06af15bc9
	golang.org/x/sys v0.0.0-20201202213521-69691e467435 // indirect
	golang.org/x/term v0.0.0-20201126162022-7de9c90e9dd1 // indirect
	golang.org/x/time v0.0.0-20200630173020-3af7569d3a1e // indirect
	golang.org/x/tools v0.0.0-20201202200335-bef1c476418a
	gomodules.xyz/jsonpatch/v2 v2.1.0 // indirect
	google.golang.org/appengine v1.6.7 // indirect
	google.golang.org/genproto v0.0.0-20201203001206-6486ece9c497 // indirect
	google.golang.org/grpc v1.34.0 // indirect
	gopkg.in/ini.v1 v1.62.0 // indirect
	gopkg.in/square/go-jose.v2 v2.5.1 // indirect
	k8s.io/api v0.19.4
	k8s.io/apiextensions-apiserver v0.19.4
	k8s.io/apimachinery v0.19.4
	k8s.io/client-go v12.0.0+incompatible
	k8s.io/code-generator v0.19.0
	k8s.io/gengo v0.0.0-20200413195148-3a45101e95ac // indirect
	k8s.io/klog/v2 v2.4.0 // indirect
	k8s.io/kube-openapi v0.0.0-20201113171705-d219536bb9fd // indirect
	k8s.io/utils v0.0.0-20201110183641-67b214c5f920 // indirect
	sigs.k8s.io/cluster-api-provider-aws v0.6.3 // indirect
	sigs.k8s.io/cluster-api-provider-azure v0.4.10
	sigs.k8s.io/cluster-api-provider-openstack v0.3.3 // indirect
	sigs.k8s.io/controller-runtime v0.6.4
	sigs.k8s.io/controller-tools v0.3.0
	sigs.k8s.io/yaml v1.2.0
)

replace (
	bitbucket.org/ww/goautoneg => github.com/munnerz/goautoneg v0.0.0-20120707110453-a547fc61f48d // 404 on bitbucket.org/ww/goautoneg
	github.com/coreos/go-systemd => github.com/coreos/go-systemd/v22 v22.0.0
	github.com/gophercloud/gophercloud => github.com/gophercloud/gophercloud v0.8.0
	github.com/metal3-io/baremetal-operator => github.com/openshift/baremetal-operator v0.0.0-20201023182211-14317000ae07
	github.com/openshift/api => github.com/openshift/api v0.0.0-20200917102736-0a191b5b9bb0
	github.com/openshift/client-go => github.com/openshift/client-go v0.0.0-20200521150516-05eb9880269c
	github.com/openshift/cloud-credential-operator => github.com/openshift/cloud-credential-operator v0.0.0-20201202215507-371eb009d9a1
	github.com/openshift/cluster-api => github.com/openshift/cluster-api v0.0.0-20191129101638-b09907ac6668
	github.com/openshift/cluster-api-provider-baremetal => github.com/openshift/cluster-api-provider-baremetal v0.0.0-20200911144710-1cf0189fc640
	github.com/openshift/cluster-api-provider-gcp => github.com/openshift/cluster-api-provider-gcp v0.0.1-0.20200917183408-90e92ed9fd9b
	github.com/openshift/cluster-api-provider-libvirt => github.com/openshift/cluster-api-provider-libvirt v0.2.1-0.20201023191903-499c4452f1f7
	github.com/openshift/cluster-api-provider-ovirt => github.com/openshift/cluster-api-provider-ovirt v0.1.1-0.20200917104433-85701abcd927
	github.com/openshift/console-operator => github.com/openshift/console-operator v0.0.0-20201028233916-50e817c3eabc
	github.com/openshift/installer => github.com/jim-minter/installer v0.9.0-master.0.20201116195329-73508e766328
	github.com/openshift/machine-api-operator => github.com/openshift/machine-api-operator v0.2.1-0.20201009151430-0af747bec740
	github.com/openshift/machine-config-operator => github.com/openshift/machine-config-operator v0.0.1-0.20201106225516-bc4ece5c0409
	github.com/operator-framework/operator-sdk => github.com/operator-framework/operator-sdk v0.19.4
	k8s.io/api => k8s.io/api v0.19.4
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.19.4
	k8s.io/apimachinery => k8s.io/apimachinery v0.19.4
	k8s.io/client-go => k8s.io/client-go v0.19.4
	k8s.io/code-generator => k8s.io/code-generator v0.19.4
	k8s.io/kube-openapi => k8s.io/kube-openapi v0.0.0-20200410145947-61e04a5be9a6
	k8s.io/kubectl => k8s.io/kubectl v0.19.4
	sigs.k8s.io/cluster-api-provider-aws => github.com/openshift/cluster-api-provider-aws v0.2.1-0.20200911195425-2710ded1034b
	sigs.k8s.io/cluster-api-provider-azure => github.com/openshift/cluster-api-provider-azure v0.1.0-alpha.3.0.20201021230208-6a32d86775de
	sigs.k8s.io/cluster-api-provider-gcp => github.com/openshift/cluster-api-provider-gcp v0.0.1-0.20200917183408-90e92ed9fd9b
	sigs.k8s.io/cluster-api-provider-openstack => github.com/openshift/cluster-api-provider-openstack v0.0.0-20200917103425-f6733e6dec7b
)
