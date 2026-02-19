# Arbitrary OpenShift Versions Feature

## Overview

This feature allows ARO clusters to be created with arbitrary OpenShift version strings instead of being limited to pre-defined versions stored in CosmosDB. This capability can be enabled through either:

1. **AFEC Feature Flag**: Protected by subscription-level feature registration for production/testing environments
2. **Development Environment**: Automatically enabled when running in local development mode

This dual-enablement approach supports both secure production testing and convenient development workflows.

## Implementation Details

### 1. AFEC Feature Flag

A new feature flag has been added to control access to this functionality:

```go
// pkg/api/featureflags.go
FeatureFlagArbitraryVersions = "Microsoft.RedHatOpenShift/ArbitraryVersions"
```

### 2. Enhanced Version Validation Logic

The `validateInstallVersion()` function in `pkg/frontend/validate.go` has been enhanced to:

- Accept a subscription document parameter to check for feature flags
- Check for both the `FeatureFlagArbitraryVersions` AFEC flag and development environment
- Apply different validation logic based on enablement conditions:

```go
allowArbitraryVersions := f.env.IsLocalDevelopmentMode() || 
    (subscription != nil && feature.IsRegisteredForFeature(subscription.Subscription.Properties, api.FeatureFlagArbitraryVersions))
```

**When arbitrary versions are enabled (either via AFEC flag OR development mode):**
- Only validates semantic version format using `semver.NewVersion`
- Bypasses the requirement for versions to be in the CosmosDB enabled list
- Allows custom version strings like `4.15.0-custom.build.123`

**When arbitrary versions are disabled:**
- Maintains existing behavior (version must be in enabled list AND valid semver)
- Preserves backward compatibility

### 3. Enhanced Image Resolution with ACR Fallback

The version resolution system in `pkg/cluster/version.go` has been enhanced to support arbitrary versions:

**Resolution Order:**
1. **CosmosDB First**: Check if the version exists in the OpenShiftVersions collection
2. **ACR Fallback**: If not found and arbitrary versions are enabled (via AFEC flag OR development mode), generate image specs using ACR patterns
3. **Error**: If neither found and arbitrary versions disabled, return error

**ACR Image Generation:**
- **Installer Image**: `{ACRDomain}/aro-installer:{major.minor}` (e.g., `arosvc.azurecr.io/aro-installer:4.15`)
- **OpenShift Image (Hive)**: `{ACRDomain}/ocp-release:{full-version}` (e.g., `arosvc.azurecr.io/ocp-release:4.15.0-custom.build.123`)
- **OpenShift Image (Traditional)**: `quay.io/openshift-release-dev/ocp-release:{full-version}`

**Installer Pull Spec Override**: Still honors `env.LiveConfig().DefaultInstallerPullSpecOverride()` when set

### 4. Updated Function Signatures

The following functions were updated to support subscription-aware validation:

- `validateInstallVersion(ctx, oc, subscription)` - Now accepts subscription document
- `openShiftClusterDocumentVersioner.GetWithSubscription()` - New method with subscription support
- PUT/PATCH handler passes subscription to validation
- Preflight validation retrieves and passes subscription document

### 5. Test Coverage

Comprehensive test cases have been added covering:

**Frontend Validation Tests:**
- ✅ Arbitrary valid semver versions with feature flag enabled
- ✅ Arbitrary valid semver versions in development mode
- ✅ Invalid versions with feature flag enabled (proper error handling)
- ✅ Invalid versions in development mode (proper error handling)
- ✅ Arbitrary versions without feature flag (blocked as expected)
- ✅ Both AFEC flag and development mode enabled (should work)
- ✅ Development mode overrides normal validation for arbitrary versions
- ✅ Existing functionality preserved for standard versions

**Image Resolution Tests:**
- ✅ ACR fallback for traditional installer (quay.io OpenShift images)
- ✅ ACR fallback for Hive installer (ACR OpenShift images)
- ✅ ACR fallback in development mode (both traditional and hive)
- ✅ CosmosDB versions take precedence over ACR fallback
- ✅ Invalid semantic versions with proper error messages
- ✅ Invalid semantic versions in development mode
- ✅ Prerelease and development version handling
- ✅ Major.minor version extraction for installer images
- ✅ Both AFEC flag and development mode scenarios
- ✅ Development mode override behavior for version resolution

## Usage

### Enabling the Feature

There are two ways to enable arbitrary version support:

#### Option 1: AFEC Feature Flag (Production/Testing)

To enable arbitrary versions for a subscription:

