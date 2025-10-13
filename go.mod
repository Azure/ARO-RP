module github.com/Azure/ARO-RP

go 1.24.4

require (
	github.com/Azure/ARO-RP/pkg/api v0.0.0-00010101000000-000000000000
	github.com/Azure/azure-sdk-for-go v63.1.0+incompatible
	github.com/Azure/azure-sdk-for-go/sdk/azcore v1.18.1
	github.com/Azure/azure-sdk-for-go/sdk/azidentity v1.10.1
	github.com/Azure/azure-sdk-for-go/sdk/containers/azcontainerregistry v0.2.3
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/authorization/armauthorization/v3 v3.0.0-beta.2
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/containerservice/armcontainerservice/v6 v6.6.0
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/cosmos/armcosmos/v2 v2.7.0
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/dns/armdns v1.2.0
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/keyvault/armkeyvault v1.5.0
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/msi/armmsi v1.3.0
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v6 v6.2.0
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/storage/armstorage v1.8.1
	github.com/Azure/azure-sdk-for-go/sdk/security/keyvault/azcertificates v1.4.0
	github.com/Azure/azure-sdk-for-go/sdk/security/keyvault/azsecrets v1.4.0
	github.com/Azure/azure-sdk-for-go/sdk/storage/azblob v1.6.1
	github.com/Azure/checkaccess-v2-go-sdk v0.0.3
	github.com/Azure/go-autorest/autorest v0.11.30
	github.com/Azure/go-autorest/autorest/adal v0.9.24
	github.com/Azure/go-autorest/autorest/date v0.3.1
	github.com/Azure/go-autorest/autorest/to v0.4.1
	github.com/Azure/go-autorest/autorest/validation v0.3.2
	github.com/Azure/go-autorest/tracing v0.6.1
	github.com/Azure/msi-dataplane v0.4.3
	github.com/apparentlymart/go-cidr v1.1.0
	github.com/codahale/etm v0.0.0-20141003032925-c00c9e6fb4c9
	github.com/containers/image/v5 v5.36.2
	github.com/containers/podman/v5 v5.6.1
	github.com/coreos/go-oidc/v3 v3.14.1
	github.com/coreos/go-semver v0.3.1
	github.com/coreos/go-systemd/v22 v22.5.1-0.20231103132048-7d375ecc2b09
	github.com/coreos/ignition/v2 v2.22.0
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc
	github.com/go-chi/chi/v5 v5.2.2
	github.com/go-jose/go-jose/v4 v4.1.1
	github.com/go-logr/logr v1.4.3
	github.com/go-test/deep v1.1.1
	github.com/gofrs/uuid v4.4.0+incompatible
	github.com/golang-jwt/jwt/v4 v4.5.2
	github.com/google/gnostic v0.5.7-v3refs
	github.com/google/go-cmp v0.7.0
	github.com/google/uuid v1.6.0
	github.com/gorilla/csrf v1.7.3
	github.com/gorilla/mux v1.8.1
	github.com/gorilla/securecookie v1.1.2
	github.com/gorilla/sessions v1.4.0
	github.com/hashicorp/go-multierror v1.1.1
	github.com/jongio/azidext/go/azidext v0.5.0
	github.com/microsoft/go-otel-audit v0.2.2
	github.com/microsoft/kiota-abstractions-go v1.9.3
	github.com/microsoft/kiota-http-go v1.5.4
	github.com/microsoft/kiota-serialization-form-go v1.1.2
	github.com/microsoft/kiota-serialization-json-go v1.1.2
	github.com/microsoft/kiota-serialization-multipart-go v1.1.2
	github.com/microsoft/kiota-serialization-text-go v1.1.2
	github.com/microsoftgraph/msgraph-sdk-go-core v1.3.2
	github.com/onsi/ginkgo/v2 v2.23.4
	github.com/onsi/gomega v1.38.0
	github.com/opencontainers/runtime-spec v1.2.1
	github.com/openshift/api v0.0.0-20240103200955-7ca3a4634e46
	github.com/openshift/client-go v0.0.0-20221019143426-16aed247da5c
	github.com/openshift/cloud-credential-operator v0.0.0-20240910012137-a0245d57d1e6
	github.com/openshift/hive/apis v0.0.0-20250612193659-8796c4f5340c
	github.com/openshift/library-go v0.0.0-20230620084201-504ca4bd5a83
	github.com/openshift/machine-config-operator v0.0.0-00010101000000-000000000000
	github.com/pires/go-proxyproto v0.8.1
	github.com/pkg/errors v0.9.1
	github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring v0.50.0
	github.com/prometheus-operator/prometheus-operator/pkg/client v0.48.1
	github.com/prometheus/client_golang v1.22.0
	github.com/prometheus/common v0.65.0
	github.com/serge1peshcoff/selenium-go-conditions v0.0.0-20170824121757-5afbdb74596b
	github.com/sirupsen/logrus v1.9.3
	github.com/spf13/cobra v1.9.1
	github.com/spf13/viper v1.20.1
	github.com/stretchr/testify v1.10.0
	github.com/tebeka/selenium v0.9.9
	github.com/ugorji/go/codec v1.2.14
	github.com/vincent-petithory/dataurl v1.0.0
	go.uber.org/mock v0.5.2
	golang.org/x/crypto v0.40.0
	golang.org/x/exp v0.0.0-20250718183923-645b1fa84792
	golang.org/x/net v0.42.0
	golang.org/x/oauth2 v0.30.0
	golang.org/x/sync v0.16.0
	golang.org/x/text v0.27.0
	golang.org/x/tools v0.35.0
	k8s.io/api v0.33.2
	k8s.io/apiextensions-apiserver v0.33.2
	k8s.io/apimachinery v0.33.2
	k8s.io/cli-runtime v0.25.16
	k8s.io/client-go v0.33.2
	k8s.io/kubectl v0.25.16
	k8s.io/kubernetes v1.33.2
	k8s.io/metrics v0.25.16
	k8s.io/utils v0.0.0-20250604170112-4c0f3b243397
	sigs.k8s.io/controller-runtime v0.21.0
	sigs.k8s.io/yaml v1.5.0
)

