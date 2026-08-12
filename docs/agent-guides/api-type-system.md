# API Type System Guide

Read this when modifying `pkg/api/`, `pkg/api/v*`, `pkg/api/admin/`, or `pkg/api/validate/`.

## Internal vs External Boundary

The RP maintains a strict type boundary:

- **Internal types** (`pkg/api/openshiftcluster.go`): Source of truth for cluster state, stored in CosmosDB
- **External types** (`pkg/api/v*/openshiftcluster.go`): ARM-facing, must match swagger spec exactly
- **Admin types** (`pkg/api/admin/openshiftcluster.go`): Internal admin API (mutual TLS), NOT customer-facing

External types are local to each API version package. They must use local type definitions (e.g., `type VMSize string`), NOT imported types from internal packages. The swagger spec defines `VMSize` as `"type": "string"`, so external types must match that.

## Three VMSize Types

These are NOT interchangeable — mixing them causes compile errors:

| Type | Location | Used by |
|------|----------|---------|
| `api.VMSize` | `pkg/api/openshiftcluster.go` | CosmosDB documents, internal logic |
| `vms.VMSize` | `pkg/api/util/vms/types.go` | Admin API, `validate` package, centralized VM metadata |
| Local `VMSize` | `pkg/api/v*/openshiftcluster.go` | External ARM responses per API version |

**Conversion pattern** (in `_convert.go` files):
```go
// internal → external (ToExternal)
MasterProfile: MasterProfile{
    VMSize: VMSize(oc.Properties.MasterProfile.VMSize),  // api.VMSize → local VMSize
}

// external → internal (ToInternal)
oc.Properties.MasterProfile.VMSize = api.VMSize(ext.Properties.MasterProfile.VMSize)
```

**Admin API is different**: `pkg/api/admin/` uses `vms.VMSize` directly. Do NOT apply the "use local VMSize" pattern to admin.

## API Version Registration

Each version registers into `api.APIs` (global map in `pkg/api/register.go`) via `init()`:

```go
// pkg/api/v20250725/register.go
func init() {
    api.APIs[APIVersion] = &api.Version{
        OpenShiftClusterConverter:    openShiftClusterConverter{},
        OpenShiftClusterStaticValidator: openShiftClusterStaticValidator{},
        // ... converters and validators
    }
}
```

The `api.Version` struct defines: `OpenShiftClusterConverter`, `OpenShiftClusterStaticValidator`, `OpenShiftClusterCredentialsConverter`, `OpenShiftClusterAdminKubeconfigConverter`, `OpenShiftVersionConverter`, `PlatformWorkloadIdentityRoleSetConverter`, and their static validators.

Frontend handlers resolve the version-specific converter at runtime:
```go
converter := f.apis[r.URL.Query().Get(api.APIVersionKey)].OpenShiftClusterConverter
```

## Files Per API Version

For `v20240812preview` and earlier versions, `pkg/api/v*` directory contains:

| File pattern | Purpose |
|-------------|---------|
| `openshiftcluster.go` | External struct definitions |
| `openshiftcluster_convert.go` | `ToExternal()` / `ToInternal()` with type casts |
| `openshiftcluster_validatestatic.go` | Request validation |
| `openshiftcluster_example.go` | Swagger example payloads |
| `register.go` | API version registration into `api.APIs` |
| `generate.go` | `//go:generate` directives for swagger |

Plus parallel files for: `openshiftclustercredentials`, `openshiftclusteradminkubeconfig`, `openshiftversion`, `platformworkloadidentityroleset`.

For `v20250725` and later versions, there is one additional file:

| File pattern | Purpose |
|-------------|---------|
| `openshiftcluster_updatepolicy.go` | Stores an `immutable.Policy` used in static validation |

## Swagger Generation

### For v20250725 and later API versions

The source of truth is now the Typespec contained in `api/redhatopenshift/resource-manager/Microsoft.RedHatOpenShift/OpenShiftClusters/*.tsp`. The Go API models in `pkg/api/v*` are generated from the TypeSpec and stored in `pkg/api/v*/generated`.

```
api/redhatopenshift/resource-manager/Microsoft.RedHatOpenShift/OpenShiftClusters/*.tsp -> hack/api/generate-from-typespec.sh swagger
```

- Generator: `hack/api/generate-from-typespec.sh swagger` wraps invocations of the `npm` scripts from `api/package.json`
- Target: `make generate-swagger-typespec`

### For v20240812preview and preceding API versions

The Swagger API spec is generated from the handwritten Go model types.

```
pkg/api/v*/openshiftcluster.go  →  hack/swagger-legacy/swagger.go  →  swagger/{stable|preview}/{version}/redhatopenshift.json
```

- Generator: `hack/swagger-legacy/swagger.go` wraps `pkg/swagger/swagger.go`
- Target: `make generate-swagger-legacy`

## Client Generation

```
api/redhatopenshift/resource-manager/Microsoft.RedHatOpenShift/OpenShiftClusters/client.tsp  →  TypeSpec (invoked via npm scripts in api/package.json)  →  pkg/client/sdk/resourcemanager/redhatopenshift/armredhatopenshift

- Make target: `make client-generate`
- Generates both Go SDK and Python SDK clients
- Generates based on the latest API version in the TypeSpec (currently `2025-07-25`)

## Adding New VM Sizes Checklist

1. `pkg/api/openshiftcluster.go` — Add internal `VMSize` constant
2. `pkg/api/admin/openshiftcluster.go` — Add admin API constant
3. `pkg/api/validate/vm.go` — Add to `supportedMasterVmSizes` and/or `supportedWorkerVmSizes`
4. `pkg/validate/dynamic/quota.go` — Define required resources (vCPUs, etc.)
5. If version-gated: add to `masterVmSizesWithMinimumVersion` / `workerVmSizesWithMinimumVersion` in `vm.go`

See also: `docs/adding-new-instance-types.md`

## Hidden Dependencies

- `GET /admin/supportedvmsizes` (`pkg/frontend/admin_supportvmsizes_list.go`) calls `validate.SupportedVMSizesByRole()` and JSON-marshals `map[api.VMSize]api.VMSizeStruct`. The `VMSizeStruct` JSON tags in `pkg/api/openshiftcluster.go` directly affect the API response format.
- `validate.VMSizeIsValidForVersion` signature has evolved over time (e.g., `requireD2sWorkers` parameter). Always check the current signature.
