module github.com/Azure/ARO-RP/pkg/api

go 1.22.9

require (
	github.com/Azure/azure-sdk-for-go/sdk/azcore v1.17.0
	github.com/Azure/go-autorest/autorest v0.11.29
	github.com/Azure/go-autorest/autorest/date v0.3.0
	github.com/Azure/go-autorest/autorest/to v0.4.0
	github.com/coreos/go-semver v0.3.1
	github.com/gofrs/uuid v4.4.0+incompatible
	github.com/ugorji/go/codec v1.2.12
	go.uber.org/mock v0.4.0
)

require (
	github.com/Azure/azure-sdk-for-go/sdk/internal v1.10.0 // indirect
	github.com/Azure/go-autorest v14.2.0+incompatible // indirect
	github.com/Azure/go-autorest/autorest/adal v0.9.23 // indirect
	github.com/Azure/go-autorest/logger v0.2.1 // indirect
	github.com/Azure/go-autorest/tracing v0.6.0 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/golang-jwt/jwt/v4 v4.5.2 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	golang.org/x/crypto v0.33.0 // indirect
	golang.org/x/net v0.35.0 // indirect
	golang.org/x/text v0.22.0 // indirect
)

replace (
	github.com/googleapis/gnostic => github.com/google/gnostic v0.5.5
	github.com/openshift/hive/apis => github.com/openshift/hive/apis v0.0.0-20231116161336-9dd47f8bfa1f
)
