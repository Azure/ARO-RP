module github.com/Azure/ARO-RP

go 1.18

require (
	github.com/Azure/azure-sdk-for-go v63.1.0+incompatible
	github.com/Azure/azure-sdk-for-go/sdk/azcore v1.7.1
	github.com/Azure/azure-sdk-for-go/sdk/azidentity v1.3.1
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v2 v2.2.1
	github.com/Azure/go-autorest/autorest v0.11.29
	github.com/Azure/go-autorest/autorest/adal v0.9.23
	github.com/Azure/go-autorest/autorest/date v0.3.0
	github.com/Azure/go-autorest/autorest/to v0.4.0
	github.com/Azure/go-autorest/autorest/validation v0.3.1
	github.com/Azure/go-autorest/tracing v0.6.0
	github.com/alvaroloes/enumer v1.1.2
	github.com/apparentlymart/go-cidr v1.1.0
	github.com/codahale/etm v0.0.0-20141003032925-c00c9e6fb4c9
	github.com/containers/image/v5 v5.24.1
	github.com/containers/podman/v4 v4.4.2
	github.com/coreos/go-oidc v2.2.1+incompatible
	github.com/coreos/go-semver v0.3.0
	github.com/coreos/go-systemd/v22 v22.5.0
	github.com/coreos/ignition/v2 v2.14.0
	github.com/davecgh/go-spew v1.1.1
	github.com/form3tech-oss/jwt-go v3.2.5+incompatible
	github.com/ghodss/yaml v1.0.1-0.20190212211648-25d852aebe32
	github.com/go-bindata/go-bindata v3.1.2+incompatible
	github.com/go-chi/chi/v5 v5.0.8
	github.com/go-logr/logr v1.2.4
	github.com/go-test/deep v1.1.0
	github.com/gofrs/uuid v4.2.0+incompatible
	github.com/golang-jwt/jwt/v4 v4.5.0
	github.com/golang/mock v1.6.0
	github.com/google/gnostic v0.5.7-v3refs
	github.com/google/go-cmp v0.5.9
	github.com/google/uuid v1.3.0
	github.com/gorilla/csrf v1.7.1
	github.com/gorilla/mux v1.8.0
	github.com/gorilla/securecookie v1.1.1
	github.com/gorilla/sessions v1.2.1
	github.com/itchyny/gojq v0.12.13
	github.com/jewzaam/go-cosmosdb v0.0.0-20230924011506-8f8942a01991
	github.com/jongio/azidext/go/azidext v0.5.0
	github.com/microsoft/kiota-abstractions-go v1.2.0
	github.com/microsoft/kiota-http-go v1.0.0
	github.com/microsoft/kiota-serialization-form-go v1.0.0
	github.com/microsoft/kiota-serialization-json-go v1.0.4
	github.com/microsoft/kiota-serialization-multipart-go v1.0.0
	github.com/microsoft/kiota-serialization-text-go v1.0.0
	github.com/microsoftgraph/msgraph-sdk-go-core v1.0.0
	github.com/onsi/ginkgo/v2 v2.7.0
	github.com/onsi/gomega v1.26.0
	github.com/open-policy-agent/frameworks/constraint v0.0.0-20221109005544-7de84dff5081
	github.com/opencontainers/runtime-spec v1.0.3-0.20220825212826-86290f6a00fb
	github.com/openshift/api v3.9.1-0.20191111211345-a27ff30ebf09+incompatible
	github.com/openshift/client-go v0.0.0-20220525160904-9e1acff93e4a
	github.com/openshift/console-operator v0.0.0-20220407014945-45d37e70e0c2
	github.com/openshift/hive/apis v0.0.0
	github.com/openshift/library-go v0.0.0-20220525173854-9b950a41acdc
	github.com/openshift/machine-config-operator v0.0.1-0.20230519222939-1abc13efbb0d
	github.com/pires/go-proxyproto v0.6.2
	github.com/pkg/errors v0.9.1
	github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring v0.50.0
	github.com/prometheus-operator/prometheus-operator/pkg/client v0.48.1
	github.com/prometheus/client_golang v1.15.1
	github.com/prometheus/common v0.42.0
	github.com/serge1peshcoff/selenium-go-conditions v0.0.0-20170824121757-5afbdb74596b
	github.com/sirupsen/logrus v1.9.0
	github.com/stretchr/testify v1.8.4
	github.com/tebeka/selenium v0.9.9
	github.com/ugorji/go/codec v1.2.7
	github.com/vincent-petithory/dataurl v1.0.0
	golang.org/x/crypto v0.14.0
	golang.org/x/net v0.17.0
	golang.org/x/oauth2 v0.7.0
	golang.org/x/sync v0.2.0
	golang.org/x/text v0.13.0
	golang.org/x/tools v0.7.0
	k8s.io/api v0.26.2
	k8s.io/apiextensions-apiserver v0.24.17
	k8s.io/apimachinery v0.26.2
	k8s.io/cli-runtime v0.24.17
	k8s.io/client-go v0.24.17
	k8s.io/code-generator v0.24.17
	k8s.io/kubectl v0.24.17
	k8s.io/kubernetes v1.24.17
	k8s.io/utils v0.0.0-20230726121419-3b25d923346b
	sigs.k8s.io/cluster-api-provider-azure v1.2.1
	sigs.k8s.io/controller-runtime v0.13.1
	sigs.k8s.io/controller-tools v0.9.0
)

