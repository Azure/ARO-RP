package dynamicworkaround

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"encoding/json"
	"fmt"
	"regexp"
)

const (
	// ControllerName is the unique controller name registered with the manager.
	ControllerName = "DynamicWorkaround"

	// SchemaVersion is the only catalog schemaVersion this controller accepts.
	// Future incompatible changes bump this; the operator refuses any catalog
	// whose schemaVersion does not match.
	SchemaVersion = "v1alpha1"

	// CatalogManagedByLabel marks every MachineConfig created by this controller.
	// The reconcile cleanup pass uses this label to find resources owned by
	// the catalog so it can prune ones no longer in the active catalog.
	CatalogManagedByLabel = "aro.openshift.io/dynamic-workaround"

	// CatalogNameLabel stores the workaround.Name from the catalog on the
	// applied MachineConfig so we can correlate live state to the spec.
	CatalogNameLabel = "aro.openshift.io/dynamic-workaround-name"

	// MachineConfigRoleLabel is the standard MCO role label.
	MachineConfigRoleLabel = "machineconfiguration.openshift.io/role"

	// MaxCatalogBytes caps how much body the fetcher will read. 1 MiB is plenty
	// for hundreds of MachineConfig fragments and stops a runaway endpoint from
	// OOMing the operator.
	MaxCatalogBytes int64 = 1 << 20

	// MaxWorkarounds caps catalog entry count as a second line of defence.
	MaxWorkarounds = 64
)

// nameRegex restricts workaround and MachineConfig names to a conservative
// DNS-label-ish subset. Anything stricter is the catalog publisher's problem.
var nameRegex = regexp.MustCompile(`^[a-z0-9]([-a-z0-9]{0,61}[a-z0-9])?$`)

// Catalog is the top-level manifest the operator fetches over HTTPS.
//
// Example:
//
//	{
//	  "schemaVersion": "v1alpha1",
//	  "catalogVersion": "2026-05-11.1",
//	  "workarounds": [ ... ]
//	}
type Catalog struct {
	// SchemaVersion must equal SchemaVersion. Used to gate breaking changes.
	SchemaVersion string `json:"schemaVersion"`

	// CatalogVersion is an opaque, monotonically meaningful string the publisher
	// controls (e.g. a date, semver, or git sha). It is recorded in logs and
	// reflected onto every applied MachineConfig so operators can correlate
	// drift to a specific catalog publication.
	CatalogVersion string `json:"catalogVersion"`

	// Workarounds is the list of catalog entries. Empty list is valid and means
	// "no workarounds should be applied" (cleanup runs as normal).
	Workarounds []Workaround `json:"workarounds"`
}

// Workaround is a single catalog entry.
type Workaround struct {
	// Name identifies the workaround across catalog revisions. Acts as the
	// cleanup key: removing a Name from the catalog removes the corresponding
	// live MachineConfig.
	Name string `json:"name"`

	// Description is informational only; included in MachineConfig annotations
	// so operators can see why a MC is on the cluster.
	Description string `json:"description,omitempty"`

	// MachineConfigName is the Kubernetes object name applied to the cluster.
	// Catalog publishers must use the standard MCO numeric prefix (e.g. 99-)
	// to ensure the MachineConfig sorts correctly relative to OpenShift's own.
	MachineConfigName string `json:"machineConfigName"`

	// Role is "master" or "worker"; sets the MCO role label on the MachineConfig.
	Role string `json:"role"`

	// Ignition is the Ignition config that will be marshaled into the
	// MachineConfig's spec.config.raw. The operator does NOT introspect it;
	// it only ensures it round-trips through json.Marshal.
	Ignition json.RawMessage `json:"ignition"`
}

// Validate runs cheap structural checks; semantic checks (parseable version
// strings etc.) happen during predicate evaluation so they get logged with
// per-workaround context.
func (c *Catalog) Validate() error {
	if c.SchemaVersion != SchemaVersion {
		return fmt.Errorf("unsupported schemaVersion %q (want %q)", c.SchemaVersion, SchemaVersion)
	}
	if c.CatalogVersion == "" {
		return fmt.Errorf("catalogVersion must be non-empty")
	}
	if len(c.Workarounds) > MaxWorkarounds {
		return fmt.Errorf("too many workarounds: %d > %d", len(c.Workarounds), MaxWorkarounds)
	}
	seen := make(map[string]struct{}, len(c.Workarounds))
	for i := range c.Workarounds {
		w := &c.Workarounds[i]
		if !nameRegex.MatchString(w.Name) {
			return fmt.Errorf("workaround[%d] name %q is not a valid DNS label", i, w.Name)
		}
		if _, dup := seen[w.Name]; dup {
			return fmt.Errorf("workaround[%d] name %q duplicated", i, w.Name)
		}
		seen[w.Name] = struct{}{}
		if !nameRegex.MatchString(w.MachineConfigName) {
			return fmt.Errorf("workaround %q: machineConfigName %q is not a valid DNS label", w.Name, w.MachineConfigName)
		}
		switch w.Role {
		case "master", "worker":
			// ok
		default:
			return fmt.Errorf("workaround %q: role must be \"master\" or \"worker\", got %q", w.Name, w.Role)
		}
		if len(w.Ignition) == 0 {
			return fmt.Errorf("workaround %q: ignition is required", w.Name)
		}
		// Ensure ignition is at least syntactically valid JSON; MCO will do the
		// deeper Ignition spec validation when it tries to render the config.
		var probe any
		if err := json.Unmarshal(w.Ignition, &probe); err != nil {
			return fmt.Errorf("workaround %q: ignition is not valid JSON: %w", w.Name, err)
		}
	}
	return nil
}
