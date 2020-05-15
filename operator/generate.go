package operator

//go:generate go get github.com/go-bindata/go-bindata/go-bindata
//go:generate go-bindata -nometadata -pkg $GOPACKAGE -prefix config config/output/...
//go:generate gofmt -s -l -w bindata.go
