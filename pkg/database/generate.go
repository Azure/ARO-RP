package database

//go:generate go run ../../vendor/github.com/golang/mock/mockgen -destination=../util/mocks/mock_$GOPACKAGE/$GOPACKAGE.go github.com/jim-minter/rp/pkg/$GOPACKAGE OpenShiftClusters,Subscriptions
//go:generate go run ../../vendor/golang.org/x/tools/cmd/goimports -local=github.com/jim-minter/rp -e -w ../util/mocks/mock_$GOPACKAGE/$GOPACKAGE.go
