# ARO-RP API versions

See [agent-guides/api-type-system.md](agent-guides/api-type-system.md)'s "Swagger Generation" section for details about the source of truth for ARO-RP API definitions and the process of getting from TypeSpec to Swagger.

## Generate Swagger for API versions >= v20250725 (`api` TypeSpec source of truth)

1. Run `make image-typespec`
2. Run `make generate-swagger-typespec` (or just `make generate`; the `generate-swagger-typespec` target is one of its dependencies)
3. (Optional; see explanation below) `git restore api/redhatopenshift/resource-manager/Microsoft.RedHatOpenShift/OpenShiftClusters/examples`
    - `make generate-swagger-typespec` includes the generation of some API example files, e.g. `api/redhatopenshift/resource-manager/Microsoft.RedHatOpenShift/OpenShiftClusters/examples`. Each time you run this `make` target, some of the randomly generated values in the examples will be regenerated, resulting in changes that `git` will pick up on. This is due to the behavior of the `oav` utility used to generate the examples. **Please avoid committing the example files unless you meant to change them**; in more general terms, please avoid using `git add *` and instead ensure you commit only the files you need to change.

## Generate Swagger for API versions <= v20240812preview (`pkg/api` Go struct source of truth)

We've kept around the `make` target for the older API versions from back when the Go structs in `pkg/api` were the source of truth for the API specifications. A bespoke solution in `pkg/swagger` converts the Go structs to Swagger. It's unlikely we will ever need it, but it's still here just in case.

There is only one step: `make generate-swagger-legacy`

## Notes about some design choices made during the migration to TypeSpec

- You will note that we store the Swagger API specs from before and after the migration in two different places: `api/redhatopenshift/resource-manager/Microsoft.RedHatOpenShift/OpenShiftClusters` (new) and `swagger/redhatopenshift/resource-manager/Microsoft.RedHatOpenShift/openshiftclusters` (old). While you might expect the specs for all API versions to be stored in a single place, the post-migration API specifications are stored under the `api` directory in the same directory structure used upstream (aka in https://github.com/Azure/azure-rest-api-specs) to make future upstream contributions easier. The specs for the older API versions remain where they are and as they are because it's unlikely we will need to do much with them for the remainder of ARO-RP's lifecycle.
- We used https://github.com/Azure/ARO-HCP/tree/main/api as our starting point when migrating the ARO-RP API spec to TypeSpec. However, we avoided using a VSCode dev container and instead used a TypeSpec-specific container image and individual `podman run` commands encapsulated within `make` targets to align with existing patterns in ARO-RP.
- The Azure Rest API specs "common types" are duplicated between `api/common-types` and `swagger/common-types`. It would make more sense to have a single source of truth, but `make client`, which builds Go and Python SDK clients used for dev and testing, depends on the common types and is not working as of the time of writing of this doc. With this in mind, everything we need in the post-migration world is in kept in `api`, and `swagger` is left mostly untouched. We can revisit the old and broken `make client` when and if we have to.