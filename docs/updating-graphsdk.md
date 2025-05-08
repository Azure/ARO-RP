# Updating Microsoft Graph SDK

`pkg/util/graph/graphsdk` is a specially generated version of the Microsoft Graph API SDK. Since it uses the stable v1.0 of the MSGraph API, it is unlikely that it will need to be updated, although updates to the Kiota generation/libraries or new endpoints being required may require this.

If an update is required, perform the following:

1. Install the latest version of [Kiota](https://github.com/microsoft/kiota) (most likely by downloading it and putting it in your `$PATH`).
1. If new endpoints/schemas are required: Add the relevant endpoints and in `hack/graphsdk/openapi.yaml` from the [msgraph-metadata](https://github.com/microsoftgraph/msgraph-metadata/blob/master/openapi/v1.0/openapi.yaml) version.
1. Run `make generate-kiota` and commit the result.
1. Run `kiota info -d ./hack/graphsdk/openapi.yaml -l Go` to get the version of the Kiota libraries that are needed and update them. Then, commit the result.
