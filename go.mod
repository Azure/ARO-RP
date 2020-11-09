module github.com/Azure/ARO-RP

go 1.14

exclude (
	github.com/openshift/cluster-api-provider-gcp v0.0.0-00010101000000-000000000000
	sigs.k8s.io/cluster-api-provider-aws v0.0.0
)

require (
	github.com/AlekSi/gocov-xml v0.0.0-20190121064608-3a14fb1c4737
	github.com/Azure/azure-sdk-for-go v48.0.0+incompatible
	github.com/Azure/go-autorest/autorest v0.11.10
	github.com/Azure/go-autorest/autorest/adal v0.9.5
	github.com/Azure/go-autorest/autorest/azure/auth v0.5.3
	github.com/Azure/go-autorest/autorest/date v0.3.0
	github.com/Azure/go-autorest/autorest/to v0.4.0
	github.com/Azure/go-autorest/autorest/validation v0.3.0
	github.com/Azure/go-autorest/tracing v0.6.0
	github.com/Microsoft/go-winio v0.4.15 // indirect
	github.com/ajeddeloh/go-json v0.0.0-20200220154158-5ae607161559 // indirect
	github.com/alvaroloes/enumer v1.1.2
	github.com/apparentlymart/go-cidr v1.1.0
	github.com/axw/gocov v1.0.0
	github.com/containers/image/v5 v5.7.0
	github.com/containers/libtrust v0.0.0-20200511145503-9c3a6c22cd9a // indirect
	github.com/containers/storage v1.23.9 // indirect
	github.com/coreos/go-systemd/v22 v22.1.0
	github.com/form3tech-oss/jwt-go v3.2.2+incompatible
	github.com/ghodss/yaml v1.0.1-0.20190212211648-25d852aebe32
	github.com/go-bindata/go-bindata v3.1.2+incompatible
	github.com/go-logr/logr v0.3.0
	github.com/go-test/deep v1.0.7
	github.com/golang/mock v1.4.4
	github.com/google/go-cmp v0.5.2
	github.com/google/gofuzz v1.2.0 // indirect
	github.com/google/uuid v1.1.2 // indirect
	github.com/googleapis/gnostic v0.5.3
	github.com/gorilla/mux v1.8.0
	github.com/gregjones/httpcache v0.0.0-20190611155906-901d90724c79 // indirect
	github.com/jim-minter/go-cosmosdb v0.0.0-20200923160222-1528d2db09d6
	github.com/jstemmer/go-junit-report v0.9.1
	github.com/mattn/go-colorable v0.1.8 // indirect
	github.com/mgutz/ansi v0.0.0-20200706080929-d51e80ef957d // indirect
	github.com/nxadm/tail v1.4.5 // indirect
	github.com/onsi/ginkgo v1.14.2
	github.com/onsi/gomega v1.10.3
	github.com/openshift/api v3.9.1-0.20191111211345-a27ff30ebf09+incompatible
	github.com/openshift/client-go v0.0.0-20200827190008-3062137373b5
	github.com/openshift/console-operator v0.0.0-20201027112558-54699a7c42f2
	github.com/openshift/installer v0.16.1
	github.com/openshift/machine-api-operator v0.2.1-0.20201002104344-6abfb5440597
	github.com/openshift/machine-config-operator v4.2.0-alpha.0.0.20190917115525-033375cbe820+incompatible
	github.com/operator-framework/operator-sdk v1.1.0
	github.com/pborman/uuid v1.2.1 // indirect
	github.com/prometheus/client_golang v1.8.0 // indirect
	github.com/prometheus/common v0.14.0
	github.com/satori/go.uuid v1.2.0
	github.com/sirupsen/logrus v1.7.0
	github.com/syndtr/gocapability v0.0.0-20200815063812-42c35b437635 // indirect
	github.com/ugorji/go/codec v1.1.13
	go4.org v0.0.0-20200411211856-f5505b9728dd // indirect
	golang.org/x/crypto v0.0.0-20201016220609-9e8e0b390897
	golang.org/x/net v0.0.0-20201031054903-ff519b6c9102 // indirect
	golang.org/x/oauth2 v0.0.0-20200902213428-5d25da1a8d43 // indirect
	golang.org/x/sync v0.0.0-20201020160332-67f06af15bc9
	golang.org/x/sys v0.0.0-20201101102859-da207088b7d1 // indirect
	golang.org/x/text v0.3.4 // indirect
	golang.org/x/time v0.0.0-20200630173020-3af7569d3a1e // indirect
	golang.org/x/tools v0.0.0-20201104183702-aefe0b742155
	gomodules.xyz/jsonpatch/v2 v2.1.0 // indirect
	google.golang.org/appengine v1.6.7 // indirect
	gopkg.in/square/go-jose.v2 v2.5.1 // indirect
	gopkg.in/yaml.v3 v3.0.0-20200615113413-eeeca48fe776 // indirect
	k8s.io/api v0.19.3
	k8s.io/apiextensions-apiserver v0.19.3
	k8s.io/apimachinery v0.19.3
	k8s.io/client-go v12.0.0+incompatible
	k8s.io/code-generator v0.19.0
	k8s.io/gengo v0.0.0-20200413195148-3a45101e95ac // indirect
	k8s.io/klog/v2 v2.4.0 // indirect
	k8s.io/kube-openapi v0.0.0-20201104192653-842b07581b16 // indirect
	k8s.io/utils v0.0.0-20201027101359-01387209bb0d // indirect
	sigs.k8s.io/cluster-api-provider-azure v0.4.9
	sigs.k8s.io/controller-runtime v0.6.3
	sigs.k8s.io/controller-tools v0.3.0
)