require (
	dario.cat/mergo v1.0.2 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/internal v1.11.1 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/security/keyvault/internal v1.2.0 // indirect
	github.com/Azure/go-ansiterm v0.0.0-20250102033503-faa5f7b0171c // indirect
	github.com/Azure/go-autorest v14.2.0+incompatible // indirect
	github.com/Azure/go-autorest/logger v0.2.2 // indirect
	github.com/Azure/retry v0.0.0-20250701224816-85c6a88f883d // indirect
	github.com/AzureAD/microsoft-authentication-library-for-go v1.4.2 // indirect
	github.com/BurntSushi/toml v1.5.0 // indirect
	github.com/BurntSushi/xgb v0.0.0-20210121224620-deaf085860bc // indirect
	github.com/MakeNowJust/heredoc v1.0.0 // indirect
	github.com/Microsoft/go-winio v0.6.2 // indirect
	github.com/Microsoft/hcsshim v0.13.0 // indirect
	github.com/VividCortex/ewma v1.2.0 // indirect
	github.com/acarl005/stripansi v0.0.0-20180116102854-5a71ef0e047d // indirect
	github.com/antlr4-go/antlr/v4 v4.13.1 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/blang/semver v3.5.1+incompatible // indirect
	github.com/blang/semver/v4 v4.0.0 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/chai2010/gettext-go v1.0.3 // indirect
	github.com/chzyer/readline v1.5.1 // indirect
	github.com/containerd/cgroups/v3 v3.0.5 // indirect
	github.com/containerd/errdefs v1.0.0 // indirect
	github.com/containerd/errdefs/pkg v0.3.0 // indirect
	github.com/containerd/log v0.1.0 // indirect
	github.com/containerd/platforms v1.0.0-rc.1 // indirect
	github.com/containerd/stargz-snapshotter/estargz v0.16.3 // indirect
	github.com/containerd/typeurl/v2 v2.2.3 // indirect
	github.com/containers/buildah v1.41.4 // indirect
	github.com/containers/common v0.64.2 // indirect
	github.com/containers/libtrust v0.0.0-20230121012942-c1716e8a8d01 // indirect
	github.com/containers/ocicrypt v1.2.1 // indirect
	github.com/containers/psgo v1.9.0 // indirect
	github.com/containers/storage v1.59.1 // indirect
	github.com/coreos/vcontext v0.0.0-20231102161604-685dc7299dc5 // indirect
	github.com/cyberphone/json-canonicalization v0.0.0-20241213102144-19d51d7fe467 // indirect
	github.com/cyphar/filepath-securejoin v0.4.1 // indirect
	github.com/disiqueira/gotree/v3 v3.0.2 // indirect
	github.com/distribution/reference v0.6.0 // indirect
	github.com/docker/distribution v2.8.3+incompatible // indirect
	github.com/docker/docker v28.3.3+incompatible // indirect
	github.com/docker/docker-credential-helpers v0.9.3 // indirect
	github.com/docker/go-connections v0.5.0 // indirect
	github.com/docker/go-units v0.5.0 // indirect
	github.com/elazarl/goproxy v1.7.2 // indirect
	github.com/emicklei/go-restful/v3 v3.12.2 // indirect
	github.com/evanphx/json-patch v5.6.0+incompatible // indirect
	github.com/evanphx/json-patch/v5 v5.6.0 // indirect
	github.com/exponent-io/jsonpath v0.0.0-20210407135951-1de76d718b3f // indirect
	github.com/felixge/httpsnoop v1.0.4 // indirect
	github.com/fsnotify/fsnotify v1.9.0 // indirect
	github.com/ghodss/yaml v1.0.1-0.20220118164431-d8423dcdf344 // indirect
	github.com/go-errors/errors v1.5.1 // indirect
	github.com/go-json-experiment/json v0.0.0-20250714165856-be8212f5270d // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-openapi/jsonpointer v0.21.1 // indirect
	github.com/go-openapi/jsonreference v0.21.0 // indirect
	github.com/go-openapi/swag v0.23.1 // indirect
	github.com/go-task/slim-sprig/v3 v3.0.0 // indirect
	github.com/go-viper/mapstructure/v2 v2.4.0 // indirect
	github.com/godbus/dbus/v5 v5.1.1-0.20241109141217-c266b19b28e9 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang-jwt/jwt/v5 v5.2.3 // indirect
	github.com/golang/groupcache v0.0.0-20241129210726-2c02b8208cf8 // indirect
	github.com/golang/protobuf v1.5.4 // indirect
	github.com/google/btree v1.1.3 // indirect
	github.com/google/go-containerregistry v0.20.6 // indirect
	github.com/google/go-intervals v0.0.2 // indirect
	github.com/google/gofuzz v1.2.0 // indirect
	github.com/google/pprof v0.0.0-20250630185457-6e76a2b096b5 // indirect
	github.com/google/shlex v0.0.0-20191202100458-e7afc7fbc510 // indirect
	github.com/googleapis/gnostic v0.6.8 // indirect
	github.com/gorilla/schema v1.4.1 // indirect
	github.com/gregjones/httpcache v0.0.0-20190611155906-901d90724c79 // indirect
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.27.1 // indirect
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/imdario/mergo v0.3.15 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/jedib0t/go-pretty/v6 v6.6.7 // indirect
	github.com/jinzhu/copier v0.4.0 // indirect
	github.com/josharian/intern v1.0.0 // indirect
	github.com/jpillora/backoff v1.0.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/kevinburke/ssh_config v1.2.0 // indirect
	github.com/klauspost/compress v1.18.0 // indirect
	github.com/klauspost/pgzip v1.2.6 // indirect
	github.com/kr/fs v0.1.0 // indirect
	github.com/kylelemons/godebug v1.1.0 // indirect
	github.com/letsencrypt/boulder v0.20250714.0 // indirect
	github.com/liggitt/tabwriter v0.0.0-20181228230101-89fcab3d43de // indirect
	github.com/mailru/easyjson v0.9.0 // indirect
	github.com/manifoldco/promptui v0.9.0 // indirect
	github.com/mattn/go-runewidth v0.0.16 // indirect
	github.com/mattn/go-sqlite3 v1.14.28 // indirect
	github.com/microsoft/kiota-authentication-azure-go v1.3.1 // indirect
	github.com/miekg/pkcs11 v1.1.1 // indirect
	github.com/mistifyio/go-zfs/v3 v3.0.1 // indirect
	github.com/mitchellh/go-wordwrap v1.0.1 // indirect
	github.com/moby/docker-image-spec v1.3.1 // indirect
	github.com/moby/spdystream v0.5.0 // indirect
	github.com/moby/sys/capability v0.4.0 // indirect
	github.com/moby/sys/mountinfo v0.7.2 // indirect
	github.com/moby/sys/user v0.4.0 // indirect
	github.com/moby/sys/userns v0.1.0 // indirect
	github.com/moby/term v0.5.2 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/monochromegane/go-gitignore v0.0.0-20200626010858-205db1a8cc00 // indirect
	github.com/morikuni/aec v1.0.0 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/mwitkow/go-conntrack v0.0.0-20190716064945-2f068394615f // indirect
	github.com/nxadm/tail v1.4.11 // indirect
	github.com/opencontainers/cgroups v0.0.4 // indirect
	github.com/opencontainers/go-digest v1.0.0 // indirect
	github.com/opencontainers/image-spec v1.1.1 // indirect
	github.com/opencontainers/runc v1.3.0 // indirect
	github.com/opencontainers/runtime-tools v0.9.1-0.20250523060157-0ea5ed0382a2 // indirect
	github.com/opencontainers/selinux v1.12.0 // indirect
	github.com/openshift/custom-resource-status v1.1.3-0.20220503160415-f2fdb4999d87 // indirect
	github.com/pelletier/go-toml/v2 v2.2.4 // indirect
	github.com/peterbourgon/diskv v2.0.1+incompatible // indirect
	github.com/pkg/browser v0.0.0-20240102092130-5ac0b6a4141c // indirect
	github.com/pkg/sftp v1.13.9 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/proglottis/gpgme v0.1.4 // indirect
	github.com/prometheus/client_model v0.6.2 // indirect
	github.com/prometheus/procfs v0.17.0 // indirect
	github.com/rivo/uniseg v0.4.7 // indirect
	github.com/robfig/cron v1.2.0 // indirect
	github.com/russross/blackfriday v1.6.0 // indirect
	github.com/sagikazarmark/locafero v0.9.0 // indirect
	github.com/sanity-io/litter v1.5.8 // indirect
	github.com/secure-systems-lab/go-securesystemslib v0.9.0 // indirect
	github.com/sigstore/fulcio v1.7.1 // indirect
	github.com/sigstore/protobuf-specs v0.5.0 // indirect
	github.com/sigstore/sigstore v1.9.5 // indirect
	github.com/skeema/knownhosts v1.3.1 // indirect
	github.com/smallstep/pkcs7 v0.2.1 // indirect
	github.com/sourcegraph/conc v0.3.0 // indirect
	github.com/spf13/afero v1.14.0 // indirect
	github.com/spf13/cast v1.9.2 // indirect
	github.com/spf13/pflag v1.0.7 // indirect
	github.com/std-uritemplate/std-uritemplate/go/v2 v2.0.5 // indirect
	github.com/stefanberger/go-pkcs11uri v0.0.0-20230803200340-78284954bff6 // indirect
	github.com/subosito/gotenv v1.6.0 // indirect
	github.com/sylabs/sif/v2 v2.21.1 // indirect
	github.com/tchap/go-patricia/v2 v2.3.3 // indirect
	github.com/titanous/rocacheck v0.0.0-20171023193734-afe73141d399 // indirect
	github.com/ulikunitz/xz v0.5.15 // indirect
	github.com/vbatts/tar-split v0.12.1 // indirect
	github.com/vbauerster/mpb/v8 v8.10.2 // indirect
	github.com/vmihailenco/msgpack/v4 v4.3.13 // indirect
	github.com/vmihailenco/tagparser v0.1.2 // indirect
	github.com/xlab/treeprint v1.1.0 // indirect
	go.opencensus.io v0.24.0 // indirect
	go.opentelemetry.io/auto/sdk v1.1.0 // indirect
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.62.0 // indirect
	go.opentelemetry.io/otel v1.37.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace v1.37.0 // indirect
	go.opentelemetry.io/otel/metric v1.37.0 // indirect
	go.opentelemetry.io/otel/trace v1.37.0 // indirect
	go.opentelemetry.io/proto/otlp v1.7.0 // indirect
	go.starlark.net v0.0.0-20220328144851-d1966c6b9fcd // indirect
	go.uber.org/automaxprocs v1.6.0 // indirect
	go.uber.org/goleak v1.3.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	go.yaml.in/yaml/v2 v2.4.2 // indirect
	golang.org/x/mod v0.26.0 // indirect
	golang.org/x/sys v0.34.0 // indirect
	golang.org/x/term v0.33.0 // indirect
	golang.org/x/time v0.12.0 // indirect
	gomodules.xyz/jsonpatch/v2 v2.5.0 // indirect
	google.golang.org/appengine v1.6.8 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20250715232539-7130f93afb79 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20250715232539-7130f93afb79 // indirect
	google.golang.org/grpc v1.72.2 // indirect
	google.golang.org/protobuf v1.36.6 // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
	gopkg.in/tomb.v1 v1.0.0-20141024135613-dd632973f1e7 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	k8s.io/apiserver v0.33.2 // indirect
	k8s.io/component-base v0.33.2 // indirect
	k8s.io/klog/v2 v2.130.1 // indirect
	k8s.io/kube-aggregator v0.33.2 // indirect
	k8s.io/kube-openapi v0.0.0-20250318190949-c8a335a9a2ff // indirect
	sigs.k8s.io/json v0.0.0-20241014173422-cfa47c3a1cc8 // indirect
	sigs.k8s.io/kube-storage-version-migrator v0.0.4 // indirect
	sigs.k8s.io/kustomize/api v0.12.1 // indirect
	sigs.k8s.io/kustomize/kyaml v0.13.9 // indirect
	sigs.k8s.io/structured-merge-diff/v4 v4.7.0 // indirect
	tags.cncf.io/container-device-interface v1.0.1 // indirect
)

