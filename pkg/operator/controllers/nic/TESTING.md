# NIC Controller - Testing Documentation

## Test Coverage Summary

**Overall Coverage**: 44.8% of statements
**Test File**: `nic_controller_test.go` (16KB, 530 lines)

### Coverage by Function

| Function | Coverage | Notes |
|----------|----------|-------|
| `extractNICNameFromMachine` | **100%** | ✅ Fully tested |
| `isNICInFailedState` | **100%** | ✅ Fully tested |
| `isNotFoundError` | **100%** | ✅ Fully tested |
| `Reconcile` | **72.2%** | ✅ Core paths tested |
| `reconcileNICForMachine` | **43.8%** | Partial (Azure calls mocked) |
| `reconcileNICsForMachineSet` | **41.2%** | Partial (Azure calls mocked) |
| `reconcileAllNICs` | 0% | Requires Azure SDK mocks |
| `reconcileNIC` | 0% | Requires Azure SDK mocks |
| `SetupWithManager` | 0% | Controller setup (integration test) |
| `NewReconciler` | 0% | Simple constructor |

---

## Test Cases

### 1. `TestExtractNICNameFromMachine` ✅
**Purpose**: Verify NIC name extraction from Machine resources

**Test Cases**:
- ✅ Master machine with valid provider spec
- ✅ Worker machine with valid provider spec
- ✅ Machine with nil provider spec (error case)
- ✅ Machine with invalid JSON (error case)
- ✅ Machine with empty name (error case)

**Coverage**: 100%

**Example**:
```go
machine := getValidMachine("cluster-abc-master-0", true)
nicName, err := extractNICNameFromMachine(machine)
// Expected: "cluster-abc-master-0-nic"
```

---

### 2. `TestIsNICInFailedState` ✅
**Purpose**: Verify detection of failed NIC provisioning states

**Test Cases**:
- ✅ NIC in Failed state → returns true
- ✅ NIC in Succeeded state → returns false
- ✅ NIC in Creating state → returns false
- ✅ NIC in Updating state → returns false
- ✅ NIC in Deleting state → returns false
- ✅ NIC with nil Properties → returns false
- ✅ NIC with nil ProvisioningState → returns false

**Coverage**: 100%

**Example**:
```go
nic := &armnetwork.Interface{
    Properties: &armnetwork.InterfacePropertiesFormat{
        ProvisioningState: pointerutils.ToPtr(armnetwork.ProvisioningStateFailed),
    },
}
result := isNICInFailedState(nic)
// Expected: true
```

---

### 3. `TestIsNotFoundError` ✅
**Purpose**: Verify 404 Not Found error detection

**Test Cases**:
- ✅ Nil error → returns false
- ✅ "ResourceNotFound" error → returns true
- ✅ Generic "NotFound" error → returns true
- ✅ "404" status code error → returns true
- ✅ Permission denied error (403) → returns false
- ✅ Network error → returns false
- ✅ Timeout error → returns false

**Coverage**: 100%

**Example**:
```go
err := errors.New("404 Not Found: ResourceNotFound")
result := isNotFoundError(err)
// Expected: true
```

---

### 4. `TestReconcilerControllerDisabled` ✅
**Purpose**: Verify controller skips reconciliation when disabled

**Test Steps**:
1. Create Cluster CR with `aro.nic.enabled: "false"`
2. Call Reconcile()
3. Verify no error and no requeue

**Coverage**: Tests early return path

---

### 5. `TestReconcilerControllerEnabled` ✅
**Purpose**: Verify controller attempts reconciliation when enabled

**Test Steps**:
1. Create Cluster CR with `aro.nic.enabled: "true"`
2. Call Reconcile()
3. Verify it gets past the "disabled" check

**Note**: Expected to fail Azure env parsing in test environment

---

### 6. `TestReconcileNICForMachineNotFound` ✅
**Purpose**: Verify graceful handling of non-existent machines

**Test Steps**:
1. Create empty cluster (no machines)
2. Attempt to reconcile non-existent machine
3. Verify error is handled gracefully

---

### 7. `TestReconcileNICsForMachineSet` ✅
**Purpose**: Verify finding machines owned by a MachineSet

**Test Cases**:
- ✅ MachineSet with multiple machines (3 machines)
- ✅ MachineSet with no machines (empty)

**Test Steps**:
1. Create MachineSet
2. Create Machines with OwnerReferences pointing to MachineSet
3. Verify correct machines are found
4. Verify NIC names can be extracted from each machine

---

