package deploy

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

const (
	rpVMSSScriptPath       = "generator/scripts/rpVMSS.sh"
	utilServicesScriptPath = "generator/scripts/util-services.sh"
)

func TestBuildRestartScript(t *testing.T) {
	buildRestartScriptTests := []struct {
		name     string
		services []rpRestartService
		want     string
	}{
		{
			name: "empty",
			want: "",
		},
		{
			name:     "blocking",
			services: []rpRestartService{{name: "aro-rp"}},
			want:     "systemctl restart aro-rp",
		},
		{
			name:     "nonBlocking",
			services: []rpRestartService{{name: "aro-mimo-actuator", noBlock: true}},
			want:     "systemctl restart --no-block aro-mimo-actuator",
		},
		{
			name: "mixedPreservesOrder",
			services: []rpRestartService{
				{name: "aro-mimo-actuator", noBlock: true},
				{name: "aro-rp"},
			},
			want: "systemctl restart --no-block aro-mimo-actuator; systemctl restart aro-rp",
		},
	}

	for _, tt := range buildRestartScriptTests {
		t.Run(tt.name, func(t *testing.T) {
			if got := buildRestartScript(tt.services); got != tt.want {
				t.Errorf("buildRestartScript(%v) = %q, want %q", tt.services, got, tt.want)
			}
		})
	}
}

// TestRPRestartScript pins the rendered script. restartOldScaleset passes it to
// RunCommandAndWait, which does not inspect the run command's result, so a unit
// named here which does not exist on the host fails silently. Any change to
// this string should therefore be deliberate.
func TestRPRestartScript(t *testing.T) {
	want := "systemctl restart --no-block aro-mimo-actuator; " +
		"systemctl restart --no-block aro-mimo-scheduler; " +
		"systemctl restart aro-monitor; " +
		"systemctl restart aro-portal; " +
		"systemctl restart aro-rp"

	if rpRestartScript != want {
		t.Errorf("rpRestartScript = %q, want %q", rpRestartScript, want)
	}
}

// TestRPRestartServicesAreDefined checks that every unit named in
// rpRestartServices is installed on the RP scale set. A unit which is not would
// cause systemctl to fail on every instance, and because the result of the run
// command is never inspected the deployment would still report success.
func TestRPRestartServicesAreDefined(t *testing.T) {
	defined := unitsWithFiles(t)
	enabled := unitsEnabledOnRPVMSS(t)

	for _, s := range rpRestartServices {
		if !defined[s.name] {
			t.Errorf("rpRestartServices contains %q, but %s writes no unit file for it", s.name, utilServicesScriptPath)
		}
		if !enabled[s.name] {
			t.Errorf("rpRestartServices contains %q, but %s does not enable it on the RP scale set", s.name, rpVMSSScriptPath)
		}
	}
}

// TestRPRestartServicesCoverKeyVaultConsumers is the regression test for the
// defect this list previously carried. aro-mimo-actuator and aro-mimo-scheduler
// were deployed to the RP scale set and read secrets from Key Vault, but were
// not restarted when PreDeploy rotated one, so they ran indefinitely against
// the secret versions they had enumerated at start-up.
//
// The invariant is derived from the deployment scripts rather than restated
// here, so that adding a Key Vault-consuming service to the RP scale set fails
// this test until the author has decided how a rotation is to reach it.
func TestRPRestartServicesCoverKeyVaultConsumers(t *testing.T) {
	restarted := make(map[string]bool, len(rpRestartServices))
	for _, s := range rpRestartServices {
		restarted[s.name] = true
	}

	consumers := unitsPassedKeyVaultPrefix(t)
	if len(consumers) == 0 {
		t.Fatalf("unitsPassedKeyVaultPrefix() = empty, want at least one; has the format of %s changed?", utilServicesScriptPath)
	}

	enabled := unitsEnabledOnRPVMSS(t)

	for unit := range consumers {
		// Units deployed to other scale sets, aro-gateway among them, are out
		// of scope: restartOldScaleset only ever targets the RP scale set.
		if !enabled[unit] {
			continue
		}
		if !restarted[unit] {
			t.Errorf("%q is enabled on the RP scale set and is passed KEYVAULT_PREFIX, but is absent from rpRestartServices, so it will not pick up a rotated secret", unit)
		}
	}
}