// Exclude google cloud APIs from the go sum tree
exclude (
	cloud.google.com/go v0.26.0
	cloud.google.com/go v0.41.0
)

// remove old github.com/google, github.com/googleapis/ google.golang.org dependencies
exclude (
	github.com/google/gnostic v0.5.5
	github.com/google/go-cmp v0.2.0
	github.com/google/go-cmp v0.3.0
	github.com/google/go-cmp v0.3.1
	github.com/google/go-cmp v0.4.0
	github.com/google/go-cmp v0.5.0
	github.com/google/go-cmp v0.5.1
	github.com/google/go-cmp v0.5.3
	github.com/google/go-cmp v0.5.5
	github.com/google/go-cmp v0.5.8
	github.com/google/go-cmp v0.5.9
	github.com/google/go-cmp v0.6.0
	github.com/google/gofuzz v1.0.0
	github.com/google/gofuzz v1.1.0
	github.com/google/pprof v0.0.0-20210407192527-94a9f03dee38
	github.com/google/uuid v1.1.2
	github.com/google/uuid v1.2.0
	github.com/google/uuid v1.3.0
	google.golang.org/api v0.7.0
	google.golang.org/appengine v1.1.0
	google.golang.org/appengine v1.5.0
	google.golang.org/appengine v1.6.5
	google.golang.org/appengine v1.6.7
	google.golang.org/genproto v0.0.0-20180817151627-c66870c02cf8
	google.golang.org/genproto v0.0.0-20190502173448-54afdca5d873
	google.golang.org/genproto v0.0.0-20190819201941-24fa4b261c55
	google.golang.org/genproto v0.0.0-20200423170343-7949de9c1215
	google.golang.org/genproto v0.0.0-20200513103714-09dca8ec2884
	google.golang.org/genproto v0.0.0-20200526211855-cb27e3aa2013
	google.golang.org/genproto v0.0.0-20201019141844-1ed22bb0c154
	google.golang.org/genproto v0.0.0-20210602131652-f16073e35f0c
	google.golang.org/genproto v0.0.0-20220502173005-c8bf987b8c21
	google.golang.org/grpc v1.19.0
	google.golang.org/grpc v1.20.1
	google.golang.org/grpc v1.23.0
	google.golang.org/grpc v1.25.1
	google.golang.org/grpc v1.27.0
	google.golang.org/grpc v1.29.1
	google.golang.org/grpc v1.33.1
	google.golang.org/grpc v1.33.2
	google.golang.org/grpc v1.36.0
	google.golang.org/grpc v1.37.0
	google.golang.org/grpc v1.38.0
	google.golang.org/grpc v1.47.0
	google.golang.org/grpc v1.73.0
	google.golang.org/protobuf v1.24.0
	google.golang.org/protobuf v1.25.0
	google.golang.org/protobuf v1.26.0-rc.1
	google.golang.org/protobuf v1.26.0
	google.golang.org/protobuf v1.27.1
	google.golang.org/protobuf v1.28.0
)

