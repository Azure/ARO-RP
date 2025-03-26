module github.com/Azure/ARO-RP

go 1.22.9

require (
	github.com/Azure/azure-sdk-for-go v63.1.0+incompatible
	github.com/Azure/azure-sdk-for-go/sdk/azcore v1.17.0
	github.com/Azure/azure-sdk-for-go/sdk/azidentity v1.8.2
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/authorization/armauthorization/v3 v3.0.0-beta.2
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/containerservice/armcontainerservice/v6 v6.2.0
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/cosmos/armcosmos/v2 v2.5.0
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/dns/armdns v1.2.0
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/keyvault/armkeyvault v1.4.0
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/msi/armmsi v1.2.0
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v2 v2.2.1
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/storage/armstorage v1.5.0
	github.com/Azure/azure-sdk-for-go/sdk/security/keyvault/azcertificates v1.1.0
	github.com/Azure/azure-sdk-for-go/sdk/security/keyvault/azsecrets v1.3.1
	github.com/Azure/azure-sdk-for-go/sdk/storage/azblob v1.3.2
	github.com/Azure/checkaccess-v2-go-sdk v0.0.3
	github.com/Azure/go-autorest/autorest v0.11.29
	github.com/Azure/go-autorest/autorest/adal v0.9.23
	github.com/Azure/go-autorest/autorest/date v0.3.0
	github.com/Azure/go-autorest/autorest/to v0.4.0
	github.com/Azure/go-autorest/autorest/validation v0.3.1
	github.com/Azure/go-autorest/tracing v0.6.0
	github.com/Azure/msi-dataplane v0.4.2
	github.com/apparentlymart/go-cidr v1.1.0
	github.com/codahale/etm v0.0.0-20141003032925-c00c9e6fb4c9
	github.com/containers/image/v5 v5.33.1
	github.com/containers/podman/v5 v5.3.2
	github.com/coreos/go-oidc/v3 v3.11.0
	github.com/coreos/go-semver v0.3.0
	github.com/coreos/go-systemd/v22 v22.5.1-0.20231103132048-7d375ecc2b09
	github.com/coreos/ignition/v2 v2.14.0
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc
	github.com/go-chi/chi/v5 v5.0.8
	github.com/go-jose/go-jose/v4 v4.0.5
	github.com/go-logr/logr v1.4.2
	github.com/go-test/deep v1.1.1
	github.com/gofrs/uuid v4.4.0+incompatible
	github.com/golang-jwt/jwt/v4 v4.5.1
	github.com/google/gnostic v0.5.7-v3refs
	github.com/google/go-cmp v0.6.0
	github.com/google/uuid v1.6.0
	github.com/gorilla/csrf v1.7.2
	github.com/gorilla/mux v1.8.1
	github.com/gorilla/securecookie v1.1.2
	github.com/gorilla/sessions v1.2.2
	github.com/hashicorp/go-multierror v1.1.1
	github.com/jongio/azidext/go/azidext v0.5.0
	github.com/microsoft/go-otel-audit v0.2.1
	github.com/microsoft/kiota-abstractions-go v1.2.0
	github.com/microsoft/kiota-http-go v1.0.0
	github.com/microsoft/kiota-serialization-form-go v1.0.0
	github.com/microsoft/kiota-serialization-json-go v1.0.4
	github.com/microsoft/kiota-serialization-multipart-go v1.0.0
	github.com/microsoft/kiota-serialization-text-go v1.0.0
	github.com/microsoftgraph/msgraph-sdk-go-core v1.0.0
	github.com/onsi/ginkgo/v2 v2.21.0
	github.com/onsi/gomega v1.35.1
	github.com/open-policy-agent/frameworks/constraint v0.0.0-20221109005544-7de84dff5081
	github.com/opencontainers/runtime-spec v1.2.0
	github.com/openshift/api v0.0.0-20240103200955-7ca3a4634e46
	github.com/openshift/client-go v0.0.0-20221019143426-16aed247da5c
	github.com/openshift/cloud-credential-operator v0.0.0-20240910012137-a0245d57d1e6
	github.com/openshift/hive/apis v0.0.0-20250212001559-5d3f4d77dc90
	github.com/openshift/library-go v0.0.0-20230620084201-504ca4bd5a83
	github.com/openshift/machine-config-operator v0.0.0-00010101000000-000000000000
	github.com/pires/go-proxyproto v0.6.2
	github.com/pkg/errors v0.9.1
	github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring v0.50.0
	github.com/prometheus-operator/prometheus-operator/pkg/client v0.48.1
	github.com/prometheus/client_golang v1.20.2
	github.com/prometheus/common v0.57.0
	github.com/serge1peshcoff/selenium-go-conditions v0.0.0-20170824121757-5afbdb74596b
	github.com/sirupsen/logrus v1.9.3
	github.com/spf13/viper v1.20.0-alpha.6
	github.com/stretchr/testify v1.10.0
	github.com/tebeka/selenium v0.9.9
	github.com/ugorji/go/codec v1.2.12
	github.com/vincent-petithory/dataurl v1.0.0
	go.uber.org/mock v0.4.0
	golang.org/x/crypto v0.33.0
	golang.org/x/exp v0.0.0-20241009180824-f66d83c29e7c
	golang.org/x/net v0.35.0
	golang.org/x/oauth2 v0.23.0
	golang.org/x/sync v0.11.0
	golang.org/x/text v0.22.0
	golang.org/x/tools v0.26.0
	k8s.io/api v0.31.1
	k8s.io/apiextensions-apiserver v0.27.2
	k8s.io/apimachinery v0.31.1
	k8s.io/cli-runtime v0.25.16
	k8s.io/client-go v0.27.3
	k8s.io/kubectl v0.24.17
	k8s.io/kubernetes v1.28.4
	k8s.io/utils v0.0.0-20240921022957-49e7df575cb6
	sigs.k8s.io/controller-runtime v0.15.0
	sigs.k8s.io/yaml v1.4.0
)

