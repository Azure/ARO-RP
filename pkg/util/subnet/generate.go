package subnet

//go:generate go run ../../../vendor/github.com/golang/mock/mockgen -destination=../mocks/mock_$GOPACKAGE/$GOPACKAGE.go github.com/jim-minter/rp/pkg/util/$GOPACKAGE Manager
//go:generate go run ../../../vendor/golang.org/x/tools/cmd/goimports -local=github.com/jim-minter/rp -e -w ../mocks/mock_$GOPACKAGE/$GOPACKAGE.go