exclude (
	github.com/Azure/go-ansiterm v0.0.0-20170929234023-d6e3b3328b78
	github.com/Azure/go-ansiterm v0.0.0-20210608223527-2377c96fe795
	github.com/Azure/go-autorest/autorest/adal v0.9.22
	github.com/Azure/go-autorest/autorest/mocks v0.4.1
	github.com/BurntSushi/toml v1.2.0
	github.com/Microsoft/go-winio v0.4.14
	github.com/asaskevich/govalidator v0.0.0-20190424111038-f61b66f89f4a
	github.com/asaskevich/govalidator v0.0.0-20200907205600-7a23bdc65eef
	github.com/chai2010/gettext-go v0.0.0-20160711120539-c6fed771bfd5
	github.com/chzyer/readline v0.0.0-20180603132655-2972be24d48e
	github.com/cilium/ebpf v0.4.0
	// Remove unneeded go-restful v2
	github.com/emicklei/go-restful v2.15.0+incompatible
	github.com/go-logr/logr v1.2.2
	github.com/golang/protobuf v1.3.2
	github.com/golang/protobuf v1.3.4
	github.com/golang/protobuf v1.4.0-rc.1
	github.com/golang/protobuf v1.4.0-rc.1.0.20200221234624-67d41d38c208
	github.com/golang/protobuf v1.4.0-rc.2
	github.com/golang/protobuf v1.4.0-rc.4.0.20200313231945-b860323f09d0
	github.com/golang/protobuf v1.4.0
	github.com/golang/protobuf v1.4.1
	github.com/moby/spdystream v0.2.0
	github.com/moby/term v0.0.0-20210619224110-3f7ff695adc6
	github.com/modern-go/reflect2 v0.0.0-20180701023420-4b7aa43c6742
	github.com/modern-go/reflect2 v1.0.1
	github.com/munnerz/goautoneg v0.0.0-20120707110453-a547fc61f48d
	// exclude old openshift library-go
	github.com/pkg/errors v0.8.1
	github.com/pkg/sftp v1.10.1
	github.com/pkg/sftp v1.13.1
	// remove old prometheus deps
	github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring v0.44.1
	github.com/prometheus/client_golang v0.9.1
	github.com/prometheus/client_golang v1.0.0
	github.com/prometheus/client_model v0.2.0
	// trip dependency tree from old cobra
	github.com/spf13/cobra v0.0.5
	github.com/spf13/cobra v1.2.1
	github.com/spf13/pflag v1.0.5
	github.com/spf13/pflag v1.0.6
	github.com/stretchr/objx v0.1.0
	github.com/stretchr/objx v0.2.0
	github.com/stretchr/objx v0.4.0
	github.com/stretchr/objx v0.5.0
	github.com/stretchr/testify v0.0.0-20161117074351-18a02ba4a312
	github.com/stretchr/testify v1.3.0
	github.com/stretchr/testify v1.4.0
	github.com/stretchr/testify v1.5.1
	github.com/stretchr/testify v1.6.1
	github.com/stretchr/testify v1.7.0
	github.com/stretchr/testify v1.7.1
	github.com/stretchr/testify v1.8.0
	github.com/stretchr/testify v1.8.1
	github.com/stretchr/testify v1.8.2
	gopkg.in/check.v1 v0.0.0-20161208181325-20d25e280405
	gopkg.in/check.v1 v1.0.0-20180628173108-788fd7840127
	gopkg.in/check.v1 v1.0.0-20190902080502-41f04d3bba15
	gopkg.in/check.v1 v1.0.0-20200227125254-8fa46927fb4f
	gopkg.in/yaml.v2 v2.2.2
	gopkg.in/yaml.v2 v2.2.3
	gopkg.in/yaml.v2 v2.2.4
	gopkg.in/yaml.v2 v2.2.5
	gopkg.in/yaml.v2 v2.2.8
	gopkg.in/yaml.v2 v2.3.0
	gopkg.in/yaml.v3 v3.0.0-20200615113413-eeeca48fe776
	gopkg.in/yaml.v3 v3.0.0-20210107192922-496545a6307b
)