require (
	dario.cat/mergo v1.0.1 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/internal v1.10.0 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/security/keyvault/internal v1.1.1 // indirect
	github.com/Azure/go-ansiterm v0.0.0-20230124172434-306776ec8161 // indirect
	github.com/Azure/go-autorest v14.2.0+incompatible // indirect
	github.com/Azure/go-autorest/logger v0.2.1 // indirect
	github.com/Azure/retry v0.0.0-20240325164105-70e16f388626 // indirect
	github.com/AzureAD/microsoft-authentication-library-for-go v1.3.3 // indirect
	github.com/BurntSushi/toml v1.4.0 // indirect
	github.com/BurntSushi/xgb v0.0.0-20210121224620-deaf085860bc // indirect
	github.com/MakeNowJust/heredoc v1.0.0 // indirect
	github.com/Microsoft/go-winio v0.6.2 // indirect
	github.com/Microsoft/hcsshim v0.12.9 // indirect
	github.com/VividCortex/ewma v1.2.0 // indirect
	github.com/acarl005/stripansi v0.0.0-20180116102854-5a71ef0e047d // indirect
	github.com/antlr/antlr4/runtime/Go/antlr v0.0.0-20220418222510-f25a4f6275ed // indirect
	github.com/antlr4-go/antlr/v4 v4.13.1 // indirect
	github.com/asaskevich/govalidator v0.0.0-20230301143203-a9d515a09cc2 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/blang/semver v3.5.1+incompatible // indirect
	github.com/blang/semver/v4 v4.0.0 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/chai2010/gettext-go v1.0.2 // indirect
	github.com/chzyer/readline v1.5.1 // indirect
	github.com/cjlapao/common-go v0.0.39 // indirect
	github.com/containerd/cgroups/v3 v3.0.3 // indirect
	github.com/containerd/errdefs v0.3.0 // indirect
	github.com/containerd/errdefs/pkg v0.3.0 // indirect
	github.com/containerd/log v0.1.0 // indirect
	github.com/containerd/platforms v0.2.1 // indirect
	github.com/containerd/stargz-snapshotter/estargz v0.15.1 // indirect
	github.com/containerd/typeurl/v2 v2.2.0 // indirect
	github.com/containers/buildah v1.38.1 // indirect
	github.com/containers/common v0.61.1 // indirect
	github.com/containers/libtrust v0.0.0-20230121012942-c1716e8a8d01 // indirect
	github.com/containers/ocicrypt v1.2.0 // indirect
	github.com/containers/psgo v1.9.0 // indirect
	github.com/containers/storage v1.56.1 // indirect
	github.com/coreos/vcontext v0.0.0-20231102161604-685dc7299dc5 // indirect
	github.com/cyberphone/json-canonicalization v0.0.0-20231217050601-ba74d44ecf5f // indirect
	github.com/cyphar/filepath-securejoin v0.3.4 // indirect
	github.com/disiqueira/gotree/v3 v3.0.2 // indirect
	github.com/distribution/reference v0.6.0 // indirect
	github.com/docker/distribution v2.8.3+incompatible // indirect
	github.com/docker/docker v27.3.1+incompatible // indirect
	github.com/docker/docker-credential-helpers v0.8.2 // indirect
	github.com/docker/go-connections v0.5.0 // indirect
	github.com/docker/go-units v0.5.0 // indirect
	github.com/elazarl/goproxy v1.7.2 // indirect
	github.com/emicklei/go-restful/v3 v3.10.1 // indirect
	github.com/evanphx/json-patch v5.6.0+incompatible // indirect
	github.com/exponent-io/jsonpath v0.0.0-20210407135951-1de76d718b3f // indirect
	github.com/felixge/httpsnoop v1.0.4 // indirect
	github.com/fsnotify/fsnotify v1.8.0 // indirect
	github.com/ghodss/yaml v1.0.1-0.20190212211648-25d852aebe32 // indirect
	github.com/go-errors/errors v1.4.2 // indirect
	github.com/go-json-experiment/json v0.0.0-20240418180308-af2d5061e6c2 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-openapi/analysis v0.23.0 // indirect
	github.com/go-openapi/errors v0.22.0 // indirect
	github.com/go-openapi/jsonpointer v0.21.0 // indirect
	github.com/go-openapi/jsonreference v0.21.0 // indirect
	github.com/go-openapi/loads v0.22.0 // indirect
	github.com/go-openapi/runtime v0.28.0 // indirect
	github.com/go-openapi/spec v0.21.0 // indirect
	github.com/go-openapi/strfmt v0.23.0 // indirect
	github.com/go-openapi/swag v0.23.0 // indirect
	github.com/go-openapi/validate v0.24.0 // indirect
	github.com/go-stack/stack v1.8.1 // indirect
	github.com/go-task/slim-sprig/v3 v3.0.0 // indirect
	github.com/go-viper/mapstructure/v2 v2.0.0 // indirect
	github.com/godbus/dbus/v5 v5.1.1-0.20240921181615-a817f3cc4a9e // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang-jwt/jwt/v5 v5.2.1 // indirect
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da // indirect
	github.com/golang/protobuf v1.5.4 // indirect
	github.com/google/btree v1.0.1 // indirect
	github.com/google/cel-go v0.12.6 // indirect
	github.com/google/go-containerregistry v0.20.2 // indirect
	github.com/google/go-intervals v0.0.2 // indirect
	github.com/google/gofuzz v1.2.0 // indirect
	github.com/google/pprof v0.0.0-20241029153458-d1b30febd7db // indirect
	github.com/google/shlex v0.0.0-20191202100458-e7afc7fbc510 // indirect
	github.com/googleapis/gnostic v0.6.8 // indirect
	github.com/gorilla/schema v1.4.1 // indirect
	github.com/gregjones/httpcache v0.0.0-20190611155906-901d90724c79 // indirect
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/imdario/mergo v0.3.15 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/jedib0t/go-pretty/v6 v6.5.6 // indirect
	github.com/jinzhu/copier v0.4.0 // indirect
	github.com/josharian/intern v1.0.0 // indirect
	github.com/jpillora/backoff v1.0.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/kevinburke/ssh_config v1.2.0 // indirect
	github.com/klauspost/compress v1.17.11 // indirect
	github.com/klauspost/pgzip v1.2.6 // indirect
	github.com/kr/fs v0.1.0 // indirect
	github.com/kylelemons/godebug v1.1.0 // indirect
	github.com/letsencrypt/boulder v0.0.0-20240620165639-de9c06129bec // indirect
	github.com/liggitt/tabwriter v0.0.0-20181228230101-89fcab3d43de // indirect
	github.com/mailru/easyjson v0.7.7 // indirect
	github.com/manifoldco/promptui v0.9.0 // indirect
	github.com/mattn/go-runewidth v0.0.16 // indirect
	github.com/mattn/go-sqlite3 v1.14.24 // indirect
	github.com/microsoft/kiota-authentication-azure-go v1.0.0 // indirect
	github.com/miekg/pkcs11 v1.1.1 // indirect
	github.com/mistifyio/go-zfs/v3 v3.0.1 // indirect
	github.com/mitchellh/go-wordwrap v1.0.1 // indirect
	github.com/mitchellh/mapstructure v1.5.0 // indirect
	github.com/moby/docker-image-spec v1.3.1 // indirect
	github.com/moby/spdystream v0.2.0 // indirect
	github.com/moby/sys/capability v0.3.0 // indirect
	github.com/moby/sys/mountinfo v0.7.2 // indirect
	github.com/moby/sys/user v0.3.0 // indirect
	github.com/moby/sys/userns v0.1.0 // indirect
	github.com/moby/term v0.5.0 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/monochromegane/go-gitignore v0.0.0-20200626010858-205db1a8cc00 // indirect
	github.com/morikuni/aec v1.0.0 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/mwitkow/go-conntrack v0.0.0-20190716064945-2f068394615f // indirect
	github.com/nxadm/tail v1.4.11 // indirect
	github.com/oklog/ulid v1.3.1 // indirect
	github.com/opencontainers/go-digest v1.0.0 // indirect
	github.com/opencontainers/image-spec v1.1.0 // indirect
	github.com/opencontainers/runc v1.2.1 // indirect
	github.com/opencontainers/runtime-tools v0.9.1-0.20241001195557-6c9570a1678f // indirect
	github.com/opencontainers/selinux v1.11.1 // indirect
	github.com/openshift/custom-resource-status v1.1.3-0.20220503160415-f2fdb4999d87 // indirect
	github.com/ostreedev/ostree-go v0.0.0-20210805093236-719684c64e4f // indirect
	github.com/pelletier/go-toml/v2 v2.2.2 // indirect
	github.com/peterbourgon/diskv v2.0.1+incompatible // indirect
	github.com/pkg/browser v0.0.0-20240102092130-5ac0b6a4141c // indirect
	github.com/pkg/sftp v1.13.7 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/proglottis/gpgme v0.1.3 // indirect
	github.com/prometheus/client_model v0.6.1 // indirect
	github.com/prometheus/procfs v0.15.1 // indirect
	github.com/rivo/uniseg v0.4.7 // indirect
	github.com/robfig/cron v1.2.0 // indirect
	github.com/russross/blackfriday v1.6.0 // indirect
	github.com/sagikazarmark/locafero v0.6.0 // indirect
	github.com/sanity-io/litter v1.5.5 // indirect
	github.com/secure-systems-lab/go-securesystemslib v0.8.0 // indirect
	github.com/sigstore/fulcio v1.6.4 // indirect
	github.com/sigstore/rekor v1.3.6 // indirect
	github.com/sigstore/sigstore v1.8.9 // indirect
	github.com/skeema/knownhosts v1.3.0 // indirect
	github.com/sourcegraph/conc v0.3.0 // indirect
	github.com/spf13/afero v1.11.0 // indirect
	github.com/spf13/cast v1.6.0 // indirect
	github.com/spf13/cobra v1.8.1 // indirect
	github.com/spf13/pflag v1.0.6-0.20210604193023-d5e0c0615ace // indirect
	github.com/stefanberger/go-pkcs11uri v0.0.0-20230803200340-78284954bff6 // indirect
	github.com/stoewer/go-strcase v1.2.0 // indirect
	github.com/subosito/gotenv v1.6.0 // indirect
	github.com/sylabs/sif/v2 v2.19.1 // indirect
	github.com/syndtr/gocapability v0.0.0-20200815063812-42c35b437635 // indirect
	github.com/tchap/go-patricia/v2 v2.3.1 // indirect
	github.com/titanous/rocacheck v0.0.0-20171023193734-afe73141d399 // indirect
	github.com/ulikunitz/xz v0.5.12 // indirect
	github.com/vbatts/tar-split v0.11.6 // indirect
	github.com/vbauerster/mpb/v8 v8.8.3 // indirect
	github.com/vmihailenco/msgpack/v4 v4.3.13 // indirect
	github.com/vmihailenco/tagparser v0.1.1 // indirect
	github.com/xlab/treeprint v1.1.0 // indirect
	github.com/yosida95/uritemplate/v3 v3.0.2 // indirect
	go.mongodb.org/mongo-driver v1.14.0 // indirect
	go.mozilla.org/pkcs7 v0.0.0-20210826202110-33d05740a352 // indirect
	go.opencensus.io v0.24.0 // indirect
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.53.0 // indirect
	go.opentelemetry.io/otel v1.28.0 // indirect
	go.opentelemetry.io/otel/metric v1.28.0 // indirect
	go.opentelemetry.io/otel/trace v1.28.0 // indirect
	go.starlark.net v0.0.0-20220328144851-d1966c6b9fcd // indirect
	go.uber.org/multierr v1.11.0 // indirect
	golang.org/x/mod v0.21.0 // indirect
	golang.org/x/sys v0.30.0 // indirect
	golang.org/x/term v0.29.0 // indirect
	golang.org/x/time v0.6.0 // indirect
	gomodules.xyz/jsonpatch/v2 v2.3.0 // indirect
	google.golang.org/appengine v1.6.8 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20240814211410-ddb44dafa142 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20240903143218-8af14fe29dc1 // indirect
	google.golang.org/grpc v1.67.0 // indirect
	google.golang.org/protobuf v1.35.2 // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
	gopkg.in/tomb.v1 v1.0.0-20141024135613-dd632973f1e7 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	k8s.io/apiserver v0.26.2 // indirect
	k8s.io/component-base v0.27.2 // indirect
	k8s.io/klog/v2 v2.130.1 // indirect
	k8s.io/kube-aggregator v0.27.1 // indirect
	k8s.io/kube-openapi v0.0.0-20230501164219-8b0f38b5fd1f // indirect
	sigs.k8s.io/json v0.0.0-20241009153224-e386a8af8d30 // indirect
	sigs.k8s.io/kube-storage-version-migrator v0.0.4 // indirect
	sigs.k8s.io/kustomize/api v0.12.1 // indirect
	sigs.k8s.io/kustomize/kyaml v0.13.9 // indirect
	sigs.k8s.io/structured-merge-diff/v4 v4.4.1 // indirect
	tags.cncf.io/container-device-interface v0.8.1 // indirect
)