require (
	github.com/Azure/azure-sdk-for-go/sdk/internal v1.3.0 // indirect
	github.com/Azure/go-ansiterm v0.0.0-20210617225240-d185dfc1b5a1 // indirect
	github.com/Azure/go-autorest v14.2.0+incompatible // indirect
	github.com/Azure/go-autorest/logger v0.2.1 // indirect
	github.com/AzureAD/microsoft-authentication-library-for-go v1.1.1 // indirect
	github.com/BurntSushi/toml v1.2.1 // indirect
	github.com/BurntSushi/xgb v0.0.0-20210121224620-deaf085860bc // indirect
	github.com/MakeNowJust/heredoc v1.0.0 // indirect
	github.com/Microsoft/go-winio v0.6.0 // indirect
	github.com/Microsoft/hcsshim v0.9.6 // indirect
	github.com/VividCortex/ewma v1.2.0 // indirect
	github.com/acarl005/stripansi v0.0.0-20180116102854-5a71ef0e047d // indirect
	github.com/antlr/antlr4/runtime/Go/antlr v0.0.0-20210826220005-b48c857c3a0e // indirect
	github.com/asaskevich/govalidator v0.0.0-20230301143203-a9d515a09cc2 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/blang/semver v3.5.1+incompatible // indirect
	github.com/blang/semver/v4 v4.0.0 // indirect
	github.com/cespare/xxhash/v2 v2.2.0 // indirect
	github.com/chai2010/gettext-go v0.0.0-20160711120539-c6fed771bfd5 // indirect
	github.com/chzyer/readline v1.5.1 // indirect
	github.com/cilium/ebpf v0.7.0 // indirect
	github.com/cjlapao/common-go v0.0.39 // indirect
	github.com/container-orchestrated-devices/container-device-interface v0.5.3 // indirect
	github.com/containerd/cgroups v1.0.4 // indirect
	github.com/containerd/containerd v1.6.18 // indirect
	github.com/containerd/stargz-snapshotter/estargz v0.14.3 // indirect
	github.com/containers/buildah v1.29.0 // indirect
	github.com/containers/common v0.51.0 // indirect
	github.com/containers/libtrust v0.0.0-20230121012942-c1716e8a8d01 // indirect
	github.com/containers/ocicrypt v1.1.7 // indirect
	github.com/containers/psgo v1.8.0 // indirect
	github.com/containers/storage v1.45.3 // indirect
	github.com/coreos/vcontext v0.0.0-20211021162308-f1dbbca7bef4 // indirect
	github.com/creack/pty v1.1.17 // indirect
	github.com/cyberphone/json-canonicalization v0.0.0-20220623050100-57a0ce2678a7 // indirect
	github.com/cyphar/filepath-securejoin v0.2.4 // indirect
	github.com/disiqueira/gotree/v3 v3.0.2 // indirect
	github.com/docker/distribution v2.8.2+incompatible // indirect
	github.com/docker/docker v24.0.7+incompatible // indirect
	github.com/docker/docker-credential-helpers v0.7.0 // indirect
	github.com/docker/go-connections v0.4.1-0.20210727194412-58542c764a11 // indirect
	github.com/docker/go-units v0.5.0 // indirect
	github.com/emicklei/go-restful/v3 v3.8.0 // indirect
	github.com/evanphx/json-patch v5.6.0+incompatible // indirect
	github.com/exponent-io/jsonpath v0.0.0-20210407135951-1de76d718b3f // indirect
	github.com/fatih/color v1.14.1 // indirect
	github.com/fsnotify/fsnotify v1.6.0 // indirect
	github.com/go-errors/errors v1.4.2 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-openapi/analysis v0.21.4 // indirect
	github.com/go-openapi/errors v0.20.3 // indirect
	github.com/go-openapi/jsonpointer v0.19.5 // indirect
	github.com/go-openapi/jsonreference v0.20.0 // indirect
	github.com/go-openapi/loads v0.21.2 // indirect
	github.com/go-openapi/runtime v0.26.0 // indirect
	github.com/go-openapi/spec v0.20.9 // indirect
	github.com/go-openapi/strfmt v0.21.7 // indirect
	github.com/go-openapi/swag v0.22.3 // indirect
	github.com/go-openapi/validate v0.22.1 // indirect
	github.com/go-stack/stack v1.8.1 // indirect
	github.com/go-task/slim-sprig v0.0.0-20210107165309-348f09dbbbc0 // indirect
	github.com/gobuffalo/flect v0.2.5 // indirect
	github.com/godbus/dbus/v5 v5.1.1-0.20221029134443-4b691ce883d5 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang-jwt/jwt/v5 v5.0.0 // indirect
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da // indirect
	github.com/golang/protobuf v1.5.3 // indirect
	github.com/google/btree v1.0.1 // indirect
	github.com/google/cel-go v0.10.4 // indirect
	github.com/google/go-containerregistry v0.14.0 // indirect
	github.com/google/go-intervals v0.0.2 // indirect
	github.com/google/gofuzz v1.2.0 // indirect
	github.com/google/pprof v0.0.0-20210720184732-4bb14d4b1be1 // indirect
	github.com/google/shlex v0.0.0-20191202100458-e7afc7fbc510 // indirect
	github.com/googleapis/gnostic v0.6.8 // indirect
	github.com/gorilla/schema v1.2.0 // indirect
	github.com/gregjones/httpcache v0.0.0-20190611155906-901d90724c79 // indirect
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/hashicorp/go-multierror v1.1.1 // indirect
	github.com/imdario/mergo v0.3.13 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/itchyny/timefmt-go v0.1.5 // indirect
	github.com/jinzhu/copier v0.3.5 // indirect
	github.com/josharian/intern v1.0.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/klauspost/compress v1.16.0 // indirect
	github.com/klauspost/pgzip v1.2.6-0.20220930104621-17e8dac29df8 // indirect
	github.com/kr/fs v0.1.0 // indirect
	github.com/kylelemons/godebug v1.1.0 // indirect
	github.com/letsencrypt/boulder v0.0.0-20221109233200-85aa52084eaf // indirect
	github.com/liggitt/tabwriter v0.0.0-20181228230101-89fcab3d43de // indirect
	github.com/mailru/easyjson v0.7.7 // indirect
	github.com/manifoldco/promptui v0.9.0 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.19 // indirect
	github.com/mattn/go-runewidth v0.0.14 // indirect
	github.com/mattn/go-shellwords v1.0.12 // indirect
	github.com/matttproud/golang_protobuf_extensions v1.0.4 // indirect
	github.com/microsoft/kiota-authentication-azure-go v1.0.0 // indirect
	github.com/miekg/pkcs11 v1.1.1 // indirect
	github.com/mistifyio/go-zfs/v3 v3.0.0 // indirect
	github.com/mitchellh/go-wordwrap v1.0.1 // indirect
	github.com/mitchellh/mapstructure v1.5.0 // indirect
	github.com/moby/spdystream v0.2.0 // indirect
	github.com/moby/sys/mountinfo v0.6.2 // indirect
	github.com/moby/term v0.0.0-20210619224110-3f7ff695adc6 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/monochromegane/go-gitignore v0.0.0-20200626010858-205db1a8cc00 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/nxadm/tail v1.4.8 // indirect
	github.com/oklog/ulid v1.3.1 // indirect
	github.com/opencontainers/go-digest v1.0.0 // indirect
	github.com/opencontainers/image-spec v1.1.0-rc2 // indirect
	github.com/opencontainers/runc v1.1.6 // indirect
	github.com/opencontainers/runtime-tools v0.9.1-0.20221014010322-58c91d646d86 // indirect
	github.com/opencontainers/selinux v1.10.2 // indirect
	github.com/openshift/custom-resource-status v1.1.3-0.20220503160415-f2fdb4999d87 // indirect
	github.com/ostreedev/ostree-go v0.0.0-20210805093236-719684c64e4f // indirect
	github.com/pascaldekloe/name v0.0.0-20180628100202-0fd16699aae1 // indirect
	github.com/peterbourgon/diskv v2.0.1+incompatible // indirect
	github.com/pkg/browser v0.0.0-20210911075715-681adbf594b8 // indirect
	github.com/pkg/sftp v1.13.5 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/pquerna/cachecontrol v0.1.0 // indirect
	github.com/proglottis/gpgme v0.1.3 // indirect
	github.com/prometheus/client_model v0.4.0 // indirect
	github.com/prometheus/procfs v0.9.0 // indirect
	github.com/rivo/uniseg v0.4.4 // indirect
	github.com/robfig/cron v1.2.0 // indirect
	github.com/rogpeppe/go-internal v1.9.0 // indirect
	github.com/russross/blackfriday v1.6.0 // indirect
	github.com/sigstore/fulcio v1.0.0 // indirect
	github.com/sigstore/rekor v1.2.0 // indirect
	github.com/sigstore/sigstore v1.6.4 // indirect
	github.com/spf13/cobra v1.7.0 // indirect
	github.com/spf13/pflag v1.0.6-0.20210604193023-d5e0c0615ace // indirect
	github.com/stefanberger/go-pkcs11uri v0.0.0-20201008174630-78d3cae3a980 // indirect
	github.com/stoewer/go-strcase v1.2.0 // indirect
	github.com/sylabs/sif/v2 v2.9.0 // indirect
	github.com/syndtr/gocapability v0.0.0-20200815063812-42c35b437635 // indirect
	github.com/tchap/go-patricia v2.3.0+incompatible // indirect
	github.com/theupdateframework/go-tuf v0.5.2 // indirect
	github.com/titanous/rocacheck v0.0.0-20171023193734-afe73141d399 // indirect
	github.com/ulikunitz/xz v0.5.11 // indirect
	github.com/vbatts/tar-split v0.11.2 // indirect
	github.com/vbauerster/mpb/v7 v7.5.3 // indirect
	github.com/xeipuuv/gojsonpointer v0.0.0-20190905194746-02993c407bfb // indirect
	github.com/xeipuuv/gojsonreference v0.0.0-20180127040603-bd5ef7bd5415 // indirect
	github.com/xeipuuv/gojsonschema v1.2.0 // indirect
	github.com/xlab/treeprint v1.1.0 // indirect
	github.com/yosida95/uritemplate/v3 v3.0.2 // indirect
	go.etcd.io/bbolt v1.3.6 // indirect
	go.mongodb.org/mongo-driver v1.11.3 // indirect
	go.mozilla.org/pkcs7 v0.0.0-20210826202110-33d05740a352 // indirect
	go.opencensus.io v0.24.0 // indirect
	go.opentelemetry.io/otel v1.16.0 // indirect
	go.opentelemetry.io/otel/metric v1.16.0 // indirect
	go.opentelemetry.io/otel/trace v1.16.0 // indirect
	go.starlark.net v0.0.0-20220328144851-d1966c6b9fcd // indirect
	golang.org/x/mod v0.10.0 // indirect
	golang.org/x/sys v0.13.0 // indirect
	golang.org/x/term v0.13.0 // indirect
	golang.org/x/time v0.3.0 // indirect
	gomodules.xyz/jsonpatch/v2 v2.2.0 // indirect
	google.golang.org/appengine v1.6.7 // indirect
	google.golang.org/genproto v0.0.0-20230410155749-daa745c078e1 // indirect
	google.golang.org/grpc v1.55.0 // indirect
	google.golang.org/protobuf v1.30.0 // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
	gopkg.in/square/go-jose.v2 v2.6.0 // indirect
	gopkg.in/tomb.v1 v1.0.0-20141024135613-dd632973f1e7 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	k8s.io/apiserver v0.24.17 // indirect
	k8s.io/component-base v0.25.0 // indirect
	k8s.io/gengo v0.0.0-20211129171323-c02415ce4185 // indirect
	k8s.io/klog/v2 v2.100.1 // indirect
	k8s.io/kube-aggregator v0.24.1 // indirect
	k8s.io/kube-openapi v0.0.0-20220803162953-67bda5d908f1 // indirect
	sigs.k8s.io/json v0.0.0-20221116044647-bc3834ca7abd // indirect
	sigs.k8s.io/kube-storage-version-migrator v0.0.4 // indirect
	sigs.k8s.io/kustomize/api v0.11.4 // indirect
	sigs.k8s.io/kustomize/kyaml v0.13.6 // indirect
	sigs.k8s.io/structured-merge-diff/v4 v4.2.3 // indirect
	sigs.k8s.io/yaml v1.3.0 // indirect
)

