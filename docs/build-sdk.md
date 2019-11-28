# Dev golang SDK build

This document explains how to generate SDK version

## Dev SDK build

1. Generate new api swagger spec
    ```
    export APIVERSION=v20191231preview
    go run ./hack/swagger/swagger.go -i=$APIVERSION -o pkg/api/${APIVERSION}/swagger.json
    or
    make generate
    ```

1. Create required folder sturcture with example in `rest-api-spec`

    ```
    mkdir -p rest-api-spec/redhatopenshift/resource-manager/Microsoft.RedHatOpenShift/preview/2019-12-31-preview
    # copy generated swagger spec by updating generate.go in each api subfolder
    ```

1. Update all `readme` files files as they are used to generate SDK

1. Generate dev SDK

    ```
   make generate-sdk
    ```

## Upstream GoLang build

1. Clone `azure-rest-api-spec` and azure-sdk-for-go git repositories

   ```
   git clone https://github.com/Azure/azure-rest-api-specs
   git clone https://github.com/Azure/azure-sdk-for-go
   ```

1. Run code generation

    ```
    podman run --privileged -it -v $GOPATH:/go --entrypoint autorest \
    azuresdk/autorest /go/src/github.com/Azure/azure-rest-api-specs-pr/specification/redhatopenshift/resource-manager/readme.md \
    --go --go-sdks-folder=/go/src/github.com/Azure/azure-sdk-for-go/ --multiapi \
    --use=@microsoft.azure/autorest.go@~2.1.137 --use-onever --verbose
    ```

1. Go SDK will be generate inside `azure-sdk-for-go` project

## Useful links

* https://github.com/Azure/adx-documentation-pr/wiki/SDK-generation

* https://github.com/Azure/adx-documentation-pr