// strip cloud.google.com/go dependencies -- these are not required as dependencies
exclude (
	cloud.google.com/go v0.37.4
	cloud.google.com/go v0.41.0
	cloud.google.com/go v0.44.1
	cloud.google.com/go v0.44.2
	cloud.google.com/go v0.44.3
	cloud.google.com/go v0.45.1
	cloud.google.com/go v0.46.3
	cloud.google.com/go v0.50.0
	cloud.google.com/go v0.52.0
	cloud.google.com/go v0.53.0
	cloud.google.com/go v0.54.0
	cloud.google.com/go v0.56.0
	cloud.google.com/go v0.57.0
	cloud.google.com/go v0.58.0
	cloud.google.com/go v0.75.0
	cloud.google.com/go v0.81.0
	cloud.google.com/go v0.97.0
	cloud.google.com/go v0.99.0
	cloud.google.com/go v0.100.1
	cloud.google.com/go v0.100.2
	cloud.google.com/go v0.102.0
	cloud.google.com/go v0.102.1
	cloud.google.com/go v0.104.0
	cloud.google.com/go v0.105.0
	cloud.google.com/go v0.107.0
	cloud.google.com/go v0.110.0
	cloud.google.com/go/accessapproval v1.4.0
	cloud.google.com/go/accessapproval v1.5.0
	cloud.google.com/go/accessapproval v1.6.0
	cloud.google.com/go/accesscontextmanager v1.3.0
	cloud.google.com/go/accesscontextmanager v1.4.0
	cloud.google.com/go/accesscontextmanager v1.6.0
	cloud.google.com/go/accesscontextmanager v1.7.0
	cloud.google.com/go/aiplatform v1.22.0
	cloud.google.com/go/aiplatform v1.24.0
	cloud.google.com/go/aiplatform v1.27.0
	cloud.google.com/go/aiplatform v1.35.0
	cloud.google.com/go/aiplatform v1.36.1
	cloud.google.com/go/aiplatform v1.37.0
	cloud.google.com/go/analytics v0.11.0
	cloud.google.com/go/analytics v0.12.0
	cloud.google.com/go/analytics v0.17.0
	cloud.google.com/go/analytics v0.18.0
	cloud.google.com/go/analytics v0.19.0
	cloud.google.com/go/apigateway v1.3.0
	cloud.google.com/go/apigateway v1.4.0
	cloud.google.com/go/apigateway v1.5.0
	cloud.google.com/go/apigeeconnect v1.3.0
	cloud.google.com/go/apigeeconnect v1.4.0
	cloud.google.com/go/apigeeconnect v1.5.0
	cloud.google.com/go/apigeeregistry v0.4.0
	cloud.google.com/go/apigeeregistry v0.5.0
	cloud.google.com/go/apigeeregistry v0.6.0
	cloud.google.com/go/apikeys v0.4.0
	cloud.google.com/go/apikeys v0.5.0
	cloud.google.com/go/apikeys v0.6.0
	cloud.google.com/go/appengine v1.4.0
	cloud.google.com/go/appengine v1.5.0
	cloud.google.com/go/appengine v1.6.0
	cloud.google.com/go/appengine v1.7.0
	cloud.google.com/go/appengine v1.7.1
	cloud.google.com/go/area120 v0.5.0
	cloud.google.com/go/area120 v0.6.0
	cloud.google.com/go/area120 v0.7.0
	cloud.google.com/go/area120 v0.7.1
	cloud.google.com/go/artifactregistry v1.6.0
	cloud.google.com/go/artifactregistry v1.7.0
	cloud.google.com/go/artifactregistry v1.8.0
	cloud.google.com/go/artifactregistry v1.9.0
	cloud.google.com/go/artifactregistry v1.11.1
	cloud.google.com/go/artifactregistry v1.11.2
	cloud.google.com/go/artifactregistry v1.12.0
	cloud.google.com/go/artifactregistry v1.13.0
	cloud.google.com/go/asset v1.5.0
	cloud.google.com/go/asset v1.7.0
	cloud.google.com/go/asset v1.8.0
	cloud.google.com/go/asset v1.9.0
	cloud.google.com/go/asset v1.10.0
	cloud.google.com/go/asset v1.11.1
	cloud.google.com/go/asset v1.12.0
	cloud.google.com/go/asset v1.13.0
	cloud.google.com/go/assuredworkloads v1.5.0
	cloud.google.com/go/assuredworkloads v1.6.0
	cloud.google.com/go/assuredworkloads v1.7.0
	cloud.google.com/go/assuredworkloads v1.8.0
	cloud.google.com/go/assuredworkloads v1.9.0
	cloud.google.com/go/assuredworkloads v1.10.0
	cloud.google.com/go/automl v1.5.0
	cloud.google.com/go/automl v1.6.0
	cloud.google.com/go/automl v1.7.0
	cloud.google.com/go/automl v1.8.0
	cloud.google.com/go/automl v1.12.0
	cloud.google.com/go/baremetalsolution v0.3.0
	cloud.google.com/go/baremetalsolution v0.4.0
	cloud.google.com/go/baremetalsolution v0.5.0
	cloud.google.com/go/batch v0.3.0
	cloud.google.com/go/batch v0.4.0
	cloud.google.com/go/batch v0.7.0
	cloud.google.com/go/beyondcorp v0.2.0
	cloud.google.com/go/beyondcorp v0.3.0
	cloud.google.com/go/beyondcorp v0.4.0
	cloud.google.com/go/beyondcorp v0.5.0
	cloud.google.com/go/bigquery v1.0.1
	cloud.google.com/go/bigquery v1.8.0
	cloud.google.com/go/bigquery v1.42.0
	cloud.google.com/go/bigquery v1.43.0
	cloud.google.com/go/bigquery v1.44.0
	cloud.google.com/go/bigquery v1.47.0
	cloud.google.com/go/bigquery v1.48.0
	cloud.google.com/go/bigquery v1.49.0
	cloud.google.com/go/bigquery v1.50.0
	cloud.google.com/go/billing v1.4.0
	cloud.google.com/go/billing v1.5.0
	cloud.google.com/go/billing v1.6.0
	cloud.google.com/go/billing v1.7.0
	cloud.google.com/go/billing v1.12.0
	cloud.google.com/go/billing v1.13.0
	cloud.google.com/go/binaryauthorization v1.1.0
	cloud.google.com/go/binaryauthorization v1.2.0
	cloud.google.com/go/binaryauthorization v1.3.0
	cloud.google.com/go/binaryauthorization v1.4.0
	cloud.google.com/go/binaryauthorization v1.5.0
	cloud.google.com/go/certificatemanager v1.3.0
	cloud.google.com/go/certificatemanager v1.4.0
	cloud.google.com/go/certificatemanager v1.6.0
	cloud.google.com/go/channel v1.8.0
	cloud.google.com/go/channel v1.9.0
	cloud.google.com/go/channel v1.11.0
	cloud.google.com/go/channel v1.12.0
	cloud.google.com/go/cloudbuild v1.3.0
	cloud.google.com/go/cloudbuild v1.4.0
	cloud.google.com/go/cloudbuild v1.6.0
	cloud.google.com/go/cloudbuild v1.7.0
	cloud.google.com/go/cloudbuild v1.9.0
	cloud.google.com/go/clouddms v1.3.0
	cloud.google.com/go/clouddms v1.4.0
	cloud.google.com/go/clouddms v1.5.0
	cloud.google.com/go/cloudtasks v1.5.0
	cloud.google.com/go/cloudtasks v1.6.0
	cloud.google.com/go/cloudtasks v1.7.0
	cloud.google.com/go/cloudtasks v1.8.0
	cloud.google.com/go/cloudtasks v1.9.0
	cloud.google.com/go/cloudtasks v1.10.0
	cloud.google.com/go/compute v0.1.0
	cloud.google.com/go/compute v1.3.0
	cloud.google.com/go/compute v1.5.0
	cloud.google.com/go/compute v1.6.0
	cloud.google.com/go/compute v1.6.1
	cloud.google.com/go/compute v1.7.0
	cloud.google.com/go/compute v1.10.0
	cloud.google.com/go/compute v1.12.0
	cloud.google.com/go/compute v1.12.1
	cloud.google.com/go/compute v1.13.0
	cloud.google.com/go/compute v1.14.0
	cloud.google.com/go/compute v1.18.0
	cloud.google.com/go/compute v1.19.0
	cloud.google.com/go/compute v1.19.1
	cloud.google.com/go/compute/metadata v0.1.0
	cloud.google.com/go/compute/metadata v0.2.1
	cloud.google.com/go/compute/metadata v0.2.3
	cloud.google.com/go/contactcenterinsights v1.3.0
	cloud.google.com/go/contactcenterinsights v1.4.0
	cloud.google.com/go/contactcenterinsights v1.6.0
	cloud.google.com/go/container v1.6.0
	cloud.google.com/go/container v1.7.0
	cloud.google.com/go/container v1.13.1
	cloud.google.com/go/container v1.14.0
	cloud.google.com/go/container v1.15.0
	cloud.google.com/go/containeranalysis v0.5.1
	cloud.google.com/go/containeranalysis v0.6.0
	cloud.google.com/go/containeranalysis v0.7.0
	cloud.google.com/go/containeranalysis v0.9.0
	cloud.google.com/go/datacatalog v1.3.0
	cloud.google.com/go/datacatalog v1.5.0
	cloud.google.com/go/datacatalog v1.6.0
	cloud.google.com/go/datacatalog v1.7.0
	cloud.google.com/go/datacatalog v1.8.0
	cloud.google.com/go/datacatalog v1.8.1
	cloud.google.com/go/datacatalog v1.12.0
	cloud.google.com/go/datacatalog v1.13.0
	cloud.google.com/go/dataflow v0.6.0
	cloud.google.com/go/dataflow v0.7.0
	cloud.google.com/go/dataflow v0.8.0
	cloud.google.com/go/dataform v0.3.0
	cloud.google.com/go/dataform v0.4.0
	cloud.google.com/go/dataform v0.5.0
	cloud.google.com/go/dataform v0.6.0
	cloud.google.com/go/dataform v0.7.0
	cloud.google.com/go/datafusion v1.4.0
	cloud.google.com/go/datafusion v1.5.0
	cloud.google.com/go/datafusion v1.6.0
	cloud.google.com/go/datalabeling v0.5.0
	cloud.google.com/go/datalabeling v0.6.0
	cloud.google.com/go/datalabeling v0.7.0
	cloud.google.com/go/dataplex v1.3.0
	cloud.google.com/go/dataplex v1.4.0
	cloud.google.com/go/dataplex v1.5.2
	cloud.google.com/go/dataplex v1.6.0
	cloud.google.com/go/dataproc v1.7.0
	cloud.google.com/go/dataproc v1.8.0
	cloud.google.com/go/dataproc v1.12.0
	cloud.google.com/go/dataqna v0.5.0
	cloud.google.com/go/dataqna v0.6.0
	cloud.google.com/go/dataqna v0.7.0
	cloud.google.com/go/datastore v1.0.0
	cloud.google.com/go/datastore v1.10.0
	cloud.google.com/go/datastore v1.11.0
	cloud.google.com/go/datastream v1.2.0
	cloud.google.com/go/datastream v1.3.0
	cloud.google.com/go/datastream v1.4.0
	cloud.google.com/go/datastream v1.5.0
	cloud.google.com/go/datastream v1.6.0
	cloud.google.com/go/datastream v1.7.0
	cloud.google.com/go/deploy v1.4.0
	cloud.google.com/go/deploy v1.5.0
	cloud.google.com/go/deploy v1.6.0
	cloud.google.com/go/deploy v1.8.0
	cloud.google.com/go/dialogflow v1.15.0
	cloud.google.com/go/dialogflow v1.16.1
	cloud.google.com/go/dialogflow v1.17.0
	cloud.google.com/go/dialogflow v1.18.0
	cloud.google.com/go/dialogflow v1.19.0
	cloud.google.com/go/dialogflow v1.29.0
	cloud.google.com/go/dialogflow v1.31.0
	cloud.google.com/go/dialogflow v1.32.0
	cloud.google.com/go/dlp v1.6.0
	cloud.google.com/go/dlp v1.7.0
	cloud.google.com/go/dlp v1.9.0
	cloud.google.com/go/documentai v1.7.0
	cloud.google.com/go/documentai v1.8.0
	cloud.google.com/go/documentai v1.9.0
	cloud.google.com/go/documentai v1.10.0
	cloud.google.com/go/documentai v1.16.0
	cloud.google.com/go/documentai v1.18.0
	cloud.google.com/go/domains v0.6.0
	cloud.google.com/go/domains v0.7.0
	cloud.google.com/go/domains v0.8.0
	cloud.google.com/go/edgecontainer v0.1.0
	cloud.google.com/go/edgecontainer v0.2.0
	cloud.google.com/go/edgecontainer v0.3.0
	cloud.google.com/go/edgecontainer v1.0.0
	cloud.google.com/go/errorreporting v0.3.0
	cloud.google.com/go/essentialcontacts v1.3.0
	cloud.google.com/go/essentialcontacts v1.4.0
	cloud.google.com/go/essentialcontacts v1.5.0
	cloud.google.com/go/eventarc v1.7.0
	cloud.google.com/go/eventarc v1.8.0
	cloud.google.com/go/eventarc v1.10.0
	cloud.google.com/go/eventarc v1.11.0
	cloud.google.com/go/filestore v1.3.0
	cloud.google.com/go/filestore v1.4.0
	cloud.google.com/go/filestore v1.5.0
	cloud.google.com/go/filestore v1.6.0
	cloud.google.com/go/firestore v1.1.0
	cloud.google.com/go/firestore v1.9.0
	cloud.google.com/go/functions v1.6.0
	cloud.google.com/go/functions v1.7.0
	cloud.google.com/go/functions v1.8.0
	cloud.google.com/go/functions v1.9.0
	cloud.google.com/go/functions v1.10.0
	cloud.google.com/go/functions v1.12.0
	cloud.google.com/go/functions v1.13.0
	cloud.google.com/go/gaming v1.5.0
	cloud.google.com/go/gaming v1.6.0
	cloud.google.com/go/gaming v1.7.0
	cloud.google.com/go/gaming v1.8.0
	cloud.google.com/go/gaming v1.9.0
	cloud.google.com/go/gkebackup v0.2.0
	cloud.google.com/go/gkebackup v0.3.0
	cloud.google.com/go/gkebackup v0.4.0
	cloud.google.com/go/gkeconnect v0.5.0
	cloud.google.com/go/gkeconnect v0.6.0
	cloud.google.com/go/gkeconnect v0.7.0
	cloud.google.com/go/gkehub v0.9.0
	cloud.google.com/go/gkehub v0.10.0
	cloud.google.com/go/gkehub v0.11.0
	cloud.google.com/go/gkehub v0.12.0
	cloud.google.com/go/gkemulticloud v0.3.0
	cloud.google.com/go/gkemulticloud v0.4.0
	cloud.google.com/go/gkemulticloud v0.5.0
	cloud.google.com/go/grafeas v0.2.0
	cloud.google.com/go/gsuiteaddons v1.3.0
	cloud.google.com/go/gsuiteaddons v1.4.0
	cloud.google.com/go/gsuiteaddons v1.5.0
	cloud.google.com/go/iam v0.1.0
	cloud.google.com/go/iam v0.3.0
	cloud.google.com/go/iam v0.5.0
	cloud.google.com/go/iam v0.6.0
	cloud.google.com/go/iam v0.7.0
	cloud.google.com/go/iam v0.8.0
	cloud.google.com/go/iam v0.11.0
	cloud.google.com/go/iam v0.12.0
	cloud.google.com/go/iam v0.13.0
	cloud.google.com/go/iap v1.4.0
	cloud.google.com/go/iap v1.5.0
	cloud.google.com/go/iap v1.6.0
	cloud.google.com/go/iap v1.7.0
	cloud.google.com/go/iap v1.7.1
	cloud.google.com/go/ids v1.1.0
	cloud.google.com/go/ids v1.2.0
	cloud.google.com/go/ids v1.3.0
	cloud.google.com/go/iot v1.3.0
	cloud.google.com/go/iot v1.4.0
	cloud.google.com/go/iot v1.5.0
	cloud.google.com/go/iot v1.6.0
	cloud.google.com/go/kms v1.4.0
	cloud.google.com/go/kms v1.5.0
	cloud.google.com/go/kms v1.6.0
	cloud.google.com/go/kms v1.8.0
	cloud.google.com/go/kms v1.9.0
	cloud.google.com/go/kms v1.10.0
	cloud.google.com/go/kms v1.10.1
	cloud.google.com/go/language v1.4.0
	cloud.google.com/go/language v1.6.0
	cloud.google.com/go/language v1.7.0
	cloud.google.com/go/language v1.8.0
	cloud.google.com/go/language v1.9.0
	cloud.google.com/go/lifesciences v0.5.0
	cloud.google.com/go/lifesciences v0.6.0
	cloud.google.com/go/lifesciences v0.8.0
	cloud.google.com/go/logging v1.6.1
	cloud.google.com/go/logging v1.7.0
	cloud.google.com/go/longrunning v0.1.1
	cloud.google.com/go/longrunning v0.3.0
	cloud.google.com/go/longrunning v0.4.1
	cloud.google.com/go/managedidentities v1.3.0
	cloud.google.com/go/managedidentities v1.4.0
	cloud.google.com/go/managedidentities v1.5.0
	cloud.google.com/go/maps v0.1.0
	cloud.google.com/go/maps v0.6.0
	cloud.google.com/go/maps v0.7.0
	cloud.google.com/go/mediatranslation v0.5.0
	cloud.google.com/go/mediatranslation v0.6.0
	cloud.google.com/go/mediatranslation v0.7.0
	cloud.google.com/go/memcache v1.4.0
	cloud.google.com/go/memcache v1.5.0
	cloud.google.com/go/memcache v1.6.0
	cloud.google.com/go/memcache v1.7.0
	cloud.google.com/go/memcache v1.9.0
	cloud.google.com/go/metastore v1.5.0
	cloud.google.com/go/metastore v1.6.0
	cloud.google.com/go/metastore v1.7.0
	cloud.google.com/go/metastore v1.8.0
	cloud.google.com/go/metastore v1.10.0
	cloud.google.com/go/monitoring v1.7.0
	cloud.google.com/go/monitoring v1.8.0
	cloud.google.com/go/monitoring v1.12.0
	cloud.google.com/go/monitoring v1.13.0
	cloud.google.com/go/networkconnectivity v1.4.0
	cloud.google.com/go/networkconnectivity v1.5.0
	cloud.google.com/go/networkconnectivity v1.6.0
	cloud.google.com/go/networkconnectivity v1.7.0
	cloud.google.com/go/networkconnectivity v1.10.0
	cloud.google.com/go/networkconnectivity v1.11.0
	cloud.google.com/go/networkmanagement v1.4.0
	cloud.google.com/go/networkmanagement v1.5.0
	cloud.google.com/go/networkmanagement v1.6.0
	cloud.google.com/go/networksecurity v0.5.0
	cloud.google.com/go/networksecurity v0.6.0
	cloud.google.com/go/networksecurity v0.7.0
	cloud.google.com/go/networksecurity v0.8.0
	cloud.google.com/go/notebooks v1.2.0
	cloud.google.com/go/notebooks v1.3.0
	cloud.google.com/go/notebooks v1.4.0
	cloud.google.com/go/notebooks v1.5.0
	cloud.google.com/go/notebooks v1.7.0
	cloud.google.com/go/notebooks v1.8.0
	cloud.google.com/go/optimization v1.1.0
	cloud.google.com/go/optimization v1.2.0
	cloud.google.com/go/optimization v1.3.1
	cloud.google.com/go/orchestration v1.3.0
	cloud.google.com/go/orchestration v1.4.0
	cloud.google.com/go/orchestration v1.6.0
	cloud.google.com/go/orgpolicy v1.4.0
	cloud.google.com/go/orgpolicy v1.5.0
	cloud.google.com/go/orgpolicy v1.10.0
	cloud.google.com/go/osconfig v1.7.0
	cloud.google.com/go/osconfig v1.8.0
	cloud.google.com/go/osconfig v1.9.0
	cloud.google.com/go/osconfig v1.10.0
	cloud.google.com/go/osconfig v1.11.0
	cloud.google.com/go/oslogin v1.4.0
	cloud.google.com/go/oslogin v1.5.0
	cloud.google.com/go/oslogin v1.6.0
	cloud.google.com/go/oslogin v1.7.0
	cloud.google.com/go/oslogin v1.9.0
	cloud.google.com/go/phishingprotection v0.5.0
	cloud.google.com/go/phishingprotection v0.6.0
	cloud.google.com/go/phishingprotection v0.7.0
	cloud.google.com/go/policytroubleshooter v1.3.0
	cloud.google.com/go/policytroubleshooter v1.4.0
	cloud.google.com/go/policytroubleshooter v1.5.0
	cloud.google.com/go/policytroubleshooter v1.6.0
	cloud.google.com/go/privatecatalog v0.5.0
	cloud.google.com/go/privatecatalog v0.6.0
	cloud.google.com/go/privatecatalog v0.7.0
	cloud.google.com/go/privatecatalog v0.8.0
	cloud.google.com/go/pubsub v1.26.0
	cloud.google.com/go/pubsub v1.27.1
	cloud.google.com/go/pubsub v1.28.0
	cloud.google.com/go/pubsub v1.30.0
	cloud.google.com/go/pubsublite v1.5.0
	cloud.google.com/go/pubsublite v1.6.0
	cloud.google.com/go/pubsublite v1.7.0
	cloud.google.com/go/recaptchaenterprise v1.3.1
	cloud.google.com/go/recaptchaenterprise/v2 v2.1.0
	cloud.google.com/go/recaptchaenterprise/v2 v2.2.0
	cloud.google.com/go/recaptchaenterprise/v2 v2.3.0
	cloud.google.com/go/recaptchaenterprise/v2 v2.4.0
	cloud.google.com/go/recaptchaenterprise/v2 v2.5.0
	cloud.google.com/go/recaptchaenterprise/v2 v2.6.0
	cloud.google.com/go/recaptchaenterprise/v2 v2.7.0
	cloud.google.com/go/recommendationengine v0.5.0
	cloud.google.com/go/recommendationengine v0.6.0
	cloud.google.com/go/recommendationengine v0.7.0
	cloud.google.com/go/recommender v1.5.0
	cloud.google.com/go/recommender v1.6.0
	cloud.google.com/go/recommender v1.7.0
	cloud.google.com/go/recommender v1.8.0
	cloud.google.com/go/recommender v1.9.0
	cloud.google.com/go/redis v1.7.0
	cloud.google.com/go/redis v1.8.0
	cloud.google.com/go/redis v1.9.0
	cloud.google.com/go/redis v1.10.0
	cloud.google.com/go/redis v1.11.0
	cloud.google.com/go/resourcemanager v1.3.0
	cloud.google.com/go/resourcemanager v1.4.0
	cloud.google.com/go/resourcemanager v1.5.0
	cloud.google.com/go/resourcemanager v1.6.0
	cloud.google.com/go/resourcemanager v1.7.0
	cloud.google.com/go/resourcesettings v1.3.0
	cloud.google.com/go/resourcesettings v1.4.0
	cloud.google.com/go/resourcesettings v1.5.0
	cloud.google.com/go/retail v1.8.0
	cloud.google.com/go/retail v1.9.0
	cloud.google.com/go/retail v1.10.0
	cloud.google.com/go/retail v1.11.0
	cloud.google.com/go/retail v1.12.0
	cloud.google.com/go/run v0.2.0
	cloud.google.com/go/run v0.3.0
	cloud.google.com/go/run v0.8.0
	cloud.google.com/go/run v0.9.0
	cloud.google.com/go/scheduler v1.4.0
	cloud.google.com/go/scheduler v1.5.0
	cloud.google.com/go/scheduler v1.6.0
	cloud.google.com/go/scheduler v1.7.0
	cloud.google.com/go/scheduler v1.8.0
	cloud.google.com/go/scheduler v1.9.0
	cloud.google.com/go/secretmanager v1.6.0
	cloud.google.com/go/secretmanager v1.8.0
	cloud.google.com/go/secretmanager v1.9.0
	cloud.google.com/go/secretmanager v1.10.0
	cloud.google.com/go/security v1.5.0
	cloud.google.com/go/security v1.7.0
	cloud.google.com/go/security v1.8.0
	cloud.google.com/go/security v1.9.0
	cloud.google.com/go/security v1.10.0
	cloud.google.com/go/security v1.12.0
	cloud.google.com/go/security v1.13.0
	cloud.google.com/go/securitycenter v1.13.0
	cloud.google.com/go/securitycenter v1.14.0
	cloud.google.com/go/securitycenter v1.15.0
	cloud.google.com/go/securitycenter v1.16.0
	cloud.google.com/go/securitycenter v1.18.1
	cloud.google.com/go/securitycenter v1.19.0
	cloud.google.com/go/servicecontrol v1.4.0
	cloud.google.com/go/servicecontrol v1.5.0
	cloud.google.com/go/servicecontrol v1.10.0
	cloud.google.com/go/servicecontrol v1.11.0
	cloud.google.com/go/servicecontrol v1.11.1
	cloud.google.com/go/servicedirectory v1.4.0
	cloud.google.com/go/servicedirectory v1.5.0
	cloud.google.com/go/servicedirectory v1.6.0
	cloud.google.com/go/servicedirectory v1.7.0
	cloud.google.com/go/servicedirectory v1.8.0
	cloud.google.com/go/servicedirectory v1.9.0
	cloud.google.com/go/servicemanagement v1.4.0
	cloud.google.com/go/servicemanagement v1.5.0
	cloud.google.com/go/servicemanagement v1.6.0
	cloud.google.com/go/servicemanagement v1.8.0
	cloud.google.com/go/serviceusage v1.3.0
	cloud.google.com/go/serviceusage v1.4.0
	cloud.google.com/go/serviceusage v1.5.0
	cloud.google.com/go/serviceusage v1.6.0
	cloud.google.com/go/shell v1.3.0
	cloud.google.com/go/shell v1.4.0
	cloud.google.com/go/shell v1.6.0
	cloud.google.com/go/spanner v1.41.0
	cloud.google.com/go/spanner v1.44.0
	cloud.google.com/go/spanner v1.45.0
	cloud.google.com/go/speech v1.6.0
	cloud.google.com/go/speech v1.7.0
	cloud.google.com/go/speech v1.8.0
	cloud.google.com/go/speech v1.9.0
	cloud.google.com/go/speech v1.14.1
	cloud.google.com/go/speech v1.15.0
	cloud.google.com/go/storage v1.0.0
	cloud.google.com/go/storage v1.5.0
	cloud.google.com/go/storage v1.6.0
	cloud.google.com/go/storage v1.8.0
	cloud.google.com/go/storage v1.9.0
	cloud.google.com/go/storage v1.14.0
	cloud.google.com/go/storage v1.23.0
	cloud.google.com/go/storage v1.27.0
	cloud.google.com/go/storage v1.28.1
	cloud.google.com/go/storage v1.29.0
	cloud.google.com/go/storagetransfer v1.5.0
	cloud.google.com/go/storagetransfer v1.6.0
	cloud.google.com/go/storagetransfer v1.7.0
	cloud.google.com/go/storagetransfer v1.8.0
	cloud.google.com/go/talent v1.1.0
	cloud.google.com/go/talent v1.2.0
	cloud.google.com/go/talent v1.3.0
	cloud.google.com/go/talent v1.4.0
	cloud.google.com/go/talent v1.5.0
	cloud.google.com/go/texttospeech v1.4.0
	cloud.google.com/go/texttospeech v1.5.0
	cloud.google.com/go/texttospeech v1.6.0
	cloud.google.com/go/tpu v1.3.0
	cloud.google.com/go/tpu v1.4.0
	cloud.google.com/go/tpu v1.5.0
	cloud.google.com/go/trace v1.3.0
	cloud.google.com/go/trace v1.4.0
	cloud.google.com/go/trace v1.8.0
	cloud.google.com/go/trace v1.9.0
	cloud.google.com/go/translate v1.3.0
	cloud.google.com/go/translate v1.4.0
	cloud.google.com/go/translate v1.5.0
	cloud.google.com/go/translate v1.6.0
	cloud.google.com/go/translate v1.7.0
	cloud.google.com/go/video v1.8.0
	cloud.google.com/go/video v1.9.0
	cloud.google.com/go/video v1.12.0
	cloud.google.com/go/video v1.13.0
	cloud.google.com/go/video v1.14.0
	cloud.google.com/go/video v1.15.0
	cloud.google.com/go/videointelligence v1.6.0
	cloud.google.com/go/videointelligence v1.7.0
	cloud.google.com/go/videointelligence v1.8.0
	cloud.google.com/go/videointelligence v1.9.0
	cloud.google.com/go/videointelligence v1.10.0
	cloud.google.com/go/vision v1.2.0
	cloud.google.com/go/vision/v2 v2.2.0
	cloud.google.com/go/vision/v2 v2.3.0
	cloud.google.com/go/vision/v2 v2.4.0
	cloud.google.com/go/vision/v2 v2.5.0
	cloud.google.com/go/vision/v2 v2.6.0
	cloud.google.com/go/vision/v2 v2.7.0
	cloud.google.com/go/vmmigration v1.2.0
	cloud.google.com/go/vmmigration v1.3.0
	cloud.google.com/go/vmmigration v1.5.0
	cloud.google.com/go/vmmigration v1.6.0
	cloud.google.com/go/vmwareengine v0.1.0
	cloud.google.com/go/vmwareengine v0.2.2
	cloud.google.com/go/vmwareengine v0.3.0
	cloud.google.com/go/vpcaccess v1.4.0
	cloud.google.com/go/vpcaccess v1.5.0
	cloud.google.com/go/vpcaccess v1.6.0
	cloud.google.com/go/webrisk v1.4.0
	cloud.google.com/go/webrisk v1.5.0
	cloud.google.com/go/webrisk v1.6.0
	cloud.google.com/go/webrisk v1.7.0
	cloud.google.com/go/webrisk v1.8.0
	cloud.google.com/go/websecurityscanner v1.3.0
	cloud.google.com/go/websecurityscanner v1.4.0
	cloud.google.com/go/websecurityscanner v1.5.0
	cloud.google.com/go/workflows v1.6.0
	cloud.google.com/go/workflows v1.7.0
	cloud.google.com/go/workflows v1.8.0
	cloud.google.com/go/workflows v1.9.0
	cloud.google.com/go/workflows v1.10.0
)

