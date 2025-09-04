# Arbitrary OpenShift Versions Feature

## Overview

This feature allows ARO clusters to be created with arbitrary OpenShift version strings instead of being limited to pre-defined versions stored in CosmosDB. This capability is protected by an AFEC (Azure Feature Exposure Control) flag and is intended for testing and development scenarios where custom builds or pre-release versions need to be installed.

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
- Check for the `FeatureFlagArbitraryVersions` AFEC flag
- Apply different validation logic based on the flag state:

**When flag is enabled:**
- Only validates semantic version format using `semver.NewVersion`
- Bypasses the requirement for versions to be in the CosmosDB enabled list
- Allows custom version strings like `4.15.0-custom.build.123`

**When flag is disabled:**
- Maintains existing behavior (version must be in enabled list AND valid semver)
- Preserves backward compatibility

### 3. Updated Function Signatures

The following functions were updated to support subscription-aware validation:

- `validateInstallVersion(ctx, oc, subscription)` - Now accepts subscription document
- PUT/PATCH handler passes subscription to validation
- Preflight validation retrieves and passes subscription document

### 4. Test Coverage

Comprehensive test cases have been added covering:

- ✅ Arbitrary valid semver versions with feature flag enabled
- ✅ Invalid versions with feature flag enabled (proper error handling)
- ✅ Arbitrary versions without feature flag (blocked as expected)
- ✅ Existing functionality preserved for standard versions

## Usage

### Enabling the Feature

To enable arbitrary versions for a subscription:

```bash
# Register the feature flag for a subscription
az feature register --namespace Microsoft.RedHatOpenShift --name ArbitraryVersions

# Verify registration (may take a few minutes)
az feature show --namespace Microsoft.RedHatOpenShift --name ArbitraryVersions
```

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

- **AFEC Protection**: Feature is gated behind subscription-level feature registration
- **Validation**: Still enforces semantic versioning format to prevent invalid strings
- **Audit Trail**: Feature flag registration is tracked in Azure subscription logs

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

## Files Modified

- `pkg/api/featureflags.go` - Added new feature flag constant
- `pkg/frontend/validate.go` - Enhanced validation logic with AFEC flag support
- `pkg/frontend/openshiftcluster_putorpatch.go` - Updated function calls
- `pkg/frontend/openshiftcluster_preflightvalidation.go` - Added subscription retrieval
- `pkg/frontend/validate_test.go` - Added comprehensive test coverage

## Compatibility

This feature maintains full backward compatibility. Existing clusters and installations are unaffected when the feature flag is not enabled. The implementation follows the same pattern as the existing MTU3900 feature flag.

## Testing

The feature includes comprehensive unit tests that validate:

1. Feature flag enabled scenarios with valid and invalid versions
2. Feature flag disabled scenarios (existing behavior)
3. Default version assignment when no version specified
4. Error message accuracy and formatting

## Future Considerations

- Consider adding validation for minimum supported version patterns
- Potential integration with custom installer image specifications
- Monitoring and alerting for non-standard version usage