// trim old golang.org/x/ and github.com/golang/ items
exclude (
	github.com/golang/groupcache v0.0.0-20200121045136-8c9f03a8e57e
	// exclude github.com/golang/protobuf < 1.3.2 https://nvd.nist.gov/vuln/detail/CVE-2021-3121
	github.com/golang/protobuf v1.0.0
	github.com/golang/protobuf v1.1.1
	github.com/golang/protobuf v1.2.0
	github.com/golang/protobuf v1.2.1
	github.com/golang/protobuf v1.3.0
	github.com/golang/protobuf v1.3.1
	github.com/golang/protobuf v1.4.2
	github.com/golang/protobuf v1.4.3
	github.com/golang/protobuf v1.5.0
	github.com/golang/protobuf v1.5.2
	go.uber.org/mock v1.4.4
	golang.org/x/arch v0.0.0-20180920145803-b19384d3c130
	golang.org/x/crypto v0.0.0-20190308221718-c2843e01d9a2
	golang.org/x/crypto v0.0.0-20190820162420-60c769a6c586
	golang.org/x/crypto v0.0.0-20191011191535-87dc89f01550
	golang.org/x/crypto v0.0.0-20201216223049-8b5274cf687f
	golang.org/x/crypto v0.0.0-20210220033148-5ea612d1eb83
	golang.org/x/crypto v0.0.0-20210322153248-0c34fe9e7dc2
	golang.org/x/crypto v0.0.0-20210421170649-83a5a9bb288b
	golang.org/x/crypto v0.0.0-20210921155107-089bfa567519
	golang.org/x/crypto v0.0.0-20211108221036-ceb1ce70b4fa
	golang.org/x/crypto v0.0.0-20211215153901-e495a2d5b3d3
	golang.org/x/crypto v0.0.0-20220131195533-30dcbda58838
	golang.org/x/crypto v0.6.0
	golang.org/x/crypto v0.13.0
	golang.org/x/crypto v0.14.0
	golang.org/x/crypto v0.17.0
	golang.org/x/crypto v0.19.0
	golang.org/x/crypto v0.23.0
	golang.org/x/crypto v0.31.0
	golang.org/x/lint v0.0.0-20190409202823-959b441ac422
	golang.org/x/lint v0.0.0-20190930215403-16217165b5de
	golang.org/x/lint v0.0.0-20210508222113-6edffad5e616
	golang.org/x/mod v0.4.2
	golang.org/x/mod v0.6.0-dev.0.20220106191415-9b9b3d81d5e3
	golang.org/x/mod v0.6.0-dev.0.20220419223038-86c51ed26bb4
	golang.org/x/mod v0.7.0
	golang.org/x/mod v0.8.0
	golang.org/x/mod v0.9.0
	golang.org/x/net v0.0.0-20180906233101-161cd47e91fd
	golang.org/x/net v0.0.0-20190213061140-3a22650c66bd
	golang.org/x/net v0.0.0-20190311183353-d8887717615a
	golang.org/x/net v0.0.0-20190404232315-eb5bcb51f2a3
	golang.org/x/net v0.0.0-20190603091049-60506f45cf65
	golang.org/x/net v0.0.0-20190620200207-3b0461eec859
	golang.org/x/net v0.0.0-20190628185345-da137c7871d7
	golang.org/x/net v0.0.0-20190827160401-ba9fcec4b297
	golang.org/x/net v0.0.0-20200202094626-16171245cfb2
	golang.org/x/net v0.0.0-20200226121028-0de0cce0169b
	golang.org/x/net v0.0.0-20200301022130-244492dfa37a
	golang.org/x/net v0.0.0-20200520004742-59133d7f0dd7
	golang.org/x/net v0.0.0-20200602114024-627f9648deb9
	golang.org/x/net v0.0.0-20200822124328-c89045814202
	golang.org/x/net v0.0.0-20201021035429-f5854403a974
	golang.org/x/net v0.0.0-20201110031124-69a78807bb2b
	golang.org/x/net v0.0.0-20201202161906-c7110b5ffcbb
	golang.org/x/net v0.0.0-20210226172049-e18ecbb05110
	golang.org/x/net v0.0.0-20210405180319-a5a99cb37ef4
	golang.org/x/net v0.0.0-20210421230115-4e50805a0758
	golang.org/x/net v0.0.0-20210428140749-89ef3d95e781
	golang.org/x/net v0.0.0-20210825183410-e898025ed96a
	golang.org/x/net v0.0.0-20211015210444-4f30a5c0130f
	golang.org/x/net v0.0.0-20211112202133-69e39bad7dc2
	golang.org/x/net v0.0.0-20211209124913-491a49abca63
	golang.org/x/net v0.0.0-20220127200216-cd36cc0744dd
	golang.org/x/net v0.0.0-20220225172249-27dd8689420f
	golang.org/x/net v0.0.0-20220425223048-2871e0cb64e4
	golang.org/x/net v0.0.0-20220624214902-1bab6f366d9e
	golang.org/x/net v0.0.0-20220722155237-a158d28d115b
	golang.org/x/net v0.0.0-20221014081412-f15817d10f9b
	golang.org/x/net v0.2.0
	golang.org/x/net v0.3.0
	golang.org/x/net v0.4.0
	golang.org/x/net v0.6.0
	golang.org/x/net v0.7.0
	golang.org/x/net v0.8.0
	golang.org/x/net v0.9.0
	golang.org/x/net v0.10.0
	golang.org/x/net v0.17.0
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
	golang.org/x/oauth2 v0.0.0-20220223155221-ee480838109b
	golang.org/x/oauth2 v0.0.0-20220309155454-6242fa91716a
	golang.org/x/oauth2 v0.0.0-20220411215720-9780585627b5
	golang.org/x/oauth2 v0.0.0-20220608161450-d0670ef3b1eb
	golang.org/x/oauth2 v0.0.0-20220622183110-fd043fe589d2
	golang.org/x/oauth2 v0.0.0-20220822191816-0ebed06d0094
	golang.org/x/oauth2 v0.0.0-20220909003341-f21342109be1
	golang.org/x/oauth2 v0.0.0-20221006150949-b44042a4b9c1
	golang.org/x/oauth2 v0.0.0-20221014153046-6fdb5e3db783
	golang.org/x/oauth2 v0.5.0
	golang.org/x/oauth2 v0.6.0
	golang.org/x/oauth2 v0.7.0
	golang.org/x/sync v0.0.0-20180314180146-1d60e4601c6f
	golang.org/x/sync v0.0.0-20181108010431-42b317875d0f
	golang.org/x/sync v0.0.0-20181221193216-37e7f081c4d4
	golang.org/x/sync v0.0.0-20190227155943-e225da77a7e6
	golang.org/x/sync v0.0.0-20190423024810-112230192c58
	golang.org/x/sync v0.0.0-20190911185100-cd5d95a43a6e
	golang.org/x/sync v0.0.0-20200625203802-6e8e738ad208
	golang.org/x/sync v0.0.0-20201020160332-67f06af15bc9
	golang.org/x/sync v0.0.0-20201207232520-09787c993a3a
	golang.org/x/sync v0.0.0-20210220032951-036812b2e83c
	golang.org/x/sync v0.0.0-20220722155255-886fb9371eb4
	golang.org/x/sync v0.1.0
	golang.org/x/sync v0.3.0
	golang.org/x/sync v0.6.0
	golang.org/x/sync v0.7.0
	golang.org/x/sys v0.0.0-20180903190138-2b024373dcd9
	golang.org/x/sys v0.0.0-20180905080454-ebe1bf3edb33
	golang.org/x/sys v0.0.0-20180909124046-d0be0721c37e
	golang.org/x/sys v0.0.0-20181122145206-62eef0e2fa9b
	golang.org/x/sys v0.0.0-20190215142949-d0b11bdaac8a
	golang.org/x/sys v0.0.0-20190222072716-a9d3bda3a223
	golang.org/x/sys v0.0.0-20190412213103-97732733099d
	golang.org/x/sys v0.0.0-20190422165155-953cdadca894
	golang.org/x/sys v0.0.0-20190507160741-ecd444e8653b
	golang.org/x/sys v0.0.0-20190626221950-04f50cda93cb
	golang.org/x/sys v0.0.0-20190904154756-749cb33beabd
	golang.org/x/sys v0.0.0-20191002063906-3421d5a6bb1c
	golang.org/x/sys v0.0.0-20191005200804-aed5e4c7ecf9
	golang.org/x/sys v0.0.0-20191026070338-33540a1f6037
	golang.org/x/sys v0.0.0-20191115151921-52ab43148777
	golang.org/x/sys v0.0.0-20191120155948-bd437916bb0e
	golang.org/x/sys v0.0.0-20191204072324-ce4227a45e2e
	golang.org/x/sys v0.0.0-20200116001909-b77594299b42
	golang.org/x/sys v0.0.0-20200217220822-9197077df867
	golang.org/x/sys v0.0.0-20200223170610-d5e6a3e2c0ae
	golang.org/x/sys v0.0.0-20200323222414-85ca7c5b95cd
	golang.org/x/sys v0.0.0-20200519105757-fe76b779f299
	golang.org/x/sys v0.0.0-20200610111108-226ff32320da
	golang.org/x/sys v0.0.0-20200728102440-3e129f6d46b1
	golang.org/x/sys v0.0.0-20200831180312-196b9ba8737a
	golang.org/x/sys v0.0.0-20200916030750-2334cc1a136f
	golang.org/x/sys v0.0.0-20200923182605-d9f96fdee20d
	golang.org/x/sys v0.0.0-20200930185726-fdedc70b468f
	golang.org/x/sys v0.0.0-20201119102817-f84b799fce68
	golang.org/x/sys v0.0.0-20210112080510-489259a85091
	golang.org/x/sys v0.0.0-20210119212857-b64e53b001e4
	golang.org/x/sys v0.0.0-20210124154548-22da62e12c0c
	golang.org/x/sys v0.0.0-20210330210617-4fbd30eecc44
	golang.org/x/sys v0.0.0-20210403161142-5e06dd20ab57
	golang.org/x/sys v0.0.0-20210420072515-93ed5bcd2bfe
	golang.org/x/sys v0.0.0-20210423082822-04245dca01da
	golang.org/x/sys v0.0.0-20210423185535-09eb48e85fd7
	golang.org/x/sys v0.0.0-20210510120138-977fb7262007
	golang.org/x/sys v0.0.0-20210615035016-665e8c7367d1
	golang.org/x/sys v0.0.0-20210616045830-e2b7044e8c71
	golang.org/x/sys v0.0.0-20210616094352-59db8d763f22
	golang.org/x/sys v0.0.0-20210630005230-0f9fa26af87c
	golang.org/x/sys v0.0.0-20210906170528-6f6e22806c34
	golang.org/x/sys v0.0.0-20211019181941-9d821ace8654
	golang.org/x/sys v0.0.0-20211029165221-6e7872819dc8
	golang.org/x/sys v0.0.0-20211216021012-1d35b9e2eb4e
	golang.org/x/sys v0.0.0-20220209214540-3681064d5158
	golang.org/x/sys v0.0.0-20220310020820-b874c991c1a5
	golang.org/x/sys v0.0.0-20220319134239-a9b59b0215f8
	golang.org/x/sys v0.0.0-20220422013727-9388b58f7150
	golang.org/x/sys v0.0.0-20220520151302-bc2c85ada10a
	golang.org/x/sys v0.0.0-20220610221304-9f5ed59c137d
	golang.org/x/sys v0.0.0-20220715151400-c0bba94af5f8
	golang.org/x/sys v0.0.0-20220722155257-8c9f86f7a55f
	golang.org/x/sys v0.0.0-20220728004956-3c1f35247d10
	golang.org/x/sys v0.0.0-20220811171246-fbc7d0a398ab
	golang.org/x/sys v0.0.0-20220817070843-5a390386f1f2
	golang.org/x/sys v0.0.0-20220823224334-20c2bfdbfe24
	golang.org/x/sys v0.0.0-20220908164124-27713097b956
	golang.org/x/sys v0.0.0-20220909162455-aba9fc2a8ff2
	golang.org/x/sys v0.1.0
	golang.org/x/sys v0.2.0
	golang.org/x/sys v0.3.0
	golang.org/x/sys v0.5.0
	golang.org/x/sys v0.6.0
	golang.org/x/sys v0.7.0
	golang.org/x/sys v0.8.0
	golang.org/x/sys v0.13.0
	golang.org/x/term v0.0.0-20201117132131-f5c789dd3221
	golang.org/x/term v0.0.0-20201126162022-7de9c90e9dd1
	golang.org/x/term v0.0.0-20210615171337-6886f2dfbf5b
	golang.org/x/term v0.0.0-20210927222741-03fcf44c2211
	golang.org/x/term v0.2.0
	golang.org/x/term v0.3.0
	golang.org/x/term v0.5.0
	golang.org/x/term v0.6.0
	golang.org/x/term v0.7.0
	golang.org/x/term v0.8.0
	golang.org/x/term v0.13.0
	golang.org/x/text v0.3.0
	golang.org/x/text v0.3.2
	golang.org/x/text v0.3.3
	golang.org/x/text v0.3.4
	golang.org/x/text v0.3.5
	golang.org/x/text v0.3.6
	golang.org/x/text v0.3.7
	golang.org/x/text v0.3.8
	golang.org/x/text v0.4.0
	golang.org/x/text v0.5.0
	golang.org/x/text v0.7.0
	golang.org/x/text v0.8.0
	golang.org/x/text v0.9.0
	golang.org/x/text v0.13.0
	golang.org/x/time v0.0.0-20200416051211-89c76fbcd5d1
	golang.org/x/time v0.0.0-20210220033141-f8bda1e9f3ba
	golang.org/x/time v0.0.0-20210723032227-1f47c861a9ac
	golang.org/x/time v0.0.0-20220210224613-90d013bbcef8
	golang.org/x/tools v0.0.0-20180917221912-90fa682c2a6e
	golang.org/x/tools v0.0.0-20190226205152-f727befe758c
	golang.org/x/tools v0.0.0-20190311212946-11955173bddd
	golang.org/x/tools v0.0.0-20190425150028-36563e24a262
	golang.org/x/tools v0.0.0-20190524140312-2c0ae7006135
	golang.org/x/tools v0.0.0-20190524210228-3d17549cdc6b
	golang.org/x/tools v0.0.0-20190531172133-b3315ee88b7d
	golang.org/x/tools v0.0.0-20190624222133-a101b041ded4
	golang.org/x/tools v0.0.0-20190628153133-6cdbf07be9d0
	golang.org/x/tools v0.0.0-20190706070813-72ffa07ba3db
	golang.org/x/tools v0.0.0-20191108193012-7d206e10da11
	golang.org/x/tools v0.0.0-20191119224855-298f0cb1881e
	golang.org/x/tools v0.0.0-20200130002326-2f3ba24bd6e7
	golang.org/x/tools v0.0.0-20200505023115-26f46d2f7ef8
	golang.org/x/tools v0.0.0-20200509030707-2212a7e161a5
	golang.org/x/tools v0.0.0-20200610160956-3e83d1e96d0e
	golang.org/x/tools v0.0.0-20200616133436-c1934b75d054
	golang.org/x/tools v0.0.0-20200619180055-7c47624df98f
	golang.org/x/tools v0.0.0-20200916195026-c9a70fc28ce3
	golang.org/x/tools v0.0.0-20201224043029-2b0845dc783e
	golang.org/x/tools v0.0.0-20210106214847-113979e3529a
	golang.org/x/tools v0.1.1
	golang.org/x/tools v0.1.2
	golang.org/x/tools v0.1.5
	golang.org/x/tools v0.1.9
	golang.org/x/tools v0.1.10-0.20220218145154-897bd77cd717
	golang.org/x/tools v0.1.10
	golang.org/x/tools v0.1.12
	golang.org/x/tools v0.3.0
	golang.org/x/tools v0.4.0
	golang.org/x/tools v0.6.0
	golang.org/x/tools v0.7.0
	golang.org/x/xerrors v0.0.0-20190717185122-a985d3407aa7
	golang.org/x/xerrors v0.0.0-20191204190536-9bdfabe68543
	golang.org/x/xerrors v0.0.0-20200804184101-5ec99f83aff1
	golang.org/x/xerrors v0.0.0-20220907171357-04be3eba64a2
)