### 8. `TestNICNamingConvention` ✅
**Purpose**: Verify NIC naming follows expected pattern

**Test Cases**:
- ✅ `cluster-abc-master-0` → `cluster-abc-master-0-nic`
- ✅ `cluster-abc-master-1` → `cluster-abc-master-1-nic`
- ✅ `cluster-abc-master-2` → `cluster-abc-master-2-nic`
- ✅ `cluster-abc-worker-xyz-1` → `cluster-abc-worker-xyz-1-nic`
- ✅ `cluster-abc-worker-xyz-2` → `cluster-abc-worker-xyz-2-nic`

---

### 9. `TestProvisioningStateDetection` ✅
**Purpose**: Verify detection of all Azure provisioning states

**Test Cases**:
- ✅ Failed → detected as failed
- ✅ Succeeded → detected as healthy
- ✅ Creating → detected as healthy
- ✅ Updating → detected as healthy
- ✅ Deleting → detected as healthy

---

## Running Tests

### Run All Tests
```bash
go test -v ./pkg/operator/controllers/nic/...
```

**Expected Output**:
```
=== RUN   TestExtractNICNameFromMachine
--- PASS: TestExtractNICNameFromMachine (0.00s)
=== RUN   TestIsNICInFailedState
--- PASS: TestIsNICInFailedState (0.00s)
=== RUN   TestIsNotFoundError
--- PASS: TestIsNotFoundError (0.00s)
=== RUN   TestReconcilerControllerDisabled
--- PASS: TestReconcilerControllerDisabled (0.00s)
=== RUN   TestReconcilerControllerEnabled
--- PASS: TestReconcilerControllerEnabled (0.00s)
=== RUN   TestReconcileNICForMachineNotFound
--- PASS: TestReconcileNICForMachineNotFound (0.00s)
=== RUN   TestReconcileNICsForMachineSet
--- PASS: TestReconcileNICsForMachineSet (0.00s)
=== RUN   TestNICNamingConvention
--- PASS: TestNICNamingConvention (0.00s)
=== RUN   TestProvisioningStateDetection
--- PASS: TestProvisioningStateDetection (0.00s)
PASS
ok  	github.com/Azure/ARO-RP/pkg/operator/controllers/nic	1.093s
```

---

### Run Specific Test
```bash
go test -v ./pkg/operator/controllers/nic/... -run TestExtractNICNameFromMachine
```

---

### Run with Coverage
```bash
go test ./pkg/operator/controllers/nic/... -cover
```

**Expected**:
```
ok  	github.com/Azure/ARO-RP/pkg/operator/controllers/nic	1.140s	coverage: 44.8% of statements
```

---

### Generate Coverage Report
```bash
# Generate coverage profile
go test ./pkg/operator/controllers/nic/... -coverprofile=coverage.out

# View coverage by function
go tool cover -func=coverage.out | grep nic_controller.go

# Generate HTML coverage report
go tool cover -html=coverage.out -o coverage.html
open coverage.html  # macOS
```

---

## Test Helpers

### `getValidClusterInstance(nicEnabled bool)`
Creates a test Cluster CR with NIC controller enabled/disabled

**Example**:
```go
cluster := getValidClusterInstance(true)  // Controller enabled
cluster := getValidClusterInstance(false) // Controller disabled
```

---

### `getValidMachine(name string, isMaster bool)`
Creates a test Machine resource

**Example**:
```go
masterMachine := getValidMachine("cluster-abc-master-0", true)
workerMachine := getValidMachine("cluster-abc-worker-1", false)
```

**Generated Machine**:
- Name: As specified
- Namespace: `openshift-machine-api`
- Labels: `machine.openshift.io/cluster-api-machine-role: master|worker`
- ProviderSpec: Valid Azure provider spec JSON

---

### `getValidMachineSet(name string)`
Creates a test MachineSet resource

**Example**:
```go
machineSet := getValidMachineSet("test-machineset")
```

**Generated MachineSet**:
- Name: As specified
- Namespace: `openshift-machine-api`
- Replicas: 3
- Labels: `machine.openshift.io/cluster-api-machine-role: worker`

---

## Future Test Enhancements

### Azure SDK Mocking
To improve coverage, add tests with mocked Azure clients:

```go
import (
    "go.uber.org/mock/gomock"
    mock_armnetwork "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/azuresdk/armnetwork"
)

func TestReconcileNICWithMock(t *testing.T) {
    ctrl := gomock.NewController(t)
    defer ctrl.Finish()

    mockNICClient := mock_armnetwork.NewMockInterfacesClient(ctrl)

    // Setup expectations
    mockNICClient.EXPECT().Get(gomock.Any(), "rg", "nic-name", nil).Return(
        armnetwork.InterfacesClientGetResponse{
            Interface: armnetwork.Interface{
                Properties: &armnetwork.InterfacePropertiesFormat{
                    ProvisioningState: pointerutils.ToPtr(armnetwork.ProvisioningStateFailed),
                },
            },
        },
        nil,
    )

    // Test reconciliation with mock
    rm := &reconcileManager{
        nicClient: mockNICClient,
        // ... other fields
    }

    err := rm.reconcileNIC(ctx, "nic-name")
    // Assert expectations
}
```

---

### Integration Tests
Add E2E tests that run against a real (or emulated) cluster:

```bash
# E2E test scenarios
- Create cluster with failed NIC
- Trigger event-driven reconciliation
- Verify NIC recovered
- Trigger periodic reconciliation
- Verify all NICs scanned
```

---

### Table-Driven Test Expansion

#### Expand `TestReconcile` with more scenarios:
```go
tests := []struct {
    name           string
    clusterEnabled bool
    request        ctrl.Request
    machines       []machinev1beta1.Machine
    expectedResult ctrl.Result
    expectError    bool
}{
    {
        name: "Periodic reconciliation with healthy NICs",
        clusterEnabled: true,
        request: ctrl.Request{Name: "cluster"},
        // ...
    },
    {
        name: "Event-driven with master machine change",
        clusterEnabled: true,
        request: ctrl.Request{Name: "cluster-abc-master-0"},
        // ...
    },
}
```

---

## Test Maintenance

### Adding New Tests
1. Follow table-driven test pattern
2. Use descriptive test names
3. Add comments for complex logic
4. Update coverage metrics

### Updating Existing Tests
When changing controller logic:
1. Update affected test expectations
2. Run full test suite: `go test ./pkg/operator/controllers/nic/...`
3. Verify coverage hasn't decreased: `go test -cover`
4. Update this documentation

---

## Continuous Integration

### Pre-Commit Checks
```bash
# Run before committing
make test-go
# or
go test ./pkg/operator/controllers/nic/...
make lint-go
```

### CI Pipeline
Tests run automatically on:
- Pull request creation
- Commits to PR branches
- Merge to main

**Required Checks**:
- ✅ All tests pass
- ✅ No linting errors
- ✅ Coverage ≥ 40%

---

## Coverage Goals

| Component | Current | Target |
|-----------|---------|--------|
| Helper Functions | 100% | 100% ✅ |
| Reconcile Logic | 72.2% | 80% |
| Overall | 44.8% | 60% |

**To Improve Coverage**:
1. Add Azure SDK mocks for `reconcileNIC()` and `reconcileAllNICs()`
2. Add integration tests for `SetupWithManager()`
3. Test error paths in `Reconcile()`

---

## Troubleshooting

### Test Failures

#### "undefined: client"
**Issue**: Missing import
**Fix**: Add `"sigs.k8s.io/controller-runtime/pkg/client"`

#### "panic: runtime error: invalid memory address"
**Issue**: Nil Azure client in tests
**Fix**: Don't call Azure APIs in unit tests, or mock the client

#### "controller is disabled" in logs
**Issue**: Test cluster has NIC controller disabled
**Fix**: Use `getValidClusterInstance(true)` instead of `getValidClusterInstance(false)`

---

### Coverage Not Updating

```bash
# Clean test cache
go clean -testcache

# Re-run tests with coverage
go test ./pkg/operator/controllers/nic/... -cover
```

---

## Best Practices

### ✅ Do's
- Use table-driven tests for multiple scenarios
- Test both success and error paths
- Use descriptive test names
- Mock external dependencies (Azure SDK)
- Keep tests fast (< 1 second per test)

### ❌ Don'ts
- Don't make real Azure API calls in unit tests
- Don't test implementation details
- Don't have flaky tests
- Don't skip error checking in tests
- Don't duplicate test logic

---

## References

- [ARO-RP Testing Guidelines](../../../docs/testing.md)
- [Kubernetes Controller Testing](https://book.kubebuilder.io/cronjob-tutorial/writing-tests.html)
- [Go Testing Best Practices](https://go.dev/doc/tutorial/add-a-test)
- [Gomock Documentation](https://github.com/uber-go/mock)

---

## Contact

**Questions about tests?**
- Slack: #aro-platform
- Owner: ARO Platform Team