exclude (
	// trim dependency tree from old googlecloud/go
	cloud.google.com/go v0.37.4
	cloud.google.com/go v0.41.0
	cloud.google.com/go v0.44.1
	cloud.google.com/go v0.44.2
	cloud.google.com/go v0.45.1
	cloud.google.com/go v0.46.3
	cloud.google.com/go v0.50.0
	cloud.google.com/go v0.52.0
	cloud.google.com/go v0.53.0
	cloud.google.com/go v0.54.0
	cloud.google.com/go v0.56.0
	cloud.google.com/go v0.57.0
	// trim dependency tree from old googlecloud/firestore
	cloud.google.com/go/firestore v1.1.0
	// trim dependency tree from old google/go/storage
	cloud.google.com/go/storage v1.0.0
	cloud.google.com/go/storage v1.5.0
	cloud.google.com/go/storage v1.6.0
	cloud.google.com/go/storage v1.8.0
	// exclude Azure SDKs that we are not compatible with
	github.com/Azure/azure-sdk-for-go v48.0.0+incompatible
	github.com/Azure/azure-sdk-for-go v55.0.0+incompatible
	github.com/Azure/azure-sdk-for-go v63.2.0+incompatible
	github.com/Azure/azure-sdk-for-go v63.3.0+incompatible
	github.com/Azure/azure-sdk-for-go v63.4.0+incompatible
	github.com/Azure/azure-sdk-for-go v64.0.0+incompatible
	github.com/Azure/azure-sdk-for-go v64.1.0+incompatible
	github.com/Azure/azure-sdk-for-go v64.2.0+incompatible
	github.com/Azure/azure-sdk-for-go v65.0.0+incompatible
	github.com/Azure/azure-sdk-for-go v66.0.0+incompatible
	github.com/Azure/azure-sdk-for-go v67.0.0+incompatible
	github.com/Azure/azure-sdk-for-go v67.1.0+incompatible
	github.com/Azure/azure-sdk-for-go v67.2.0+incompatible
	github.com/Azure/azure-sdk-for-go v67.3.0+incompatible
	github.com/Azure/azure-sdk-for-go v67.4.0+incompatible
	github.com/Azure/azure-sdk-for-go v68.0.0+incompatible
	github.com/Azure/go-ansiterm v0.0.0-20170929234023-d6e3b3328b78
	github.com/Azure/go-ansiterm v0.0.0-20210608223527-2377c96fe795
	github.com/Azure/go-autorest/autorest v0.11.12
	github.com/Azure/go-autorest/autorest v0.11.18
	github.com/Azure/go-autorest/autorest v0.11.24
	github.com/Azure/go-autorest/autorest v0.9.0
	github.com/Azure/go-autorest/autorest/adal v0.9.13
	github.com/Azure/go-autorest/autorest/adal v0.9.18
	github.com/Azure/go-autorest/autorest/adal v0.9.5
	github.com/Azure/go-autorest/autorest/mocks v0.4.1
	github.com/Azure/go-autorest/autorest/to v0.3.0
	github.com/Azure/go-autorest/autorest/validation v0.1.0
	// exclude github.com/containerd/containerd < 1.6.1, 1.5.10, 1.14.12 https://nvd.nist.gov/vuln/detail/CVE-2022-23648
	github.com/containerd/containerd v1.2.10
	github.com/containerd/containerd v1.2.7
	github.com/containerd/containerd v1.3.0
	github.com/containerd/containerd v1.3.2
	github.com/containerd/containerd v1.4.1
	github.com/containerd/containerd v1.4.3
	github.com/containerd/containerd v1.4.4
	github.com/containerd/containerd v1.4.9
	github.com/containerd/containerd v1.5.0-beta.1
	github.com/containerd/containerd v1.5.0-beta.3
	github.com/containerd/containerd v1.5.0-beta.4
	github.com/containerd/containerd v1.5.0-rc.0
	github.com/containerd/containerd v1.5.1
	github.com/containerd/containerd v1.5.2
	github.com/containerd/containerd v1.5.7
	github.com/containerd/containerd v1.5.9
	// force use of go.etcd.io/bbolt
	github.com/coreos/bbolt v1.3.0
	github.com/coreos/bbolt v1.3.2
	github.com/coreos/bbolt v1.3.3
	// remove ancient dockers
	github.com/docker/distribution v0.0.0-20180920194744-16128bbac47f
	github.com/docker/distribution v0.0.0-20191216044856-a8371794149d
	github.com/docker/distribution v2.7.0+incompatible
	github.com/docker/distribution v2.7.1+incompatible
	// force use of go.etcd.io/bbolt
	github.com/etcd-io/bbolt v1.3.3
	github.com/etcd-io/bbolt v1.3.6
	// exclude github.com/golang/protobuf < 1.3.2 https://nvd.nist.gov/vuln/detail/CVE-2021-3121
	github.com/gogo/protobuf v1.0.0
	github.com/gogo/protobuf v1.1.1
	github.com/gogo/protobuf v1.2.0
	github.com/gogo/protobuf v1.2.1
	github.com/gogo/protobuf v1.3.0
	github.com/gogo/protobuf v1.3.1
	// force use of golang.org/x/lint
	github.com/golang/lint v0.0.0-20180702182130-06c8688daad7
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
	// force use of github.com/envoyproxy/protoc-gen-validate
	github.com/lyft/protoc-gen-validate v0.0.13
	// busted install path
	github.com/mikefarah/yaml/v2 v2.4.0
	// exclude old openshift library-go
	github.com/openshift/library-go v0.0.0-20211220195323-eca2c467c492
	github.com/openshift/library-go v0.0.0-20220121154930-b7889002d63e
	// Enable after installer is removed
	//github.com/openshift/library-go v0.0.0-20220525173854-9b950a41acdc
	// no 3.11
	github.com/openshift/machine-config-operator v3.11.0+incompatible
	// trip dependency tree from old prometheus common
	github.com/prometheus/common v0.10.0
	github.com/prometheus/common v0.15.0
	github.com/prometheus/common v0.26.0
	// https://www.whitesourcesoftware.com/vulnerability-database/WS-2018-0594
	github.com/satori/go.uuid v0.0.0
	github.com/satori/uuid v0.0.0
	// trip dependency tree from old cobra
	github.com/spf13/cobra v0.0.2-0.20171109065643-2da4a54c5cee
	github.com/spf13/cobra v0.0.3
	github.com/spf13/cobra v0.0.5
	github.com/spf13/cobra v1.0.0
	github.com/spf13/cobra v1.1.1
	github.com/spf13/cobra v1.1.3
	github.com/spf13/cobra v1.2.1
	go.etcd.io/bbolt v1.3.2
	go.etcd.io/bbolt v1.3.3
	go.etcd.io/bbolt v1.3.5
	// Enable after installer is removed
	//go.etcd.io/bbolt v1.3.6
	// trim dependency tree from old etcd
	go.etcd.io/etcd v0.0.0-20191023171146-3cf2f69b5738
	// trim dependency tree from old opencensus
	go.opencensus.io v0.20.1
	go.opencensus.io v0.20.2
	go.opencensus.io v0.21.0
	go.opencensus.io v0.22.0
	go.opencensus.io v0.22.2
	go.opencensus.io v0.22.3
	go.opencensus.io v0.22.4
	go.opencensus.io v0.22.5
	//go.opencensus.io v0.23.0
	// trim dependency tree from old oauth2s
	golang.org/x/oauth2 v0.0.0-20180821212333-d2e6202438be
	golang.org/x/oauth2 v0.0.0-20190226205417-e64efc72b421
	golang.org/x/oauth2 v0.0.0-20190402181905-9f3314589c9a
	golang.org/x/oauth2 v0.0.0-20190604053449-0f29369cfe45
	golang.org/x/oauth2 v0.0.0-20191202225959-858c2ad4c8b6
	golang.org/x/oauth2 v0.0.0-20200107190931-bf48bf16ab8d
	golang.org/x/oauth2 v0.0.0-20200902213428-5d25da1a8d43
	golang.org/x/oauth2 v0.0.0-20201109201403-9fd604954f58
	golang.org/x/oauth2 v0.0.0-20201208152858-08078c50e5b5
	golang.org/x/oauth2 v0.0.0-20210218202405-ba52d332ba99
	golang.org/x/oauth2 v0.0.0-20210220000619-9bb904979d93
	golang.org/x/oauth2 v0.0.0-20210313182246-cd4f82c27b84
	golang.org/x/oauth2 v0.0.0-20210402161424-2e8d93401602
	golang.org/x/oauth2 v0.0.0-20210427180440-81ed05c6b58c
	golang.org/x/oauth2 v0.0.0-20210514164344-f6687ab2804c
	golang.org/x/oauth2 v0.0.0-20210628180205-a41e5a781914
	golang.org/x/oauth2 v0.0.0-20210805134026-6f1e6394065a
	golang.org/x/oauth2 v0.0.0-20210819190943-2bc19b11175f
	golang.org/x/oauth2 v0.0.0-20211005180243-6b3c2da341f1
	golang.org/x/oauth2 v0.0.0-20211104180415-d3ed0bb246c8
	// don't import google api
	google.golang.org/api v0.13.0
	google.golang.org/api v0.14.0
	google.golang.org/api v0.15.0
	google.golang.org/api v0.17.0
	google.golang.org/api v0.18.0
	google.golang.org/api v0.19.0
	google.golang.org/api v0.20.0
	google.golang.org/api v0.22.0
	google.golang.org/api v0.24.0
	google.golang.org/api v0.28.0
	google.golang.org/api v0.29.0
	google.golang.org/api v0.3.1
	google.golang.org/api v0.3.2
	google.golang.org/api v0.30.0
	google.golang.org/api v0.35.0
	google.golang.org/api v0.36.0
	google.golang.org/api v0.4.0
	google.golang.org/api v0.40.0
	google.golang.org/api v0.41.0
	google.golang.org/api v0.43.0
	google.golang.org/api v0.44.0
	google.golang.org/api v0.46.0
	google.golang.org/api v0.47.0
	google.golang.org/api v0.48.0
	google.golang.org/api v0.50.0
	google.golang.org/api v0.51.0
	google.golang.org/api v0.54.0
	google.golang.org/api v0.55.0
	google.golang.org/api v0.56.0
	google.golang.org/api v0.57.0
	google.golang.org/api v0.59.0
	google.golang.org/api v0.61.0
	google.golang.org/api v0.62.0
	google.golang.org/api v0.7.0
	google.golang.org/api v0.8.0
	google.golang.org/api v0.9.0
	// force use of cloud.google.com/go
	google.golang.org/cloud v0.0.0-20151119220103-975617b05ea8
	// trim dependency tree from old grpcs
	google.golang.org/grpc v1.17.0
	google.golang.org/grpc v1.19.0
	google.golang.org/grpc v1.20.0
	google.golang.org/grpc v1.20.1
	google.golang.org/grpc v1.21.0
	google.golang.org/grpc v1.21.1
	google.golang.org/grpc v1.22.1
	google.golang.org/grpc v1.23.1
	google.golang.org/grpc v1.24.0
	google.golang.org/grpc v1.25.1
	google.golang.org/grpc v1.26.0
	google.golang.org/grpc v1.27.0
	google.golang.org/grpc v1.27.1
	google.golang.org/grpc v1.28.0
	google.golang.org/grpc v1.29.1
	// trim dependency tree from old protobufs
	google.golang.org/protobuf v0.0.0-20200109180630-ec00e32a8dfd
	google.golang.org/protobuf v0.0.0-20200221191635-4d8936d0db64
	google.golang.org/protobuf v0.0.0-20200228230310-ab0ca4ff8a60
	google.golang.org/protobuf v1.20.1-0.20200309200217-e05f789c0967
	google.golang.org/protobuf v1.21.0
	google.golang.org/protobuf v1.22.0
	google.golang.org/protobuf v1.23.0
	google.golang.org/protobuf v1.23.1-0.20200526195155-81db48ad09cc
	google.golang.org/protobuf v1.24.0
	google.golang.org/protobuf v1.25.0
	google.golang.org/protobuf v1.26.0
	google.golang.org/protobuf v1.26.0-rc.1

)

