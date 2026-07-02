# ACR Authentication Local Testing Guide

This guide explains what aspects of the ACR authentication implementation can be tested locally and what requires Azure DevOps pipeline execution.

## What CAN Be Tested Locally

### 1. YAML Syntax Validation ✅

**Tool**: `scripts/validate-pipelines.sh`

**What it tests**:
- YAML syntax errors
- Basic structure validation
- Template references exist

**How to run**:
```bash
./scripts/validate-pipelines.sh
```

**What it doesn't test**:
- Azure DevOps-specific features (service connections, variables)
- Pipeline execution logic
- Actual authentication

---

### 2. Azure CLI ACR Authentication ✅

**Tool**: `scripts/test-acr-auth.sh`

**Prerequisites**:
- Azure CLI installed (`az`)
- Logged in to Azure: `az login`
- Appropriate permissions on ACR registries

**What it tests**:
- Azure CLI can access ACR registries
- `az acr login` works for accessible registries
- Your Azure account has proper permissions

**How to run**:
```bash
# Login to Azure first
az login

# Run the test script
./scripts/test-acr-auth.sh
```

**Expected results**:
- ✅ If you're in the E2E subscription: Should successfully authenticate to `arosvcdev`
- ⚠️ If you're in a different subscription: Will show access warnings (expected)
- ⚠️ For `arointsvc` in MSIT tenant: May fail cross-tenant (expected locally)

**What it doesn't test**:
- Service connection authentication (ADO-specific)
- Cross-tenant authentication via service principals
- Actual image push/pull operations

---

### 3. Verify Pipeline Template References ✅

**Manual check**:
```bash
# Verify all template files exist
find .pipelines/templates -name "*.yml"

# Check that ci.yml references only existing templates
grep "template:" .pipelines/ci.yml | while read -r line; do
    template=$(echo "$line" | sed 's/.*template: //' | tr -d ' ')
    if [ -f ".pipelines/$template" ]; then
        echo "✅ $template exists"
    else
        echo "❌ $template MISSING"
    fi
done
```

---

### 4. Check for Hardcoded Credentials ✅

**Security audit**:
```bash
# Search for potential credential leaks in pipeline files
echo "Checking for hardcoded credentials..."

# Check for base64 encoded strings (potential credentials)
grep -r "base64" .pipelines/ --include="*.yml" | grep -v "acrCredentialsJSON"

# Check for password/secret keywords
grep -ri "password\|secret\|credential" .pipelines/ --include="*.yml" | \
    grep -v "# " | \
    grep -v "acrCredentialsJSON" | \
    grep -v "displayName"

# Check for potential ACR tokens
grep -r "ACR.*TOKEN\|ACR.*PASSWORD" .pipelines/ --include="*.yml"

echo "✅ Audit complete"
```

---

### 5. Verify Environment Variables Usage ✅

**Check variable usage**:
```bash
# List all variables defined in ci.yml
echo "Variables defined in ci.yml:"
grep -A 1 "^variables:" .pipelines/ci.yml
grep "  - name:" .pipelines/ci.yml

# Verify no credential variables
echo ""
echo "Checking for credential-related variables..."
grep -E "(USERNAME|PASSWORD|TOKEN|SECRET)" .pipelines/ci.yml || echo "✅ No credential variables found"
```

---

### 6. Documentation Review ✅

**Manual checklist**:
- [ ] Read `docs/acr-authentication.md`
- [ ] Verify all referenced service connections are documented
- [ ] Verify all ACR registries are documented
- [ ] Check that deprecated templates are marked as such
- [ ] Ensure troubleshooting section covers common scenarios

---

## What CANNOT Be Tested Locally (Requires ADO)

### 1. Service Connection Authentication ❌

**Why**: Service connections are configured in Azure DevOps and use managed identities or service principals that aren't accessible locally.

**Where to test**: Azure DevOps pipeline run

**What to check**:
- Pipeline logs show successful login via service connections
- No "authentication failed" errors
- Images are pushed to ACR successfully

---

### 2. Cross-Tenant Authentication ❌

**Why**: Requires Azure DevOps service connection configured for cross-tenant access.

**Where to test**: Azure DevOps pipeline run

**What to check**:
- `ado-pipeline-dev-image-push` service connection works
- Can push to `arosvcdev.azurecr.io` from pipeline
- No "tenant mismatch" errors

---

### 3. Dynamic TAG Generation ❌

**Why**: Uses Azure DevOps built-in variables like `$(System.PullRequest.PullRequestId)`.

**Where to test**: Actual PR pipeline run

**What to check**:
- TAG is set correctly: `pr-{id}-{commit}` for PRs, `master-{commit}` for master
- Images are tagged with the correct TAG
- E2E tests reference the correct TAG

**Workaround for local testing**:
```bash
# Simulate TAG generation
export TAG="local-test-$(git rev-parse --short HEAD)"
echo "TAG would be: $TAG"

# This only simulates the logic, doesn't test the actual pipeline
```

---

### 4. End-to-End Image Push/Pull ❌

**Why**: Requires:
- Docker daemon
- ACR authentication
- Build environment
- Network access to ACR

**Where to test**: Azure DevOps pipeline run

