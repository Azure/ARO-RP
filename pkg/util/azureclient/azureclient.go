package azureclient

//go:generate go run ../../../vendor/github.com/golang/mock/mockgen -destination=../mocks/mock_$GOPACKAGE/azureclient.go github.com/jim-minter/rp/pkg/util/$GOPACKAGE Client
//go:generate gofmt -s -l -w ../../util/mocks/mock_$GOPACKAGE/azureclient.go
//go:generate go run ../../../vendor/golang.org/x/tools/cmd/goimports -local=github.com/jim-minter/rp -e -w ../mocks/mock_$GOPACKAGE/azureclient.go

import (
	"github.com/Azure/go-autorest/autorest"
)

// Client returns the Client
type Client interface {
	Client() autorest.Client
}