// exclude ancient k8s versions
exclude (
	k8s.io/api v0.0.0
	k8s.io/api v0.18.0-beta.2
	k8s.io/api v0.18.3
	k8s.io/api v0.19.2
	k8s.io/api v0.19.3
	k8s.io/api v0.19.4
	k8s.io/api v0.20.0
	k8s.io/api v0.20.6
	k8s.io/api v0.21.0
	k8s.io/api v0.21.1
	k8s.io/api v0.22.1
	//k8s.io/api v0.23.0
	k8s.io/api v0.23.1
	k8s.io/apiextensions-apiserver v0.0.0
	k8s.io/apiextensions-apiserver v0.18.0-beta.2
	k8s.io/apiextensions-apiserver v0.18.3
	k8s.io/apiextensions-apiserver v0.19.2
	k8s.io/apiextensions-apiserver v0.19.3
	k8s.io/apiextensions-apiserver v0.21.0
	k8s.io/apiextensions-apiserver v0.21.1
	k8s.io/apiextensions-apiserver v0.22.1
	//k8s.io/apiextensions-apiserver v0.23.0
	k8s.io/apiextensions-apiserver v0.23.1
	k8s.io/apiextensions-apiserver v0.23.5
	k8s.io/apimachinery v0.0.0
	k8s.io/apimachinery v0.18.0-beta.2
	k8s.io/apimachinery v0.18.3
	k8s.io/apimachinery v0.19.2
	k8s.io/apimachinery v0.19.3
	k8s.io/apimachinery v0.19.4
	k8s.io/apimachinery v0.20.0
	k8s.io/apimachinery v0.20.2
	k8s.io/apimachinery v0.20.6
	k8s.io/apimachinery v0.21.0
	k8s.io/apimachinery v0.21.1
	k8s.io/apimachinery v0.22.1
	//k8s.io/apimachinery v0.23.0
	k8s.io/apimachinery v0.23.1
	k8s.io/apimachinery v0.23.5
	k8s.io/apiserver v0.0.0
	k8s.io/apiserver v0.20.6
	k8s.io/apiserver v0.21.0
	k8s.io/apiserver v0.22.1
	//k8s.io/apiserver v0.23.0
	k8s.io/apiserver v0.23.1
	k8s.io/apiserver v0.23.5
	k8s.io/cli-runtime v0.0.0
	k8s.io/cli-runtime v0.21.0
	//k8s.io/cli-runtime v0.23.0
	k8s.io/cli-runtime v0.23.1
	k8s.io/client-go v0.0.0
	k8s.io/client-go v0.18.0-beta.2
	k8s.io/client-go v0.19.2
	k8s.io/client-go v0.19.3
	k8s.io/client-go v0.19.4
	k8s.io/client-go v0.20.0
	k8s.io/client-go v0.20.6
	k8s.io/client-go v0.21.0
	k8s.io/client-go v0.21.1
	k8s.io/client-go v0.22.1
	//k8s.io/client-go v0.23.0
	k8s.io/client-go v0.23.1
	k8s.io/client-go v0.23.5
	// k8s.io/cloud-provider v0.0.0
	k8s.io/code-generator v0.0.0
	k8s.io/code-generator v0.18.0-beta.2
	k8s.io/code-generator v0.19.7
	k8s.io/code-generator v0.20.0
	k8s.io/code-generator v0.21.0
	//k8s.io/code-generator v0.23.0
	k8s.io/component-base v0.0.0
	k8s.io/component-base v0.19.2
	k8s.io/component-base v0.19.4
	k8s.io/component-base v0.20.6
	k8s.io/component-base v0.21.0
	k8s.io/component-base v0.21.1
	k8s.io/component-base v0.22.1
	//k8s.io/component-base v0.23.0
	k8s.io/component-base v0.23.1
	k8s.io/component-base v0.23.5
	// k8s.io/component-helpers v0.0.0
	k8s.io/controller-manager v0.0.0
	k8s.io/cri-api v0.0.0
	k8s.io/cri-api v0.20.6
	// k8s.io/csi-translation-lib v0.0.0
	k8s.io/gengo v0.0.0-20201113003025-83324d819ded
	k8s.io/gengo v0.0.0-20210813121822-485abfe95c7c
	// Enable after installer is removed
	k8s.io/klog v1.0.0
	k8s.io/klog/v2 v2.0.0
	k8s.io/klog/v2 v2.2.0
	k8s.io/klog/v2 v2.30.0
	k8s.io/klog/v2 v2.4.0
	k8s.io/klog/v2 v2.60.1
	k8s.io/klog/v2 v2.8.0
	k8s.io/klog/v2 v2.9.0
	k8s.io/kube-aggregator v0.0.0
	k8s.io/kube-aggregator v0.18.0-beta.2
	k8s.io/kube-aggregator v0.23.0
	k8s.io/kube-openapi v0.0.0-20210421082810-95288971da7e
	k8s.io/kube-openapi v0.0.0-20211115234752-e816edb12b65
	k8s.io/kube-scheduler v0.0.0
	k8s.io/kubectl v0.0.0
	k8s.io/kubectl v0.21.0
	k8s.io/kubectl v0.22.0
	k8s.io/kubectl v0.23.0
	k8s.io/kubectl v0.23.1
	k8s.io/kubelet v0.0.0
	k8s.io/legacy-cloud-providers v0.0.0
	k8s.io/metrics v0.0.0
	k8s.io/mount-utils v0.0.0
	k8s.io/pod-security-admission v0.0.0
	k8s.io/sample-apiserver v0.0.0
	k8s.io/system-validators v1.6.0
	k8s.io/utils v0.0.0-20201110183641-67b214c5f920
	k8s.io/utils v0.0.0-20210111153108-fddb29f9d009
	k8s.io/utils v0.0.0-20210802155522-efc7438f0176
	k8s.io/utils v0.0.0-20210930125809-cb0fa318a74b
	k8s.io/utils v0.0.0-20211116205334-6203023598ed
	k8s.io/utils v0.0.0-20211208161948-7d6a63dca704
	k8s.io/utils v0.0.0-20220210201930-3a6ce19ff2f9
	sigs.k8s.io/apiserver-network-proxy/konnectivity-client v0.0.22
	sigs.k8s.io/apiserver-network-proxy/konnectivity-client v0.0.25
	sigs.k8s.io/apiserver-network-proxy/konnectivity-client v0.0.30
	sigs.k8s.io/controller-runtime v0.7.0
	sigs.k8s.io/controller-runtime v0.9.0
	sigs.k8s.io/controller-runtime v0.9.0-beta.1.0.20210512131817-ce2f0c92d77e
	sigs.k8s.io/controller-tools v0.2.8
	sigs.k8s.io/controller-tools v0.3.0
	sigs.k8s.io/controller-tools v0.4.1
	sigs.k8s.io/controller-tools v0.6.0
	sigs.k8s.io/controller-tools v0.6.2
	sigs.k8s.io/controller-tools v0.7.0
	sigs.k8s.io/kubebuilder/v3 v3.3.0
	sigs.k8s.io/kustomize/api v0.10.1
	sigs.k8s.io/kustomize/kyaml v0.10.21
	sigs.k8s.io/kustomize/kyaml v0.13.0
	sigs.k8s.io/structured-merge-diff/v4 v4.0.2
	sigs.k8s.io/structured-merge-diff/v4 v4.1.2
	sigs.k8s.io/structured-merge-diff/v4 v4.2.0
	sigs.k8s.io/structured-merge-diff/v4 v4.2.1
	sigs.k8s.io/yaml v1.2.0
)