// unitsWithFiles returns the set of systemd units for which util-services.sh
// writes a unit file.
// It must only be called from the same goroutine as started the test.
func unitsWithFiles(t *testing.T) map[string]bool {
	t.Helper()

	units := make(map[string]bool)
	for _, block := range parseServiceBlocks(t) {
		units[block.name] = true
	}
	return units
}

// unitsPassedKeyVaultPrefix returns the set of systemd units whose container is
// passed KEYVAULT_PREFIX, which is how a service is given the means to read the
// encryption secrets that PreDeploy rotates.
// It must only be called from the same goroutine as started the test.
func unitsPassedKeyVaultPrefix(t *testing.T) map[string]bool {
	t.Helper()

	units := make(map[string]bool)
	for _, block := range parseServiceBlocks(t) {
		if strings.Contains(block.body, "-e KEYVAULT_PREFIX") {
			units[block.name] = true
		}
	}
	return units
}

// A serviceBlock is the region of util-services.sh which configures one systemd
// unit.
type serviceBlock struct {
	name string
	body string
}

// serviceFilenameRE matches the assignments in util-services.sh which name a
// unit file. The script writes them in several forms: a prefixed variable
// (aro_rp_service_filename), a bare one (service_filename), and either quote
// style. All must be matched, because a line which is missed does not merely go
// unchecked — its body is attributed to the preceding block instead, which
// could credit one service's KEYVAULT_PREFIX to another.
var serviceFilenameRE = regexp.MustCompile(`[\s_]service_filename=['"]/etc/systemd/system/([^'"]+)\.service['"]`)

// unitFileAssignmentRE is a deliberately loose match on the same assignments,
// used only to check that serviceFilenameRE did not miss any. It is kept
// independent of serviceFilenameRE so that the two cannot drift together.
var unitFileAssignmentRE = regexp.MustCompile(`service_filename=['"]?/etc/systemd/system/`)

// parseServiceBlocks splits util-services.sh into one block per unit file that
// it writes. A block runs from the line naming a unit file to the line naming
// the next one, which is enough to attribute the podman arguments in a
// service's ExecStart to that service.
// It must only be called from the same goroutine as started the test.
func parseServiceBlocks(t *testing.T) []serviceBlock {
	t.Helper()

	var blocks []serviceBlock
	assignments := 0
	for _, line := range strings.Split(readScript(t, utilServicesScriptPath), "\n") {
		if unitFileAssignmentRE.MatchString(line) {
			assignments++
		}
		if m := serviceFilenameRE.FindStringSubmatch(line); m != nil {
			blocks = append(blocks, serviceBlock{name: m[1]})
			continue
		}
		if len(blocks) > 0 {
			blocks[len(blocks)-1].body += line + "\n"
		}
	}

	if len(blocks) == 0 {
		t.Fatalf("parseServiceBlocks() = empty, want at least one; has the format of %s changed?", utilServicesScriptPath)
	}

	// A shortfall means a unit file is named in a form serviceFilenameRE does
	// not recognise, so its configuration is being read as part of another
	// service's.
	if len(blocks) != assignments {
		t.Fatalf("parseServiceBlocks() returned %d blocks for %d unit file assignments in %s; a unit file is named in a form serviceFilenameRE does not match", len(blocks), assignments, utilServicesScriptPath)
	}

	return blocks
}

var aroServicesRE = regexp.MustCompile(`(?s)local -ra aro_services=\((.*?)\)`)

// unitsEnabledOnRPVMSS returns the set of systemd units enabled on the RP scale
// set when an instance is provisioned.
// It must only be called from the same goroutine as started the test.
func unitsEnabledOnRPVMSS(t *testing.T) map[string]bool {
	t.Helper()

	m := aroServicesRE.FindStringSubmatch(readScript(t, rpVMSSScriptPath))
	if m == nil {
		t.Fatalf("no aro_services array found in %s; has the format of the script changed?", rpVMSSScriptPath)
	}

	units := make(map[string]bool)
	for _, field := range strings.Fields(m[1]) {
		if name := strings.Trim(field, `"'`); name != "" {
			units[name] = true
		}
	}

	if len(units) == 0 {
		t.Fatalf("the aro_services array in %s parsed as empty", rpVMSSScriptPath)
	}
	return units
}

// readScript returns the contents of a deployment script.
// It must only be called from the same goroutine as started the test.
func readScript(t *testing.T, path string) string {
	t.Helper()

	b, err := os.ReadFile(filepath.FromSlash(path))
	if err != nil {
		t.Fatalf("os.ReadFile(%q) returned unexpected error: %v", path, err)
	}
	return string(b)
}
