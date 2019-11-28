//go:generate go run ../../../hack/swagger -o swagger.json -i $GOPACKAGE
//go:generate cp swagger.json ../../../rest-api-spec/redhatopenshift/resource-manager/Microsoft.RedHatOpenShift/preview/2019-12-31-preview/redhatopenshift.json

package v20191231preview