replace (
	bitbucket.org/ww/goautoneg => github.com/munnerz/goautoneg v0.0.0-20120707110453-a547fc61f48d // 404 on bitbucket.org/ww/goautoneg
	// Replace old GoGo Protobuf versions https://nvd.nist.gov/vuln/detail/CVE-2021-3121
	github.com/gogo/protobuf => github.com/gogo/protobuf v1.3.2
	// https://www.whitesourcesoftware.com/vulnerability-database/WS-2018-0594
	github.com/satori/go.uuid => github.com/satori/go.uuid v1.2.1-0.20181028125025-b2ce2384e17b
	github.com/spf13/pflag => github.com/spf13/pflag v1.0.6-0.20210604193023-d5e0c0615ace
	k8s.io/api => k8s.io/api v0.24.17
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.24.17
	k8s.io/apimachinery => k8s.io/apimachinery v0.24.17
	k8s.io/apiserver => k8s.io/apiserver v0.24.17
	k8s.io/client-go => k8s.io/client-go v0.24.17
	k8s.io/code-generator => k8s.io/code-generator v0.24.17
	k8s.io/component-base => k8s.io/component-base v0.24.17
	k8s.io/kube-aggregator => k8s.io/kube-aggregator v0.24.17
	k8s.io/kube-controller-manager => k8s.io/kube-controller-manager v0.24.17
	k8s.io/kube-proxy => k8s.io/kube-proxy v0.24.17
	k8s.io/kube-scheduler => k8s.io/kube-scheduler v0.24.17
	k8s.io/kubectl => k8s.io/kubectl v0.24.17
	k8s.io/kubernetes => k8s.io/kubernetes v1.24.17
	sigs.k8s.io/controller-runtime => sigs.k8s.io/controller-runtime v0.11.2
	sigs.k8s.io/controller-tools => sigs.k8s.io/controller-tools v0.5.0
)