// exclude ancient k8s versions
exclude (
	k8s.io/api v0.18.0-beta.2
	k8s.io/api v0.18.3
	k8s.io/api v0.23.3
	k8s.io/apiextensions-apiserver v0.0.0
	k8s.io/apiextensions-apiserver v0.18.0-beta.2
	k8s.io/apiextensions-apiserver v0.18.3
	k8s.io/apiextensions-apiserver v0.23.1
	k8s.io/apiextensions-apiserver v0.24.0
	k8s.io/apimachinery v0.18.0-beta.2
	k8s.io/apimachinery v0.18.3
	k8s.io/apimachinery v0.19.2
	k8s.io/apimachinery v0.23.3
	k8s.io/client-go v0.18.0-beta.2
	k8s.io/client-go v0.19.2
	k8s.io/code-generator v0.18.0-beta.2
	k8s.io/code-generator v0.23.3
	k8s.io/code-generator v0.25.16
	k8s.io/gengo v0.0.0-20210813121822-485abfe95c7c
	k8s.io/gengo v0.0.0-20211129171323-c02415ce4185
	k8s.io/klog v1.0.0
	k8s.io/klog/v2 v2.0.0
	k8s.io/klog/v2 v2.2.0
	k8s.io/klog/v2 v2.40.1
	k8s.io/klog/v2 v2.70.1
	k8s.io/kube-aggregator v0.18.0-beta.2
	k8s.io/kube-openapi v0.0.0-20210421082810-95288971da7e
	k8s.io/kube-openapi v0.0.0-20220124234850-424119656bbf
	k8s.io/kube-openapi v0.0.0-20220803162953-67bda5d908f1
	k8s.io/kubernetes v0.0.0-00010101000000-000000000000
	k8s.io/utils v0.0.0-20210802155522-efc7438f0176
	k8s.io/utils v0.0.0-20220728103510-ee6ede2d64ed
	sigs.k8s.io/apiserver-network-proxy/konnectivity-client v0.0.37
	sigs.k8s.io/controller-tools v0.2.8
	sigs.k8s.io/json v0.0.0-20220713155537-f223a00ba0e2
	sigs.k8s.io/json v0.0.0-20221116044647-bc3834ca7abd
	sigs.k8s.io/structured-merge-diff/v4 v4.2.3
	sigs.k8s.io/yaml v1.1.0
	sigs.k8s.io/yaml v1.2.0
	sigs.k8s.io/yaml v1.3.0
	sigs.k8s.io/yaml v1.4.0
)