replace (
	bitbucket.org/ww/goautoneg => github.com/munnerz/goautoneg v0.0.0-20120707110453-a547fc61f48d // 404 on bitbucket.org/ww/goautoneg
	github.com/coreos/go-systemd => github.com/coreos/go-systemd/v22 v22.0.0
	github.com/googleapis/gnostic => github.com/googleapis/gnostic v0.4.0
	github.com/openshift/api => github.com/openshift/api v0.0.0-20200917102736-0a191b5b9bb0
	github.com/openshift/client-go => github.com/openshift/client-go v0.0.0-20200521150516-05eb9880269c
	github.com/openshift/cloud-credential-operator => github.com/openshift/cloud-credential-operator v0.0.0-20200926024851-4ef74fd4ae81
	github.com/openshift/cluster-api => github.com/openshift/cluster-api v0.0.0-20191129101638-b09907ac6668
	github.com/openshift/console-operator => github.com/openshift/console-operator v0.0.0-20201028233916-50e817c3eabc
	github.com/openshift/installer => github.com/jim-minter/installer v0.9.0-master.0.20201109033733-ad3ecc758669
	github.com/openshift/machine-api-operator => github.com/openshift/machine-api-operator v0.2.1-0.20201009151430-0af747bec740
	github.com/openshift/machine-config-operator => github.com/openshift/machine-config-operator v0.0.1-0.20201106225516-bc4ece5c0409
	github.com/operator-framework/operator-sdk => github.com/operator-framework/operator-sdk v0.19.4
	k8s.io/api => k8s.io/api v0.18.3
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.18.3
	k8s.io/apimachinery => k8s.io/apimachinery v0.18.3
	k8s.io/client-go => k8s.io/client-go v0.18.3
	k8s.io/code-generator => k8s.io/code-generator v0.18.3
	k8s.io/kube-openapi => k8s.io/kube-openapi v0.0.0-20200410145947-61e04a5be9a6
	k8s.io/kubectl => k8s.io/kubectl v0.18.3
	sigs.k8s.io/cluster-api-provider-azure => github.com/openshift/cluster-api-provider-azure v0.1.0-alpha.3.0.20201021230208-6a32d86775de
)