// Installer dependencies. Some of them are being used directly in the RP.
replace (
	github.com/BurntSushi/toml => github.com/BurntSushi/toml v0.3.1
	github.com/containers/image => github.com/containers/image v3.0.2+incompatible
	github.com/coreos/bbolt => go.etcd.io/bbolt v1.3.6
	github.com/coreos/prometheus-operator => github.com/prometheus-operator/prometheus-operator v0.48.1
	github.com/golang/lint => golang.org/x/lint v0.0.0-20210508222113-6edffad5e616
	github.com/googleapis/gnostic => github.com/google/gnostic v0.5.5
	github.com/openshift/api => github.com/openshift/api v0.0.0-20230426102702-398424d53f74
	github.com/openshift/client-go => github.com/openshift/client-go v0.0.0-20220603133046-984ee5ebedcf
	github.com/openshift/cloud-credential-operator => github.com/openshift/cloud-credential-operator v0.0.0-20200316201045-d10080b52c9e
	github.com/openshift/console-operator => github.com/openshift/console-operator v0.0.0-20220318130441-e44516b9c315
	github.com/openshift/library-go => github.com/openshift/library-go v0.0.0-20230222114049-eac44a078a6e
	github.com/openshift/machine-api-operator => github.com/openshift/machine-api-operator v0.2.1-0.20220124104622-668c5b52b104
	github.com/openshift/machine-config-operator => github.com/openshift/machine-config-operator v0.0.1-0.20220319215057-e6ba00b88555
	go.mongodb.org/mongo-driver => go.mongodb.org/mongo-driver v1.9.4
	google.golang.org/grpc => google.golang.org/grpc v1.40.0
	sigs.k8s.io/cluster-api-provider-azure => github.com/openshift/cluster-api-provider-azure v0.1.0-alpha.3.0.20210626224711-5d94c794092f
	sigs.k8s.io/kustomize/api => sigs.k8s.io/kustomize/api v0.11.2
	sigs.k8s.io/kustomize/kyaml => sigs.k8s.io/kustomize/kyaml v0.13.3
	sigs.k8s.io/structured-merge-diff => sigs.k8s.io/structured-merge-diff v1.0.1-0.20191108220359-b1b620dd3f06
)

replace github.com/openshift/hive/apis => github.com/openshift/hive/apis v0.0.0-20230811220652-70b666ec89b0