// remove old github.com/google, github.com/googleapis/ google.golang.org dependencies
exclude (
	github.com/google/go-cmp v0.3.0
	github.com/google/go-cmp v0.3.1
	github.com/google/go-cmp v0.4.0
	github.com/google/go-cmp v0.5.0
	github.com/google/go-cmp v0.5.1
	github.com/google/go-cmp v0.5.2
	github.com/google/go-cmp v0.5.3
	github.com/google/go-cmp v0.5.4
	github.com/google/go-cmp v0.5.5
	github.com/google/go-cmp v0.5.6
	github.com/google/go-cmp v0.5.8
	github.com/google/go-cmp v0.5.9
	github.com/google/go-containerregistry v0.5.1
	github.com/google/gofuzz v1.0.0
	github.com/google/gofuzz v1.1.0
	github.com/google/pprof v0.0.0-20181127221834-b4f47329b966
	github.com/google/pprof v0.0.0-20210407192527-94a9f03dee38
	github.com/google/uuid v0.0.0-20170306145142-6a5e28554805
	github.com/google/uuid v1.1.1
	github.com/google/uuid v1.1.2
	github.com/google/uuid v1.2.0
	github.com/googleapis/google-cloud-go-testing v0.0.0-20200911160855-bcd43fbb19e8
	google.golang.org/api v0.3.1
	google.golang.org/api v0.3.2
	google.golang.org/api v0.4.0
	google.golang.org/api v0.7.0
	google.golang.org/api v0.8.0
	google.golang.org/api v0.9.0
	google.golang.org/api v0.13.0
	google.golang.org/api v0.14.0
	google.golang.org/api v0.15.0
	google.golang.org/api v0.17.0
	google.golang.org/api v0.18.0
	google.golang.org/api v0.19.0
	google.golang.org/api v0.20.0
	google.golang.org/api v0.22.0
	google.golang.org/api v0.24.0
	google.golang.org/api v0.26.0
	google.golang.org/api v0.28.0
	google.golang.org/api v0.29.0
	google.golang.org/api v0.30.0
	google.golang.org/api v0.35.0
	google.golang.org/api v0.36.0
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
	google.golang.org/appengine v1.1.0
	google.golang.org/cloud v0.0.0-20151119220103-975617b05ea8
	google.golang.org/genproto v0.0.0-20190819201941-24fa4b261c55
	google.golang.org/genproto v0.0.0-20200224152610-e50cd9704f63
	google.golang.org/genproto v0.0.0-20200423170343-7949de9c1215
	google.golang.org/genproto v0.0.0-20200513103714-09dca8ec2884
	google.golang.org/genproto v0.0.0-20200527145253-8367513e4ece
	google.golang.org/genproto v0.0.0-20200610104632-a5b850bcf112
	google.golang.org/genproto v0.0.0-20201019141844-1ed22bb0c154
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
	google.golang.org/protobuf v1.26.0-rc.1
	google.golang.org/protobuf v1.26.0
	google.golang.org/protobuf v1.27.1
	google.golang.org/protobuf v1.28.0
	google.golang.org/protobuf v1.28.1
	google.golang.org/protobuf v1.29.1
	google.golang.org/protobuf v1.30.0
)

