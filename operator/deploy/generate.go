package deploy

//go:generate go get github.com/go-bindata/go-bindata/go-bindata
//go:generate go-bindata -nometadata -pkg $GOPACKAGE resources.yaml
//go:generate gofmt -s -l -w bindata.go
