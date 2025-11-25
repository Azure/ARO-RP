# NIC Reconciliation Controller - Implementation Guide

## Table of Contents
- [Overview](#overview)
- [Architecture](#architecture)
- [Code Changes](#code-changes)
- [Implementation Details](#implementation-details)
- [Testing](#testing)
- [Troubleshooting](#troubleshooting)

---

## Overview

### Problem Statement
Network Interface Cards (NICs) in Azure can sometimes enter a "Failed" or "Canceled" provisioning state due to:
- Transient Azure platform issues
- Network policy conflicts
- Subnet capacity problems
- Race conditions during node scaling

When NICs fail, VMs cannot be created, leading to:
- Failed cluster scaling operations
- Unreachable nodes
- Manual intervention required by SREs

### Solution
Implemented a **hybrid reconciliation controller** that:
1. **Event-Driven**: Responds immediately to Machine/MachineSet changes
2. **Periodic**: Scans all NICs every hour as a safety net
3. **Self-Healing**: Automatically retries failed NIC provisioning

---

## Architecture

### Controller Pattern

```
┌─────────────────────────────────────────────────────────────┐
│                    Kubernetes API Server                    │
│         (Machine, MachineSet, Cluster Resources)            │
└───────────────┬─────────────────────────────────────────────┘
                │
                │ ① Events (CREATE/UPDATE/DELETE)
                │
                ▼
┌─────────────────────────────────────────────────────────────┐
│            Controller-Runtime Informers/Cache               │
│                                                             │
│  • Machine Watch (Master nodes)                             │
│  • MachineSet Watch (Worker nodes)                          │
│  • Cluster Watch (Periodic trigger every 1 hour)            │
└───────────────┬─────────────────────────────────────────────┘
                │
                │ ② Event → Work Queue
                │
                ▼
┌─────────────────────────────────────────────────────────────┐
│                   Reconcile() Function                       │
│                                                             │
│  ┌──────────────────────────────────────────────────────┐  │
│  │ Decision: Event-Driven vs Periodic?                  │  │
│  │                                                       │  │
│  │ if (request.Name == SingletonClusterName)            │  │
│  │    → Periodic: reconcileAllNICs()                    │  │
│  │ else                                                  │  │
│  │    → Event-Driven: reconcileNICForMachine()          │  │
│  └──────────────────────────────────────────────────────┘  │
└───────────────┬─────────────────────────────────────────────┘
                │
                │ ③ Reconciliation Logic
                │
                ▼
┌─────────────────────────────────────────────────────────────┐
│                    Azure NIC Client                          │
│                                                             │
│  • Get NIC State                                            │
│  • Check if Failed/Canceled                                 │
│  • Trigger Re-provisioning (CreateOrUpdate)                 │
│  • Wait for Completion                                      │
└─────────────────────────────────────────────────────────────┘
```

---

## Code Changes

### 1. New Files Created

#### `pkg/operator/controllers/nic/const.go`
**Purpose**: Define package-level constants

```go
package nic

const (
	machineNamespace = "openshift-machine-api"
)
```

**Explanation**:
- `machineNamespace`: All Machine and MachineSet resources live in `openshift-machine-api` namespace
- This constant is used when querying Kubernetes API for machine resources

---

#### `pkg/operator/controllers/nic/doc.go`
**Purpose**: Package documentation (Go convention for package-level docs)

```go
// Package nic reconciles failed Network Interface Cards (NICs) in the cluster.
//
// This controller implements a hybrid reconciliation strategy:
// 1. Event-Driven: Watches Machine/MachineSet resources and reconciles their NICs on changes
// 2. Periodic: Scans all NICs in the resource group every hour as a safety net
//
// The controller detects NICs in "Failed" or "Canceled" provisioning states and attempts
// to recover them by triggering Azure re-provisioning.
```

**Explanation**:
- Visible when running `go doc github.com/Azure/ARO-RP/pkg/operator/controllers/nic`
- Explains the controller's purpose and strategy at a glance

---

#### `pkg/operator/controllers/nic/nic_controller.go`
**Purpose**: Main controller implementation (348 lines)

### File Structure Overview:

```
nic_controller.go
├── Imports (lines 1-30)
├── Constants (lines 32-40)
│   └── ControllerName, periodicReconcileInterval
├── Type Definitions (lines 42-62)
│   ├── Reconciler (main controller struct)
│   └── reconcileManager (per-request manager)
├── Constructor (lines 64-70)
│   └── NewReconciler()
├── Core Reconciliation (lines 72-148)
│   ├── Reconcile() - entry point
│   ├── reconcileNICForMachine() - event-driven path
│   ├── reconcileNICsForMachineSet() - handles MachineSet events
│   └── reconcileAllNICs() - periodic scan path
├── NIC Reconciliation Logic (lines 277-309)
│   └── reconcileNIC() - actual Azure API calls
├── Controller Registration (lines 311-331)
│   └── SetupWithManager() - registers watches
└── Helper Functions (lines 333-388)
    ├── extractNICNameFromMachine()
    ├── isNICInFailedState()
    └── isNotFoundError()
```

---

### 2. Modified Files

#### `pkg/operator/flags.go`

**Change 1: Added NIC flag constant**
```diff
+ NICEnabled                         = "aro.nic.enabled"
```
**Location**: Line 24 (after `MonitoringEnabled`)

**Explanation**:
- Defines the operator flag key used to enable/disable the NIC controller
- Follows ARO naming convention: `aro.<controller-name>.enabled`
- This flag can be set in the Cluster CR's `spec.operatorFlags` field

---

**Change 2: Enabled by default**
```diff
+ NICEnabled:                         FlagTrue,
```
**Location**: Line 77 (in `DefaultOperatorFlags()` function)

**Explanation**:
- NIC controller is **enabled by default** for all new clusters
- Set to `FlagTrue` (which equals string `"true"`)
- Can be disabled by setting `aro.nic.enabled: "false"` in operator flags

---

#### `cmd/aro/operator.go`

**Change 1: Import NIC controller package**
```diff
+ "github.com/Azure/ARO-RP/pkg/operator/controllers/nic"
```
**Location**: Line 43 (after `monitoring` import)

**Explanation**:
- Makes the NIC controller package available
- Allows calling `nic.NewReconciler()` and `nic.ControllerName`

---

**Change 2: Register NIC controller**
```diff
+ if err = (nic.NewReconciler(
+     log.WithField("controller", nic.ControllerName),
+     client)).SetupWithManager(mgr); err != nil {
+     return fmt.Errorf("unable to create controller %s: %v", nic.ControllerName, err)
+ }
```
**Location**: Lines 169-173 (after subnets controller, before machine controller)

**Explanation**:
- **Creates** a new NIC reconciler with:
  - Logger scoped to this controller (for log filtering)
  - Kubernetes client (for reading Machine/MachineSet/Cluster resources)
- **Registers** the controller with the manager (sets up watches and event handlers)
- **Placement**: Registered alongside other master-only controllers
- **Error handling**: Returns error if controller setup fails, preventing operator startup

---

## Implementation Details

### Core Components

#### 1. Reconciler Struct
```go
type Reconciler struct {
	log    *logrus.Entry
	client client.Client
}
```

**Fields**:
- `log`: Structured logger for this controller (includes `controller=NIC` field)
- `client`: Kubernetes client for reading Machine/MachineSet/Cluster resources

**Purpose**: Main controller struct, implements `reconcile.Reconciler` interface

---

#### 2. reconcileManager Struct
```go
type reconcileManager struct {
	log            *logrus.Entry
	client         client.Client
	instance       *arov1alpha1.Cluster
	subscriptionID string
	resourceGroup  string
	infraID        string
	nicClient      armnetwork.InterfacesClient
}
```

**Fields**:
- `instance`: ARO Cluster CR (contains config and operator flags)
- `subscriptionID`: Azure subscription ID for NIC operations
- `resourceGroup`: Cluster resource group (where NICs are created)
- `infraID`: Cluster infrastructure ID (used as NIC name prefix)
- `nicClient`: Azure SDK client for NIC operations (Get, List, CreateOrUpdate)

**Purpose**: Encapsulates per-request state and Azure clients

**Why separate from Reconciler?**:
- Azure clients are created per-request (credential refresh)
- Cluster-specific data changes per reconciliation
- Cleaner separation of concerns

---

### Key Functions Explained

#### 1. `Reconcile()` - Entry Point
**Location**: Lines 72-148

```go
func (r *Reconciler) Reconcile(ctx context.Context, request ctrl.Request) (ctrl.Result, error)
```

**Flow**:
```
1. Get Cluster CR
   ├─ Check if NIC controller is enabled (aro.nic.enabled flag)
   └─ Exit early if disabled

2. Initialize Azure Clients
   ├─ Parse Azure environment
   ├─ Create credential (DefaultAzureCredential = MSI or Service Principal)
   └─ Create NIC client

3. Extract Cluster Info
   ├─ Subscription ID
   ├─ Resource Group (where NICs live)
   └─ InfraID (cluster identifier, e.g., "cluster-abc-123")

4. Create reconcileManager
   └─ Bundle all context for this reconciliation

5. Route to Appropriate Handler
   if request.Name == "cluster":
       → Periodic reconciliation (scan all NICs)
   else:
       → Event-driven reconciliation (specific Machine/MachineSet)

6. Return Result
   ├─ Success: RequeueAfter 1 hour (for periodic)
   └─ Error: RequeueAfter 5 minutes (retry)
```

**Key Decision Point** (lines 130-139):
```go
if request.Name == arov1alpha1.SingletonClusterName {
    // Periodic: triggered by Cluster object watch
    reconErr = manager.reconcileAllNICs(ctx)
} else {
    // Event-driven: triggered by Machine/MachineSet watch
    reconErr = manager.reconcileNICForMachine(ctx, request.Name, request.Namespace)
}
```

**Why this works**:
- Cluster watch triggers with `request.Name = "cluster"` (singleton)
- Machine watch triggers with `request.Name = "<machine-name>"`
- MachineSet watch triggers with `request.Name = "<machineset-name>"`

---

#### 2. `reconcileNICForMachine()` - Event-Driven Path
**Location**: Lines 150-177

```go
func (rm *reconcileManager) reconcileNICForMachine(ctx context.Context, resourceName, namespace string) error
```

**Flow**:
```
1. Get Machine Resource
   ├─ Query Kubernetes API: GET /apis/machine.openshift.io/v1beta1/namespaces/openshift-machine-api/machines/<name>
   └─ If not found → Try MachineSet path (resource might be a MachineSet)

2. Extract NIC Name
   ├─ Unmarshal Machine.Spec.ProviderSpec (Azure-specific JSON)
   ├─ Derive NIC name: <machine-name>-nic
   └─ Example: "cluster-abc-master-0" → "cluster-abc-master-0-nic"

3. Reconcile NIC
   └─ Call reconcileNIC() with derived name
```

**Example Scenario**:
```bash
# SRE scales workers from 3 to 5
oc scale machineset worker --replicas=5

# Events triggered:
Event 1: MachineSet "worker" UPDATED
         → reconcileNICForMachine("worker", "openshift-machine-api")
         → Lists all machines owned by "worker" MachineSet
         → Reconciles NICs for all 5 workers

Event 2: Machine "worker-xyz-1" CREATED
         → reconcileNICForMachine("worker-xyz-1", "openshift-machine-api")
         → Extracts NIC name: "worker-xyz-1-nic"
         → Reconciles this specific NIC

Event 3: Machine "worker-xyz-2" CREATED
         → Similar to Event 2
```

---

#### 3. `reconcileNICsForMachineSet()` - Handle MachineSet Events
**Location**: Lines 179-208

```go
func (rm *reconcileManager) reconcileNICsForMachineSet(ctx context.Context, machineSetName, namespace string) error
```

**Flow**:
```
1. List All Machines in Namespace
   └─ GET /apis/machine.openshift.io/v1beta1/namespaces/openshift-machine-api/machines

2. Filter by OwnerReference
   for each machine:
       if machine.OwnerReferences contains:
          - Kind: MachineSet
          - Name: <machineSetName>
       then:
          → This machine belongs to our MachineSet
          → Reconcile its NIC

3. Aggregate Errors
   └─ Return combined error if any NIC reconciliation failed
```

**Why needed?**:
- MachineSet events don't directly contain Machine names
- Need to find all Machines owned by this MachineSet
- Ensures all worker NICs are reconciled when MachineSet changes

**Owner Reference Example**:
```yaml
apiVersion: machine.openshift.io/v1beta1
kind: Machine
metadata:
  name: worker-abc-xyz-1
  ownerReferences:
  - apiVersion: machine.openshift.io/v1beta1
    kind: MachineSet          # ← This is what we match
    name: worker-abc          # ← This is what we filter by
    uid: ...
```

---

#### 4. `reconcileAllNICs()` - Periodic Safety Net
**Location**: Lines 210-257

```go
func (rm *reconcileManager) reconcileAllNICs(ctx context.Context) error
```

**Flow**:
```
1. List All NICs in Resource Group
   └─ Azure API: GET /subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.Network/networkInterfaces

2. Filter by InfraID
   for each NIC:
       if NIC.Name starts with infraID:
          → This NIC belongs to our cluster
          → Check its state

3. Detect Failed NICs
   if NIC.ProvisioningState == "Failed" OR "Canceled":
       → Log warning
       → Attempt reconciliation

4. Metrics
   └─ Log: "checked X NICs, found Y failed NICs"

5. Aggregate Errors
   └─ Return combined error for all failed reconciliations
```

**Example Log Output**:
```
INFO: running periodic NIC reconciliation (full scan)
INFO: scanning all NICs in resource group: cluster-abc-rg
WARN: found NIC in failed state: cluster-abc-master-1-nic, provisioning state: Failed
INFO: attempting to reconcile NIC: cluster-abc-master-1-nic
INFO: NIC cluster-abc-master-1-nic reconciled successfully, new state: Succeeded
INFO: periodic scan complete: checked 8 NICs, found 1 failed NICs
```

**Why infraID filtering?**:
- Resource group may contain NICs from multiple sources
- InfraID uniquely identifies this cluster's resources
- Example: `cluster-abc-123-master-0-nic` starts with `cluster-abc-123`

---

#### 5. `reconcileNIC()` - Actual Reconciliation Logic
**Location**: Lines 277-309

```go
func (rm *reconcileManager) reconcileNIC(ctx context.Context, nicName string) error
```

**Flow**:
```
1. Get Current NIC State
   GET /subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.Network/networkInterfaces/{nicName}

   If 404 Not Found:
       → NIC doesn't exist (may have been deleted)
       → Nothing to reconcile
       → Return nil (success)

2. Check Provisioning State
   if state NOT IN ["Failed", "Canceled"]:
       → NIC is healthy (Succeeded, Updating, Creating, etc.)
       → Skip reconciliation
       → Return nil (success)

3. Trigger Re-provisioning
   PUT /subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.Network/networkInterfaces/{nicName}
   Body: <current NIC configuration>

   Azure behavior:
       → Receives same NIC config
       → Detects it's in Failed state
       → Retries provisioning from scratch
       → May succeed if transient issue resolved

4. Wait for Completion
   Poll operation status every few seconds
   Until: state transitions to terminal (Succeeded/Failed)

5. Return Result
   Success: state = "Succeeded"
   Failure: state still = "Failed" (will retry in 5 minutes)
```

**Azure API Details**:

**Initial State**:
```json
{
  "name": "cluster-abc-master-0-nic",
  "properties": {
    "provisioningState": "Failed",
    "ipConfigurations": [...],
    "networkSecurityGroup": {...}
  }
}
```

**Reconciliation Call** (`BeginCreateOrUpdate`):
```http
PUT /subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.Network/networkInterfaces/cluster-abc-master-0-nic
Content-Type: application/json

{
  "properties": {
    "ipConfigurations": [...],  # Same config
    "networkSecurityGroup": {...}  # Same config
  }
}
```

**Azure Processing**:
1. Receives PUT request with existing config
2. Detects resource exists in Failed state
3. Deletes failed provisioning attempt
4. Retries provisioning with same config
5. If successful → state: "Succeeded"
6. If failed again → state: "Failed" (controller will retry later)

---

#### 6. `SetupWithManager()` - Watch Registration
**Location**: Lines 311-331

```go
func (r *Reconciler) SetupWithManager(mgr ctrl.Manager) error
```

**Breakdown**:
```go
return ctrl.NewControllerManagedBy(mgr).
```
- Creates a new controller builder managed by `mgr`

```go
For(&arov1alpha1.Cluster{}, builder.WithPredicates(
    predicate.And(
        predicates.AROCluster,
        predicate.GenerationChangedPredicate{},
    ),
)).
```
- **PRIMARY WATCH**: Cluster resource (singleton named "cluster")
- **Predicates** (filters):
  - `AROCluster`: Only watch ARO cluster resources (not other CRs)
  - `GenerationChangedPredicate`: Only trigger on spec changes (not status updates)
- **Triggers**: On Cluster spec changes + periodic (every 1 hour via RequeueAfter)
- **Calls**: `Reconcile(ctx, Request{Name: "cluster"})`

```go
Watches(
    &source.Kind{Type: &machinev1beta1.Machine{}},
    &handler.EnqueueRequestForObject{},
    builder.WithPredicates(predicates.MachineRoleMaster),
).
```
- **SECONDARY WATCH**: Master Machine resources
- **Predicate**: Only master machines (label `machine.openshift.io/cluster-api-machine-role: master`)
- **Triggers**: On master Machine CREATE/UPDATE/DELETE
- **Calls**: `Reconcile(ctx, Request{Name: "<machine-name>"})`

```go
Watches(
    &source.Kind{Type: &machinev1beta1.MachineSet{}},
    &handler.EnqueueRequestForObject{},
    builder.WithPredicates(predicates.MachineRoleWorker),
).
```
- **SECONDARY WATCH**: Worker MachineSet resources
- **Predicate**: Only worker machinesets (label `machine.openshift.io/cluster-api-machine-role: worker`)
- **Triggers**: On worker MachineSet CREATE/UPDATE/DELETE
- **Calls**: `Reconcile(ctx, Request{Name: "<machineset-name>"})`

```go
Named(ControllerName).
Complete(r)
```
- Sets controller name to "NIC" (for metrics and logs)
- Completes the builder and registers the controller

**Event Flow Example**:
```
┌─────────────────────────────────────────────────────────────┐
│ Kubernetes API: New Machine Created                         │
│ Name: cluster-abc-master-2                                   │
│ Label: machine.openshift.io/cluster-api-machine-role=master │
└───────────────┬─────────────────────────────────────────────┘
                │
                ▼
┌─────────────────────────────────────────────────────────────┐
│ Informer Cache: Detects Machine CREATE event                │
└───────────────┬─────────────────────────────────────────────┘
                │
                ▼
┌─────────────────────────────────────────────────────────────┐
│ Predicate Check: Is this a master machine?                  │
│ YES → Pass event to work queue                              │
└───────────────┬─────────────────────────────────────────────┘
                │
                ▼
┌─────────────────────────────────────────────────────────────┐
│ Work Queue: Add Request{Name: "cluster-abc-master-2"}       │
└───────────────┬─────────────────────────────────────────────┘
                │
                ▼
┌─────────────────────────────────────────────────────────────┐
│ Reconcile(): Called with Request{Name: "cluster-abc-master-2"}│
│ → Event-driven path                                         │
│ → Reconcile NIC: cluster-abc-master-2-nic                   │
└─────────────────────────────────────────────────────────────┘
```

---

### Helper Functions

#### 1. `extractNICNameFromMachine()`
**Location**: Lines 336-357

```go
func extractNICNameFromMachine(machine *machinev1beta1.Machine) (string, error)
```

**Logic**:
```
1. Check if ProviderSpec exists
   if nil → return error

2. Unmarshal Azure Provider Spec
   JSON structure:
   {
     "vmSize": "Standard_D4s_v3",
     "osDisk": {...},
     "networkResourceGroup": "...",
     ...
   }

3. Derive NIC Name
   Rule: <machine-name>-nic

   Example:
   Machine: cluster-abc-master-0
   NIC:     cluster-abc-master-0-nic
```

**Why this naming convention?**:
- Azure Machine API (used by OpenShift) automatically creates NICs with `-nic` suffix
- This is a reliable convention across all ARO clusters
- No need to query Azure API to find the NIC name

**Example Machine Manifest**:
```yaml
apiVersion: machine.openshift.io/v1beta1
kind: Machine
metadata:
  name: cluster-abc-master-0
spec:
  providerSpec:
    value:
      kind: AzureMachineProviderSpec
      vmSize: Standard_D8s_v3
      # NIC will be created as: cluster-abc-master-0-nic
```

---

#### 2. `isNICInFailedState()`
**Location**: Lines 359-376

```go
func isNICInFailedState(nic *armnetwork.Interface) bool
```

**Logic**:
```
1. Null Check
   if NIC.Properties == nil OR NIC.Properties.ProvisioningState == nil:
       → Can't determine state
       → Assume healthy (return false)

2. Extract State String
   state := string(nic.Properties.ProvisioningState)

3. Check Against Failed States
   failedStates := ["Failed", "Canceled"]

   for each failedState:
       if state == failedState (case-insensitive):
           → NIC is in failed state
           → return true

4. Otherwise
   → NIC is healthy
   → return false
```

**Azure NIC Provisioning States**:
| State | Meaning | Action |
|-------|---------|--------|
| `Creating` | Initial provisioning | Wait (healthy) |
| `Updating` | Configuration change | Wait (healthy) |
| `Succeeded` | Successfully provisioned | None (healthy) |
| `Failed` | Provisioning failed | **Reconcile** |
| `Canceled` | Operation canceled | **Reconcile** |
| `Deleting` | Being deleted | Wait (healthy) |

**Why treat "Canceled" as failed?**:
- Sometimes indicates partial failure
- May be due to timeout or conflict
- Safe to retry provisioning

---

#### 3. `isNotFoundError()`
**Location**: Lines 378-388

```go
func isNotFoundError(err error) bool
```

**Logic**:
```
1. Null Check
   if err == nil:
       → No error
       → return false

2. Extract Error Message
   errStr := err.Error()

3. Check Error Patterns
   if errStr contains "ResourceNotFound":    → Azure SDK error
   OR errStr contains "NotFound":            → Generic not found
   OR errStr contains "404":                 → HTTP status code
       → Resource doesn't exist
       → return true

4. Otherwise
   → Different type of error (permissions, network, etc.)
   → return false
```

**Why needed?**:
- Distinguish between "NIC doesn't exist" (OK) vs "Permission denied" (ERROR)
- If NIC doesn't exist, nothing to reconcile (success case)
- If permission error, need to surface to user

**Example Errors**:
```go
// Not Found → isNotFoundError() = true
"GET https://...networkInterfaces/xyz: 404 Not Found: ResourceNotFound"

// Permission Error → isNotFoundError() = false
"GET https://...networkInterfaces/xyz: 403 Forbidden: AuthorizationFailed"

// Network Error → isNotFoundError() = false
"GET https://...networkInterfaces/xyz: dial tcp: connection refused"
```

---

## Testing

### Unit Testing Strategy

Create `pkg/operator/controllers/nic/nic_controller_test.go`:

```go
package nic

import (
	"context"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v6"
	machinev1beta1 "github.com/openshift/api/machine/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	"github.com/Azure/ARO-RP/pkg/util/pointerutils"
)

// Test: extractNICNameFromMachine
func TestExtractNICNameFromMachine(t *testing.T) {
	tests := []struct {
		name        string
		machine     *machinev1beta1.Machine
		expectedNIC string
		expectError bool
	}{
		{
			name: "Master machine",
			machine: &machinev1beta1.Machine{
				ObjectMeta: metav1.ObjectMeta{
					Name: "cluster-abc-master-0",
				},
				Spec: machinev1beta1.MachineSpec{
					ProviderSpec: machinev1beta1.ProviderSpec{
						Value: &runtime.RawExtension{
							Raw: []byte(`{"vmSize":"Standard_D8s_v3"}`),
						},
					},
				},
			},
			expectedNIC: "cluster-abc-master-0-nic",
			expectError: false,
		},
		{
			name: "Worker machine",
			machine: &machinev1beta1.Machine{
				ObjectMeta: metav1.ObjectMeta{
					Name: "cluster-abc-worker-xyz-1",
				},
				Spec: machinev1beta1.MachineSpec{
					ProviderSpec: machinev1beta1.ProviderSpec{
						Value: &runtime.RawExtension{
							Raw: []byte(`{"vmSize":"Standard_D4s_v3"}`),
						},
					},
				},
			},
			expectedNIC: "cluster-abc-worker-xyz-1-nic",
			expectError: false,
		},
		{
			name: "Missing ProviderSpec",
			machine: &machinev1beta1.Machine{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-machine",
				},
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nicName, err := extractNICNameFromMachine(tt.machine)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if nicName != tt.expectedNIC {
				t.Errorf("Expected NIC name %q, got %q", tt.expectedNIC, nicName)
			}
		})
	}
}

// Test: isNICInFailedState
func TestIsNICInFailedState(t *testing.T) {
	tests := []struct {
		name   string
		nic    *armnetwork.Interface
		failed bool
	}{
		{
			name: "NIC in Failed state",
			nic: &armnetwork.Interface{
				Properties: &armnetwork.InterfacePropertiesFormat{
					ProvisioningState: pointerutils.ToPtr(armnetwork.ProvisioningStateFailed),
				},
			},
			failed: true,
		},
		{
			name: "NIC in Canceled state",
			nic: &armnetwork.Interface{
				Properties: &armnetwork.InterfacePropertiesFormat{
					ProvisioningState: pointerutils.ToPtr(armnetwork.ProvisioningStateCanceled),
				},
			},
			failed: true,
		},
		{
			name: "NIC in Succeeded state",
			nic: &armnetwork.Interface{
				Properties: &armnetwork.InterfacePropertiesFormat{
					ProvisioningState: pointerutils.ToPtr(armnetwork.ProvisioningStateSucceeded),
				},
			},
			failed: false,
		},
		{
			name: "NIC in Updating state",
			nic: &armnetwork.Interface{
				Properties: &armnetwork.InterfacePropertiesFormat{
					ProvisioningState: pointerutils.ToPtr(armnetwork.ProvisioningStateUpdating),
				},
			},
			failed: false,
		},
		{
			name:   "NIC with nil Properties",
			nic:    &armnetwork.Interface{},
			failed: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isNICInFailedState(tt.nic)
			if result != tt.failed {
				t.Errorf("Expected isNICInFailedState() = %v, got %v", tt.failed, result)
			}
		})
	}
}
```

**Run Tests**:
```bash
go test -v ./pkg/operator/controllers/nic/...
```

---

### Integration Testing

#### Test Scenario 1: Event-Driven Reconciliation

**Setup**:
1. Deploy ARO cluster with NIC controller enabled
2. Manually put a NIC in failed state (simulate Azure issue)

**Steps**:
```bash
# 1. Get a worker machine
MACHINE=$(oc get machines -n openshift-machine-api -l machine.openshift.io/cluster-api-machine-role=worker -o name | head -1)

# 2. Get the NIC name (derived from machine name)
MACHINE_NAME=$(echo $MACHINE | cut -d/ -f2)
NIC_NAME="${MACHINE_NAME}-nic"

# 3. Simulate NIC failure (use Azure CLI or Portal to set state to Failed)
# This is typically done by:
# - Deleting the NIC
# - Recreating it with wrong subnet (will fail)

# 4. Trigger reconciliation by updating machine
oc annotate $MACHINE test-annotation=trigger-reconcile -n openshift-machine-api --overwrite

# 5. Watch controller logs
oc logs -n openshift-azure-operator deployment/aro-operator-master -f | grep NIC

# Expected logs:
# INFO: running event-driven NIC reconciliation for: <machine-name>
# INFO: reconciling NIC for resource: openshift-machine-api/<machine-name>
# WARN: NIC <nic-name> is in failed state: Failed
# INFO: attempting to reconcile NIC: <nic-name>
# INFO: NIC <nic-name> reconciled successfully, new state: Succeeded
```

**Verification**:
```bash
# Check NIC state in Azure
az network nic show --resource-group <cluster-rg> --name $NIC_NAME --query provisioningState
# Expected: "Succeeded"
```

---

#### Test Scenario 2: Periodic Reconciliation

**Setup**:
1. Create a NIC outside of normal machine operations (orphaned NIC)
2. Set it to failed state
3. Wait for periodic scan

**Steps**:
```bash
# 1. Wait for periodic reconciliation (happens every 1 hour)
# OR trigger manually by editing Cluster CR
oc annotate cluster cluster test-trigger=periodic -n openshift-azure-operator --overwrite

# 2. Watch controller logs
oc logs -n openshift-azure-operator deployment/aro-operator-master -f | grep "periodic NIC reconciliation"

# Expected logs:
# INFO: running periodic NIC reconciliation (full scan)
# INFO: scanning all NICs in resource group: <cluster-rg>
# WARN: found NIC in failed state: <nic-name>, provisioning state: Failed
# INFO: attempting to reconcile NIC: <nic-name>
# INFO: NIC <nic-name> reconciled successfully, new state: Succeeded
# INFO: periodic scan complete: checked X NICs, found Y failed NICs
```

---

#### Test Scenario 3: Scaling Workers

**Steps**:
```bash
# 1. Scale workers
oc scale machineset <machineset-name> --replicas=5 -n openshift-machine-api

# 2. Watch controller logs
oc logs -n openshift-azure-operator deployment/aro-operator-master -f | grep NIC

# Expected: Controller reconciles NICs for each new machine as it's created
```

---

## Troubleshooting

### Common Issues

#### Issue 1: Controller Disabled

**Symptoms**:
- No NIC reconciliation happening
- Logs show: `controller is disabled`

**Solution**:
```bash
# Check operator flags
oc get cluster cluster -n openshift-azure-operator -o jsonpath='{.spec.operatorFlags}' | jq .

# Enable NIC controller
oc patch cluster cluster -n openshift-azure-operator --type=merge -p '
{
  "spec": {
    "operatorFlags": {
      "aro.nic.enabled": "true"
    }
  }
}
'
```

---

#### Issue 2: Permission Errors

**Symptoms**:
- Logs show: `failed to list NICs: 403 Forbidden`

**Root Cause**:
- Cluster MSI or Service Principal lacks permissions

**Solution**:
```bash
# Grant Network Contributor role to cluster identity
CLUSTER_MSI=$(oc get cluster cluster -n openshift-azure-operator -o jsonpath='{.spec.platformWorkloadIdentityProfile.platformWorkloadIdentities.network_controller.resourceID}')
RESOURCE_GROUP=$(oc get cluster cluster -n openshift-azure-operator -o jsonpath='{.spec.clusterResourceGroupID}' | cut -d/ -f5)

az role assignment create \
  --assignee $CLUSTER_MSI \
  --role "Network Contributor" \
  --scope "/subscriptions/<sub-id>/resourceGroups/$RESOURCE_GROUP"
```

---

#### Issue 3: NIC Still Failed After Reconciliation

**Symptoms**:
- Controller runs
- NIC reconciliation attempts but fails
- NIC remains in Failed state

**Diagnosis**:
```bash
# Get detailed NIC error
az network nic show --resource-group <cluster-rg> --name <nic-name> --query properties.provisioningFailedReason

# Common reasons:
# - Subnet full (no available IPs)
# - NSG conflict
# - Azure platform issue
```

**Solutions**:

**Subnet Full**:
```bash
# Check subnet capacity
az network vnet subnet show \
  --resource-group <vnet-rg> \
  --vnet-name <vnet-name> \
  --name <subnet-name> \
  --query "addressPrefix"

# Expand subnet or use different subnet
```

**NSG Conflict**:
```bash
# Check NSG attachment
az network nic show --resource-group <cluster-rg> --name <nic-name> \
  --query "networkSecurityGroup"

# Fix NSG configuration
```

---

#### Issue 4: High Reconciliation Rate

**Symptoms**:
- Controller reconciling same NIC repeatedly
- Logs show continuous reconciliation loop

**Root Cause**:
- NIC keeps failing provisioning
- Azure platform issue

**Solution**:
```bash
# 1. Check Azure Service Health
# https://portal.azure.com/#blade/Microsoft_Azure_Health/AzureHealthBrowseBlade/serviceIssues

# 2. Temporarily disable controller
oc patch cluster cluster -n openshift-azure-operator --type=merge -p '
{
  "spec": {
    "operatorFlags": {
      "aro.nic.enabled": "false"
    }
  }
}
'

# 3. Manually fix underlying issue
# 4. Re-enable controller
```

---

## Monitoring and Metrics

### Key Logs to Monitor

**Event-Driven Reconciliation**:
```
INFO: running event-driven NIC reconciliation for: <resource-name>
INFO: reconciling NIC for resource: <namespace>/<resource-name>
```

**Periodic Reconciliation**:
```
INFO: running periodic NIC reconciliation (full scan)
INFO: periodic scan complete: checked <X> NICs, found <Y> failed NICs
```

**Successful Reconciliation**:
```
INFO: NIC <nic-name> reconciled successfully, new state: Succeeded
```

**Errors**:
```
ERROR: NIC reconciliation failed: <error-message>
ERROR: failed to list NICs: <error-message>
```

### Prometheus Metrics (Future Enhancement)

Consider adding metrics:
```go
// Example metrics to add
nicReconciliationTotal = prometheus.NewCounterVec(
    prometheus.CounterOpts{
        Name: "aro_nic_reconciliation_total",
        Help: "Total number of NIC reconciliations attempted",
    },
    []string{"result"}, // "success", "failure"
)

nicReconciliationDuration = prometheus.NewHistogramVec(
    prometheus.HistogramOpts{
        Name: "aro_nic_reconciliation_duration_seconds",
        Help: "Duration of NIC reconciliation operations",
    },
    []string{"operation"}, // "get", "reconcile", "list"
)
```

---

## Best Practices

### 1. Gradual Rollout

**Phase 1: Observability** (Week 1-2)
- Deploy with controller **disabled** by default
- Enable on test clusters
- Monitor logs and behavior

**Phase 2: Opt-In** (Week 3-4)
- Enable on select production clusters
- Monitor metrics
- Collect feedback

**Phase 3: Default Enabled** (Week 5+)
- Set `FlagTrue` in `DefaultOperatorFlags()`
- All new clusters get controller enabled
- Existing clusters can opt-in

### 2. Rate Limiting

Consider adding rate limiting to prevent Azure API throttling:

```go
// In reconcileManager
type reconcileManager struct {
    // ... existing fields
    rateLimiter workqueue.RateLimiter
}

// Before Azure API call
if !rm.rateLimiter.When(nicName).IsZero() {
    time.Sleep(rm.rateLimiter.When(nicName))
}
```

### 3. Error Categorization

Categorize errors for better alerting:

```go
type ReconcileError struct {
    Type    ErrorType  // Transient, Permanent, Configuration
    Message string
    NICName string
}

// Only alert on Permanent errors
if err.Type == ErrorTypePermanent {
    sendAlert(err)
}
```

---

## Future Enhancements

### 1. Support for Load Balancer NICs
Currently only reconciles VM NICs. Could extend to:
- Load balancer backend pool NICs
- Application Gateway NICs
- Private endpoint NICs

### 2. Predictive Reconciliation
Detect patterns in NIC failures and proactively reconcile:
```go
// If NIC has failed 3+ times in past hour
if nicFailureCount(nicName, 1*time.Hour) >= 3 {
    // Switch to more aggressive retry
    reconcileWithBackoff(nicName, exponentialBackoff)
}
```

### 3. Cluster-Wide Health Check
Report NIC health in Cluster status:
```yaml
status:
  conditions:
  - type: NICHealthy
    status: "True"
    reason: "AllNICsSucceeded"
    message: "All 8 NICs are in Succeeded state"
```

---

## Summary

This implementation provides:

✅ **Event-Driven Response**: Immediate reconciliation on machine changes
✅ **Periodic Safety Net**: Hourly scans catch orphaned/missed NICs
✅ **Self-Healing**: Automatically retries failed NICs
✅ **Kubernetes-Native**: Integrates with controller-runtime patterns
✅ **Azure-Aware**: Uses Azure SDK best practices
✅ **Observable**: Rich logging for troubleshooting
✅ **Configurable**: Can be enabled/disabled via operator flags

The controller follows ARO-RP conventions and patterns, making it maintainable and consistent with the rest of the codebase.