```bash
# Register the feature flag for a subscription
az feature register --namespace Microsoft.RedHatOpenShift --name ArbitraryVersions

# Verify registration (may take a few minutes)
az feature show --namespace Microsoft.RedHatOpenShift --name ArbitraryVersions
```

#### Option 2: Development Environment (Local Development)

For local development, the feature is automatically enabled when `RP_MODE=development` is set:

```bash
# Set development mode environment variable
export RP_MODE=development

# Now arbitrary versions are automatically enabled without AFEC registration
```

**Note**: Development mode bypasses AFEC flag requirements and is intended for local development only.

### Example Version Strings

Once enabled, cluster creation requests can specify custom versions such as:

- `4.15.0-custom.build.123` - Custom builds
- `4.14.0-0.nightly-2024-01-01-000000` - Nightly builds
- `4.13.25+dev.branch.feature` - Development branches
- `4.16.0-rc.1` - Release candidates

### API Usage

No changes to the API structure are required. Simply specify the desired version in the cluster creation request:

```json
{
  "properties": {
    "clusterProfile": {
      "version": "4.15.0-custom.build.123"
    }
  }
}
```

## Security Considerations

- **Dual-Layer Protection**: Feature is gated behind either:
  - **AFEC Protection**: Subscription-level feature registration for production environments
  - **Development Mode**: Local development environment detection (`RP_MODE=development`)
- **Validation**: Still enforces semantic versioning format to prevent invalid strings
- **Environment Isolation**: Development mode only works in local development environments
- **Audit Trail**: Feature flag registration is tracked in Azure subscription logs

## Image Resolution Behavior

### ACR Fallback Logic

When using arbitrary versions, the system follows this resolution order:

1. **Check CosmosDB**: First attempts to find the exact version in the OpenShiftVersions collection
2. **Generate ACR Specs**: If not found and feature flag enabled, generates image specifications:
   - Installer image uses `{major.minor}` tagging (e.g., `4.15` for version `4.15.0-custom.build.123`)
   - OpenShift image includes full version string
3. **Installation Attempt**: The installation will proceed with generated image specifications
4. **Runtime Validation**: If the images don't exist in ACR, the installation will fail during image pull

### Image Availability

The ACR fallback assumes images follow these naming conventions:
- **Production ACR**: `arosvc.azurecr.io/aro-installer:4.15`
- **Integration ACR**: `arointsvc.azurecr.io/aro-installer:4.15`

**Important**: The feature enables specifying arbitrary versions, but actual installation success depends on the availability of corresponding images in the configured ACR registry.

## Error Handling

### Invalid Semantic Version
```
400: InvalidParameter: properties.clusterProfile.version: 
The requested OpenShift version 'not-a-valid-version' is not a valid semantic version.
```

### Feature Not Enabled
```
400: InvalidParameter: properties.clusterProfile.version: 
The requested OpenShift version '4.15.0-custom.build.123' is invalid.
```

### Installation-Time Errors
If ACR images don't exist, installation will fail with image pull errors during the cluster creation process.

## Files Modified

- `pkg/api/featureflags.go` - Added new feature flag constant
- `pkg/frontend/validate.go` - Enhanced validation logic with AFEC flag support
- `pkg/frontend/openshiftcluster_putorpatch.go` - Updated function calls
- `pkg/frontend/openshiftcluster_preflightvalidation.go` - Added subscription retrieval
- `pkg/frontend/validate_test.go` - Added comprehensive test coverage
- `pkg/cluster/version.go` - Enhanced image resolution with ACR fallback logic
- `pkg/cluster/install_version.go` - Updated to use subscription-aware version resolver
- `pkg/cluster/install_version_test.go` - Added ACR fallback test coverage

## Compatibility

This feature maintains full backward compatibility. Existing clusters and installations are unaffected when the feature flag is not enabled. The implementation follows the same pattern as the existing MTU3900 feature flag.

## Testing

The feature includes comprehensive unit tests that validate:

1. **AFEC Flag Scenarios**: Feature flag enabled/disabled with valid and invalid versions
2. **Development Mode Scenarios**: Local development environment detection and behavior
3. **Combined Scenarios**: Both AFEC flag and development mode enabled
4. **Backward Compatibility**: Existing behavior preserved when feature is disabled
5. **Default Behavior**: Default version assignment when no version specified
6. **Error Handling**: Accurate error messages for all failure scenarios
7. **ACR Fallback**: Image resolution patterns for arbitrary versions
8. **Precedence Rules**: CosmosDB versions take precedence over ACR fallback

## Future Considerations

- Consider adding validation for minimum supported version patterns
- Potential integration with custom installer image specifications
- Monitoring and alerting for non-standard version usage