**What to check**:
- All three jobs in Containerized_CI stage succeed
- Images appear in `arosvcdev.azurecr.io` with correct tags
- E2E stage can pull the images

---

### 5. Service Principal Authentication ❌

**Why**: `$(aro-v4-e2e-devops-spn)` is a pipeline variable containing service principal credentials.

**Where to test**: Azure DevOps pipeline run (E2E stage)

**What to check**:
- `template-az-cli-login.yml` succeeds
- `az acr login --name arosvcdev` succeeds
- No authentication errors in E2E test logs

---

## Recommended Testing Strategy

### Phase 1: Local Pre-checks (Before PR)
```bash
# 1. Validate YAML
./scripts/validate-pipelines.sh

# 2. Check for hardcoded credentials
grep -r "password\|secret" .pipelines/ --include="*.yml" | grep -v "# \|displayName"

# 3. Verify template references
find .pipelines/templates -name "*.yml" -exec echo "✅ {}" \;

# 4. Review documentation
echo "Read docs/acr-authentication.md and verify accuracy"
```

### Phase 2: Azure CLI Test (Optional)
```bash
# Only if you have access to the Azure subscription
az login
./scripts/test-acr-auth.sh
```

### Phase 3: Pipeline Dry-Run (Recommended)

**Create a test PR with minimal changes**:
```bash
# Make a trivial change
echo "# Testing ARO-10651" >> docs/ARO-10651-local-testing.md

# Commit and push
git checkout -b test-aro-10651-acr-auth
git add .
git commit -m "test: verify ACR authentication in CI"
git push -u origin test-aro-10651-acr-auth

# Create PR in Azure DevOps
# Monitor pipeline execution
```

**What to monitor in pipeline logs**:
1. **Set_Tag_Stage**: Verify TAG is set correctly
2. **Containerized_CI jobs**:
   - Docker@2 login to arointsvc succeeds
   - template-acr-login.yml succeeds for arosvcdev
   - Image builds succeed
   - template-acr-push.yml succeeds for arosvcdev
3. **E2E jobs**:
   - template-az-cli-login.yml succeeds
   - `az acr login --name arosvcdev` succeeds
   - Docker compose pull succeeds
   - E2E tests run (may fail for other reasons, focus on image availability)

### Phase 4: Azure DevOps Verification (Required for Acceptance)

**Use**: `docs/ado-acr-credentials-verification.md`

**Requires**: Azure DevOps admin access

**Steps**:
1. Audit variable groups
2. Verify service connections
3. Check for stored credentials
4. Validate compliance with ARO-10651 acceptance criteria

---

## Common Issues and Local Debugging

### Issue: "template not found"

**Local check**:
```bash
# Find the template reference
grep -n "template:" .pipelines/ci.yml

# Verify file exists
ls -la .pipelines/templates/template-acr-login.yml
```

### Issue: "Invalid YAML syntax"

**Local check**:
```bash
# Use Python to validate YAML
python3 -c "import yaml; yaml.safe_load(open('.pipelines/ci.yml'))"

# Or use yamllint (if installed)
yamllint .pipelines/ci.yml
```

### Issue: "Variable not defined"

**Local check**:
```bash
# Find variable usage
grep "\$(.*)" .pipelines/ci.yml

# Check if variable is defined in variables section
grep -A 20 "^variables:" .pipelines/ci.yml
```

---

## What Good Looks Like

### ✅ Successful Local Tests
```
✅ All YAML files are valid
✅ All template references exist
✅ No hardcoded credentials found
✅ All documented service connections match pipeline usage
✅ Azure CLI can authenticate to accessible ACRs
```

### ✅ Successful Pipeline Run
```
✅ TAG set correctly: pr-1234-abc123def
✅ Docker login to arointsvc: succeeded
✅ ACR login to arosvcdev: succeeded  
✅ Image build: succeeded
✅ Image push to arosvcdev.azurecr.io/aro:pr-1234-abc123def: succeeded
✅ E2E can pull image: succeeded
✅ E2E tests complete: succeeded
```

### ✅ Successful ADO Verification
```
✅ No ACR credentials in variable groups
✅ Service connections use Service Principal auth
✅ No username/password service connections for ACR
✅ Pipeline runs show no authentication warnings
```

---

## Limitations

**What these tests DON'T verify**:
- Service connection configuration in Azure DevOps
- Actual cross-tenant authentication flow
- ACR repository permissions
- Image registry state
- Azure DevOps variable group contents

**Bottom line**: Local tests catch syntax errors and obvious misconfigurations, but the real validation happens in Azure DevOps pipeline execution.

---

## Next Steps

1. **Run local tests**: `./scripts/validate-pipelines.sh`
2. **Create test PR**: Push changes and monitor pipeline
3. **Verify ADO setup**: Use `docs/ado-acr-credentials-verification.md`
4. **Update Jira**: Mark ARO-10651 as complete once all verifications pass

## Questions?

- **Pipeline syntax**: See Azure DevOps YAML schema documentation
- **ACR authentication**: See `docs/acr-authentication.md`
- **ADO verification**: See `docs/ado-acr-credentials-verification.md`
- **SME**: Loki team (Azure DevOps pipeline owners)
