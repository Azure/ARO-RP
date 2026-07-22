# ARO-RP API versions

See [agent-guides/api-type-system.md](agent-guides/api-type-system.md)'s "Swagger Generation" section for details about the source of truth for ARO-RP API definitions and the process of getting from TypeSpec to Swagger.

## Generate Swagger for API versions >= v20250725 (`api` TypeSpec source of truth)

`make generate-swagger-typespec`

## Generate API examples from Swagger

`make generate-api-examples`

We've decoupled example generation from Swagger generation because the `oav` utility used to generate the examples updates some GUIDs, etc. anytime you generate the examples even if the TypeSpec and Swagger haven't changed. The pro of this approach is that when you only want to regenerate Swagger, you don't end up with extraneous git diffs that don't reflect meaningful changes that you'll want to commit. The con of this approach (it's really a result of using `oav` to generate the examples) is that CI doesn't check whether examples need updates; if you're updating the TypeSpec/Swagger, you should make sure to regenerate the examples.

## Generate Swagger for API versions <= v20240812preview (`pkg/api` Go struct source of truth)

`make generate-swagger-legacy`

We've kept around the `make` target for the older API versions from back when the Go structs in `pkg/api` were the source of truth for the API specifications. A bespoke solution in `pkg/swagger` converts the Go structs to Swagger. It's unlikely we will ever need it, but it's still here just in case.

## Notes about some design choices made during the migration to TypeSpec

- You will note that we store the Swagger API specs from before and after the migration in two different places: `api/redhatopenshift/resource-manager/Microsoft.RedHatOpenShift/OpenShiftClusters` (new) and `swagger/redhatopenshift/resource-manager/Microsoft.RedHatOpenShift/openshiftclusters` (old). While you might expect the specs for all API versions to be stored in a single place, the post-migration API specifications are stored under the `api` directory in the same directory structure used upstream (aka in https://github.com/Azure/azure-rest-api-specs) to make future upstream contributions easier. The specs for the older API versions remain where they are and as they are because it's unlikely we will need to do much with them for the remainder of ARO-RP's lifecycle.
- We used https://github.com/Azure/ARO-HCP/tree/main/api as our starting point when migrating the ARO-RP API spec to TypeSpec.
- The Azure Rest API specs "common types" are duplicated between `api/common-types` and `swagger/common-types`. It would make more sense to have a single source of truth, but `make client`, which builds Go and Python SDK clients used for dev and testing, depends on the common types and is not working as of the time of writing of this doc. With this in mind, everything we need in the post-migration world is kept in `api`, and `swagger` is left untouched aside from the removal of the v20250725 version. `make client` will be fixed soon.