// k8s.io and sigs.k8s.io pins
replace (
	k8s.io/api => k8s.io/api v0.25.16
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.25.16
	k8s.io/apimachinery => k8s.io/apimachinery v0.25.16
	k8s.io/apiserver => k8s.io/apiserver v0.25.16
	k8s.io/client-go => k8s.io/client-go v0.25.16
	k8s.io/cloud-provider => k8s.io/cloud-provider v0.25.16 // required for k8s.io/kubernetes
	k8s.io/cluster-bootstrap => k8s.io/cluster-bootstrap v0.25.16 // required for k8s.io/kubernetes
	k8s.io/code-generator => k8s.io/code-generator v0.25.16
	k8s.io/component-base => k8s.io/component-base v0.25.16
	k8s.io/csi-translation-lib => k8s.io/csi-translation-lib v0.25.16 // required for k8s.io/kubernetes
	k8s.io/kube-aggregator => k8s.io/kube-aggregator v0.25.16
	k8s.io/kube-controller-manager => k8s.io/kube-controller-manager v0.25.16
	k8s.io/kube-openapi => k8s.io/kube-openapi v0.0.0-20230501164219-8b0f38b5fd1f
	k8s.io/kube-proxy => k8s.io/kube-proxy v0.25.16
	k8s.io/kube-scheduler => k8s.io/kube-scheduler v0.25.16
	k8s.io/kubectl => k8s.io/kubectl v0.25.16
	k8s.io/kubernetes => k8s.io/kubernetes v1.25.16
	sigs.k8s.io/controller-runtime => sigs.k8s.io/controller-runtime v0.13.2
	sigs.k8s.io/kustomize/api => sigs.k8s.io/kustomize/api v0.11.2
	sigs.k8s.io/kustomize/kyaml => sigs.k8s.io/kustomize/kyaml v0.13.3
	sigs.k8s.io/structured-merge-diff => sigs.k8s.io/structured-merge-diff v1.0.1-0.20191108220359-b1b620dd3f06
)

// OpenShift pins
replace (
	github.com/googleapis/gnostic => github.com/google/gnostic v0.5.5
	github.com/openshift/api => github.com/openshift/api v0.0.0-20240103200955-7ca3a4634e46
	github.com/openshift/client-go => github.com/openshift/client-go v0.0.0-20221019143426-16aed247da5c
	github.com/openshift/hive/apis => github.com/openshift/hive/apis v0.0.0-20231116161336-9dd47f8bfa1f
	github.com/openshift/library-go => github.com/openshift/library-go v0.0.0-20230222114049-eac44a078a6e
	github.com/openshift/machine-config-operator => github.com/openshift/machine-config-operator v0.0.1-0.20230908201248-46b93e64dea6
)

// broken deps on 2.8.3
replace github.com/docker/distribution v2.8.3+incompatible => github.com/docker/distribution v2.8.2+incompatible

// ARO-RP sub-packages
replace github.com/Azure/ARO-RP/pkg/api => ./pkg/api