exclude (
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
	github.com/Azure/go-autorest/autorest v0.9.0
	github.com/Azure/go-autorest/autorest v0.11.12
	github.com/Azure/go-autorest/autorest v0.11.18
	github.com/Azure/go-autorest/autorest v0.11.24
	github.com/Azure/go-autorest/autorest v0.11.27
	github.com/Azure/go-autorest/autorest/adal v0.9.5
	github.com/Azure/go-autorest/autorest/adal v0.9.13
	github.com/Azure/go-autorest/autorest/adal v0.9.18
	github.com/Azure/go-autorest/autorest/adal v0.9.20
	github.com/Azure/go-autorest/autorest/adal v0.9.22
	github.com/Azure/go-autorest/autorest/mocks v0.4.1
	github.com/Azure/go-autorest/autorest/to v0.3.0
	github.com/Azure/go-autorest/autorest/validation v0.1.0
	github.com/BurntSushi/toml v0.3.1
	github.com/BurntSushi/toml v1.2.0
	github.com/Microsoft/go-winio v0.4.14
	github.com/asaskevich/govalidator v0.0.0-20190424111038-f61b66f89f4a
	github.com/asaskevich/govalidator v0.0.0-20200907205600-7a23bdc65eef
	github.com/chai2010/gettext-go v0.0.0-20160711120539-c6fed771bfd5
	github.com/chzyer/readline v0.0.0-20180603132655-2972be24d48e
	github.com/cilium/ebpf v0.4.0
	// exclude old containerd versions
	github.com/containerd/cgroups v1.0.1
	github.com/containerd/containerd v1.2.7
	github.com/containerd/containerd v1.2.10
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
	github.com/containerd/stargz-snapshotter/estargz v0.4.1
	github.com/containerd/stargz-snapshotter/estargz v0.12.0
	github.com/containers/storage v1.43.0
	// remove ancient dockers
	github.com/docker/distribution v0.0.0-20180920194744-16128bbac47f
	github.com/docker/distribution v0.0.0-20191216044856-a8371794149d
	github.com/docker/distribution v2.7.0+incompatible
	github.com/docker/distribution v2.7.1+incompatible
	github.com/docker/docker v1.4.2-0.20190924003213-a8608b5b67c7
	github.com/docker/docker-credential-helpers v0.6.3
	github.com/docker/go-connections v0.4.0
	github.com/docker/go-units v0.4.0
	// Remove old goproxy versions
	github.com/elazarl/goproxy v0.0.0-20180725130230-947c36da3153
	github.com/elazarl/goproxy v0.0.0-20190911111923-ecfe977594f1
	// Remove unneeded go-restful v2
	github.com/emicklei/go-restful v2.15.0+incompatible
	github.com/go-logr/logr v0.2.0
	github.com/go-logr/logr v0.4.0
	github.com/go-logr/logr v1.2.0
	github.com/go-logr/logr v1.2.2
	github.com/go-logr/logr v1.2.3
	// exclude github.com/golang/protobuf < 1.3.2 https://nvd.nist.gov/vuln/detail/CVE-2021-3121
	github.com/gogo/protobuf v1.0.0
	github.com/gogo/protobuf v1.1.1
	github.com/gogo/protobuf v1.2.0
	github.com/gogo/protobuf v1.2.1
	github.com/gogo/protobuf v1.3.0
	github.com/gogo/protobuf v1.3.1
	github.com/munnerz/goautoneg v0.0.0-20120707110453-a547fc61f48d
	// remove old runc
	github.com/opencontainers/runc v1.0.2
	github.com/opencontainers/runc v1.1.4
	// exclude old openshift library-go
	github.com/openshift/library-go v0.0.0-20211220195323-eca2c467c492
	github.com/openshift/library-go v0.0.0-20220121154930-b7889002d63e
	github.com/pkg/errors v0.8.1
	github.com/pkg/sftp v1.10.1
	github.com/pkg/sftp v1.13.1
	// remove old prometheus deps
	github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring v0.44.1
	github.com/prometheus/client_golang v1.0.0
	github.com/prometheus/client_golang v1.11.0
	github.com/prometheus/client_golang v1.11.1
	github.com/prometheus/client_golang v1.12.1
	github.com/prometheus/client_model v0.0.0-20190129233127-fd36f4220a90
	github.com/prometheus/client_model v0.0.0-20190812154241-14fe0d1b01d4
	github.com/prometheus/client_model v0.2.0
	github.com/prometheus/client_model v0.3.0
	github.com/prometheus/common v0.4.1
	github.com/prometheus/common v0.10.0
	github.com/prometheus/common v0.15.0
	github.com/prometheus/common v0.26.0
	github.com/prometheus/common v0.28.0
	github.com/prometheus/common v0.32.1
	github.com/prometheus/procfs v0.0.2
	github.com/prometheus/procfs v0.6.0
	github.com/prometheus/procfs v0.7.3
	github.com/russross/blackfriday v1.5.2
	github.com/sirupsen/logrus v1.4.1
	github.com/sirupsen/logrus v1.6.0
	github.com/sirupsen/logrus v1.7.0
	github.com/sirupsen/logrus v1.8.1
	github.com/sirupsen/logrus v1.9.0
	// trip dependency tree from old cobra
	github.com/spf13/cobra v0.0.5
	github.com/spf13/cobra v1.0.0
	github.com/spf13/cobra v1.1.1
	github.com/spf13/cobra v1.1.3
	github.com/spf13/cobra v1.2.1
	github.com/spf13/cobra v1.4.0
	github.com/spf13/pflag v1.0.3
	github.com/spf13/pflag v1.0.5
	github.com/stretchr/objx v0.1.0
	github.com/stretchr/objx v0.1.1
	github.com/stretchr/objx v0.2.0
	github.com/stretchr/objx v0.4.0
	github.com/stretchr/testify v1.2.2
	github.com/stretchr/testify v1.3.0
	github.com/stretchr/testify v1.4.0
	github.com/stretchr/testify v1.5.1
	github.com/stretchr/testify v1.6.1
	github.com/stretchr/testify v1.7.0
	github.com/stretchr/testify v1.7.1
	github.com/stretchr/testify v1.8.0
	github.com/stretchr/testify v1.8.1
	github.com/stretchr/testify v1.8.2
	github.com/xeipuuv/gojsonpointer v0.0.0-20180127040702-4e3ac2762d5f
	go.mozilla.org/pkcs7 v0.0.0-20200128120323-432b2356ecb1
	// trim dependency tree from old opencensus
	go.opencensus.io v0.20.1
	go.opencensus.io v0.20.2
	go.opencensus.io v0.21.0
	go.opencensus.io v0.22.0
	go.opencensus.io v0.22.2
	go.opencensus.io v0.22.3
	go.opencensus.io v0.22.4
	go.opencensus.io v0.22.5
	go.opencensus.io v0.23.0
	// old otel deps
	go.opentelemetry.io/contrib v0.20.0
	go.opentelemetry.io/otel v0.20.0
	go.opentelemetry.io/otel/metric v0.20.0
	go.opentelemetry.io/otel/sdk v0.20.0
	go.opentelemetry.io/otel/sdk/metric v0.20.0
	go.opentelemetry.io/otel/trace v0.20.0
	go.opentelemetry.io/proto/otlp v0.7.0
	go.opentelemetry.io/proto/otlp v0.19.0
	go.starlark.net v0.0.0-20200306205701-8dd3e2ee1dd5
	go.uber.org/atomic v1.4.0
	go.uber.org/atomic v1.7.0
	go.uber.org/goleak v1.1.10
	go.uber.org/goleak v1.1.11-0.20210813005559-691160354723
	go.uber.org/goleak v1.1.12
	go.uber.org/goleak v1.2.0
	go.uber.org/multierr v1.1.0
	go.uber.org/multierr v1.6.0
	go.uber.org/zap v1.10.0
	go.uber.org/zap v1.17.0
	go.uber.org/zap v1.19.0
	go.uber.org/zap v1.19.1
	gopkg.in/check.v1 v0.0.0-20161208181325-20d25e280405
	gopkg.in/check.v1 v1.0.0-20180628173108-788fd7840127
	gopkg.in/check.v1 v1.0.0-20190902080502-41f04d3bba15
	gopkg.in/check.v1 v1.0.0-20200227125254-8fa46927fb4f
	gopkg.in/yaml.v2 v2.2.1
	gopkg.in/yaml.v2 v2.2.2
	gopkg.in/yaml.v2 v2.2.4
	gopkg.in/yaml.v2 v2.2.8
	gopkg.in/yaml.v2 v2.3.0
	gopkg.in/yaml.v3 v3.0.0-20200313102051-9f266ea9e77c
	gopkg.in/yaml.v3 v3.0.0-20200605160147-a5ece683394c
	gopkg.in/yaml.v3 v3.0.0-20200615113413-eeeca48fe776
	gopkg.in/yaml.v3 v3.0.0-20210107192922-496545a6307b
	gopkg.in/yaml.v3 v3.0.0
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
	golang.org/x/crypto v0.14.0
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
	golang.org/x/tools v0.6.0
	golang.org/x/tools v0.7.0
	golang.org/x/xerrors v0.0.0-20190717185122-a985d3407aa7
	golang.org/x/xerrors v0.0.0-20191204190536-9bdfabe68543
	golang.org/x/xerrors v0.0.0-20200804184101-5ec99f83aff1
	golang.org/x/xerrors v0.0.0-20220907171357-04be3eba64a2
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
	k8s.io/api v0.23.0
	k8s.io/api v0.23.1
	k8s.io/apiextensions-apiserver v0.0.0
	k8s.io/apiextensions-apiserver v0.18.0-beta.2
	k8s.io/apiextensions-apiserver v0.18.3
	k8s.io/apiextensions-apiserver v0.19.2
	k8s.io/apiextensions-apiserver v0.19.3
	k8s.io/apiextensions-apiserver v0.21.0
	k8s.io/apiextensions-apiserver v0.21.1
	k8s.io/apiextensions-apiserver v0.22.1
	k8s.io/apiextensions-apiserver v0.23.0
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
	k8s.io/apimachinery v0.23.0
	k8s.io/apimachinery v0.23.1
	k8s.io/apimachinery v0.23.5
	k8s.io/apiserver v0.0.0
	k8s.io/apiserver v0.20.6
	k8s.io/apiserver v0.21.0
	k8s.io/apiserver v0.22.1
	k8s.io/apiserver v0.23.0
	k8s.io/apiserver v0.23.1
	k8s.io/apiserver v0.23.5
	k8s.io/cli-runtime v0.0.0
	k8s.io/cli-runtime v0.21.0
	k8s.io/cli-runtime v0.23.0
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
	k8s.io/client-go v0.23.0
	k8s.io/client-go v0.23.1
	k8s.io/client-go v0.23.5
	k8s.io/code-generator v0.0.0
	k8s.io/code-generator v0.18.0-beta.2
	k8s.io/code-generator v0.19.7
	k8s.io/code-generator v0.20.0
	k8s.io/code-generator v0.21.0
	k8s.io/code-generator v0.23.0
	k8s.io/component-base v0.0.0
	k8s.io/component-base v0.19.2
	k8s.io/component-base v0.19.4
	k8s.io/component-base v0.20.6
	k8s.io/component-base v0.21.0
	k8s.io/component-base v0.21.1
	k8s.io/component-base v0.22.1
	k8s.io/component-base v0.23.0
	k8s.io/component-base v0.23.1
	k8s.io/component-base v0.23.5
	k8s.io/controller-manager v0.0.0
	k8s.io/cri-api v0.0.0
	k8s.io/cri-api v0.20.6
	k8s.io/gengo v0.0.0-20201113003025-83324d819ded
	k8s.io/gengo v0.0.0-20210813121822-485abfe95c7c
	k8s.io/klog v1.0.0
	k8s.io/klog/v2 v2.0.0
	k8s.io/klog/v2 v2.2.0
	k8s.io/klog/v2 v2.4.0
	k8s.io/klog/v2 v2.8.0
	k8s.io/klog/v2 v2.9.0
	k8s.io/klog/v2 v2.30.0
	k8s.io/klog/v2 v2.40.1
	k8s.io/klog/v2 v2.60.1
	k8s.io/klog/v2 v2.70.1
	k8s.io/kube-aggregator v0.0.0
	k8s.io/kube-aggregator v0.18.0-beta.2
	k8s.io/kube-aggregator v0.23.0
	k8s.io/kube-openapi v0.0.0-20210421082810-95288971da7e
	k8s.io/kube-openapi v0.0.0-20211115234752-e816edb12b65
	k8s.io/kube-openapi v0.0.0-20220124234850-424119656bbf
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
	k8s.io/utils v0.0.0-20220728103510-ee6ede2d64ed
	sigs.k8s.io/apiserver-network-proxy/konnectivity-client v0.0.22
	sigs.k8s.io/apiserver-network-proxy/konnectivity-client v0.0.25
	sigs.k8s.io/apiserver-network-proxy/konnectivity-client v0.0.30
	sigs.k8s.io/controller-runtime v0.7.0
	sigs.k8s.io/controller-runtime v0.9.0-beta.1.0.20210512131817-ce2f0c92d77e
	sigs.k8s.io/controller-runtime v0.9.0
	sigs.k8s.io/controller-tools v0.2.8
	sigs.k8s.io/controller-tools v0.3.0
	sigs.k8s.io/controller-tools v0.4.1
	sigs.k8s.io/controller-tools v0.6.0
	sigs.k8s.io/controller-tools v0.6.2
	sigs.k8s.io/controller-tools v0.7.0
	sigs.k8s.io/json v0.0.0-20211020170558-c049b76a60c6
	sigs.k8s.io/json v0.0.0-20220713155537-f223a00ba0e2
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
	k8s.io/kube-proxy => k8s.io/kube-proxy v0.25.16
	k8s.io/kube-scheduler => k8s.io/kube-scheduler v0.25.16
	k8s.io/kubectl => k8s.io/kubectl v0.25.16
	k8s.io/kubernetes => k8s.io/kubernetes v1.25.16
	sigs.k8s.io/controller-runtime => sigs.k8s.io/controller-runtime v0.11.2
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
	go.mongodb.org/mongo-driver => go.mongodb.org/mongo-driver v1.9.4
	google.golang.org/grpc => google.golang.org/grpc v1.56.3
)

// broken deps on 2.8.3
replace github.com/docker/distribution v2.8.3+incompatible => github.com/docker/distribution v2.8.2+incompatible
