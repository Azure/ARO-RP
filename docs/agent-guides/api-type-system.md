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

Each `pkg/api/v*` directory contains:

| File pattern | Purpose |
|-------------|---------|
| `openshiftcluster.go` | External struct definitions |
| `openshiftcluster_convert.go` | `ToExternal()` / `ToInternal()` with type casts |
| `openshiftcluster_validatestatic.go` | Request validation |
| `openshiftcluster_example.go` | Swagger example payloads |
| `register.go` | API version registration into `api.APIs` |
| `generate.go` | `//go:generate` directives for swagger |

Plus parallel files for: `openshiftclustercredentials`, `openshiftclusteradminkubeconfig`, `openshiftversion`, `platformworkloadidentityroleset`.

## Swagger Generation

```
pkg/api/v*/openshiftcluster.go  →  hack/swagger/swagger.go  →  swagger/{stable|preview}/{version}/redhatopenshift.json
```

- Generator: `hack/swagger/swagger.go` wraps `pkg/swagger/swagger.go`
- Target: `make generate-swagger`
- Triggered by `//go:generate` in each version's `generate.go`

## Client Generation (destructive)

```
swagger/{version}/redhatopenshift.json  →  AutoRest (Docker)  →  pkg/client/services/redhatopenshift/mgmt/{version}/
```

- Script: `hack/apiclients/build-dev-api-clients.sh`
- Requires: Docker image `arointsvc.azurecr.io/autorest:3.7.2` (build with `make image-autorest`)
- **WARNING**: Deletes `pkg/client/services/redhatopenshift/mgmt/` before regenerating. If it fails, restore: `git checkout -- pkg/client/ python/client/`
- Generates both Go SDK and Python SDK
- Only generates for versions in the `client-generate` Makefile target (currently `2024-08-12-preview` and `2025-07-25`)

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
