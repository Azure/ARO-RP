package dynamicworkaround

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/sirupsen/logrus"

	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrlfake "sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	configv1 "github.com/openshift/api/config/v1"
	mcv1 "github.com/openshift/machine-config-operator/pkg/apis/machineconfiguration.openshift.io/v1"

	"github.com/Azure/ARO-RP/pkg/operator"
	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	"github.com/Azure/ARO-RP/pkg/util/azureclient"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/azuresdk/azsecrets"
	_ "github.com/Azure/ARO-RP/pkg/util/scheme"
)

// stubFetcher implements Fetcher for tests. It returns whatever the test sets
// in catalog (with err taking precedence). Recording fetched=true lets tests
// assert the controller did or did not attempt a network call.
type stubFetcher struct {
	catalog    *Catalog
	err        error
	fetched    bool
	gotName    string
	gotVersion string
}

func (s *stubFetcher) Fetch(_ context.Context, _ azsecrets.Client, name, version string) (*Catalog, error) {
	s.fetched = true
	s.gotName = name
	s.gotVersion = version
	if s.err != nil {
		return nil, s.err
	}
	return s.catalog, nil
}

// stubClientFactory builds a SecretsClientFactory that hands tests a recorder
// they can inspect after Reconcile. We don't need a working azsecrets.Client
// because the stubFetcher above ignores the client entirely.
type stubClientFactory struct {
	called       bool
	gotVaultURL  string
	gotEnv       string
	returnErr    error
	returnClient azsecrets.Client
}

func (s *stubClientFactory) factory() SecretsClientFactory {
	return func(env *azureclient.AROEnvironment, vaultURL string) (azsecrets.Client, error) {
		s.called = true
		s.gotVaultURL = vaultURL
		s.gotEnv = env.Name
		if s.returnErr != nil {
			return nil, s.returnErr
		}
		return s.returnClient, nil
	}
}

// fixtureClusterVersion returns a ClusterVersion CR with a single completed
// history entry — the minimum shape that satisfies version.GetClusterVersion.
func fixtureClusterVersion(ver string) *configv1.ClusterVersion {
	return &configv1.ClusterVersion{
		ObjectMeta: metav1.ObjectMeta{Name: "version"},
		Status: configv1.ClusterVersionStatus{
			History: []configv1.UpdateHistory{
				{State: configv1.CompletedUpdate, Version: ver},
			},
		},
	}
}

// fixtureAROCluster returns an ARO Cluster CR with the dynamic workaround
// flags set as the test wants. Other tests would typically use a helper
// builder but we deliberately keep this verbose so failures are easy to read.
func fixtureAROCluster(flags arov1alpha1.OperatorFlags) *arov1alpha1.Cluster {
	return &arov1alpha1.Cluster{
		ObjectMeta: metav1.ObjectMeta{Name: arov1alpha1.SingletonClusterName},
		Spec: arov1alpha1.ClusterSpec{
			Location:            "eastus",
			ArchitectureVersion: 2,
			AZEnvironment:       "AzurePublicCloud",
			OperatorFlags:       flags,
		},
	}
}

// catalogWith returns a Catalog containing a single workaround with the given
// name and MachineConfig name. Predicates live on the cluster side now, so
// the catalog doesn't carry one anymore — tests set per-cluster opt-in via
// the operator.DynamicWorkaroundPredicates flag in their fixtureAROCluster
// call.
func catalogWith(name, mcName string) *Catalog {
	return &Catalog{
		SchemaVersion:  SchemaVersion,
		CatalogVersion: "test-1",
		Workarounds: []Workaround{
			{
				Name:              name,
				MachineConfigName: mcName,
				Role:              "worker",
				Ignition:          json.RawMessage(`{"ignition":{"version":"3.2.0"}}`),
			},
		},
	}
}

// staleMC returns a MachineConfig that *looks like* one this controller
// produced (managed-by label set), so we can verify cleanup behaviour.
func staleMC(name string) *mcv1.MachineConfig {
	return &mcv1.MachineConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			Labels: map[string]string{
				CatalogManagedByLabel:  "true",
				CatalogNameLabel:       "stale-entry",
				MachineConfigRoleLabel: "worker",
			},
		},
	}
}

// foreignMC is a MachineConfig that does NOT carry the managed-by label — the
// controller must leave it alone even when cleaning up.
func foreignMC(name string) *mcv1.MachineConfig {
	return &mcv1.MachineConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:   name,
			Labels: map[string]string{MachineConfigRoleLabel: "worker"},
		},
	}
}

func newReconciler(c client.Client, f Fetcher, cf SecretsClientFactory) *Reconciler {
	return NewReconciler(logrus.NewEntry(logrus.StandardLogger()), c, f, cf)
}

// mcExists returns true iff a MachineConfig with the given name exists on the
// fake cluster. Used by cleanup/apply assertions.
func mcExists(t *testing.T, c client.Client, name string) bool {
	t.Helper()
	mc := &mcv1.MachineConfig{}
	err := c.Get(context.Background(), types.NamespacedName{Name: name}, mc)
	switch {
	case err == nil:
		return true
	case kerrors.IsNotFound(err):
		return false
	default:
		t.Fatalf("get %q: %v", name, err)
		return false
	}
}

const (
	testSecretURI = "https://aro-test-vault.vault.azure.net/secrets/dynamic-workaround-catalog"

	// predicateAlways is a per-cluster opt-in that unconditionally applies
	// the test-wa workaround. Most tests use this so they can focus on the
	// catalog/cleanup/fetcher behaviour without re-stating CEL semantics.
	predicateAlways = `{"test-wa":"true"}`
)

func TestReconcileDisabledCleansUp(t *testing.T) {
	// Pre-seed a stale catalog-managed MC plus a foreign MC. With the flag
	// off the controller must delete the stale one but leave the foreign one.
	stale := staleMC("99-aro-stale")
	foreign := foreignMC("99-someone-else")
	aro := fixtureAROCluster(arov1alpha1.OperatorFlags{
		operator.DynamicWorkaroundCatalogEnabled:   operator.FlagFalse,
		operator.DynamicWorkaroundCatalogSecretURI: testSecretURI,
		operator.DynamicWorkaroundPredicates:       predicateAlways,
	})
	c := ctrlfake.NewClientBuilder().WithObjects(aro, stale, foreign).Build()
	sf := &stubFetcher{}
	cf := &stubClientFactory{}
	r := newReconciler(c, sf, cf.factory())

	res, err := r.Reconcile(context.Background(), reconcile.Request{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.RequeueAfter == 0 {
		t.Error("expected RequeueAfter to be set even when disabled")
	}
	if sf.fetched {
		t.Error("fetcher should not have been called while disabled")
	}
	if cf.called {
		t.Error("client factory should not have been called while disabled")
	}
	if mcExists(t, c, "99-aro-stale") {
		t.Error("stale managed MachineConfig was not cleaned up")
	}
	if !mcExists(t, c, "99-someone-else") {
		t.Error("foreign MachineConfig was incorrectly deleted")
	}
}

func TestReconcileNoURICleansUp(t *testing.T) {
	stale := staleMC("99-aro-stale")
	aro := fixtureAROCluster(arov1alpha1.OperatorFlags{
		operator.DynamicWorkaroundCatalogEnabled:   operator.FlagTrue,
		operator.DynamicWorkaroundCatalogSecretURI: "",
	})
	c := ctrlfake.NewClientBuilder().WithObjects(aro, stale).Build()
	sf := &stubFetcher{}
	cf := &stubClientFactory{}
	r := newReconciler(c, sf, cf.factory())

	if _, err := r.Reconcile(context.Background(), reconcile.Request{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sf.fetched {
		t.Error("fetcher should not have been called with empty URI")
	}
	if cf.called {
		t.Error("client factory should not have been called with empty URI")
	}
	if mcExists(t, c, "99-aro-stale") {
		t.Error("stale managed MachineConfig should have been cleaned up")
	}
}

func TestReconcileMalformedURIDoesNotCleanup(t *testing.T) {
	// A malformed URI is a config error, not a transient failure. The
	// controller must log it and leave existing managed MCs alone.
	existing := staleMC("99-aro-existing")
	aro := fixtureAROCluster(arov1alpha1.OperatorFlags{
		operator.DynamicWorkaroundCatalogEnabled:   operator.FlagTrue,
		operator.DynamicWorkaroundCatalogSecretURI: "http://not-https.example/secrets/x",
	})
	cv := fixtureClusterVersion("4.17.0")
	c := ctrlfake.NewClientBuilder().WithObjects(aro, cv, existing).Build()
	sf := &stubFetcher{}
	cf := &stubClientFactory{}
	r := newReconciler(c, sf, cf.factory())

	if _, err := r.Reconcile(context.Background(), reconcile.Request{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cf.called {
		t.Error("client factory should not have been called for a malformed URI")
	}
	if !mcExists(t, c, "99-aro-existing") {
		t.Fatal("existing managed MC was deleted on URI parse error")
	}
}

func TestReconcileMalformedPredicatesDoesNotCleanup(t *testing.T) {
	// A malformed predicates flag is a config error like a malformed URI:
	// log it and leave existing managed MCs alone. Critical to verify because
	// the predicates flag is operator-edited and easy to typo.
	existing := staleMC("99-aro-existing")
	aro := fixtureAROCluster(arov1alpha1.OperatorFlags{
		operator.DynamicWorkaroundCatalogEnabled:   operator.FlagTrue,
		operator.DynamicWorkaroundCatalogSecretURI: testSecretURI,
		operator.DynamicWorkaroundPredicates:       `{not-json`,
	})
	cv := fixtureClusterVersion("4.17.0")
	c := ctrlfake.NewClientBuilder().WithObjects(aro, cv, existing).Build()
	sf := &stubFetcher{}
	cf := &stubClientFactory{}
	r := newReconciler(c, sf, cf.factory())

	if _, err := r.Reconcile(context.Background(), reconcile.Request{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sf.fetched {
		t.Error("fetcher should not have been called with malformed predicates")
	}
	if !mcExists(t, c, "99-aro-existing") {
		t.Fatal("existing managed MC was deleted on predicates parse error")
	}
}

func TestReconcileAppliesMatchingWorkaround(t *testing.T) {
	aro := fixtureAROCluster(arov1alpha1.OperatorFlags{
		operator.DynamicWorkaroundCatalogEnabled:   operator.FlagTrue,
		operator.DynamicWorkaroundCatalogSecretURI: testSecretURI,
		// Predicate references region + cluster version to verify facts
		// were gathered correctly and threaded into the CEL eval.
		operator.DynamicWorkaroundPredicates: `{"test-wa":"versionAtLeast(clusterVersion, \"4.16.0\") && region == \"eastus\""}`,
	})
	cv := fixtureClusterVersion("4.17.0")
	c := ctrlfake.NewClientBuilder().WithObjects(aro, cv).Build()

	sf := &stubFetcher{catalog: catalogWith("test-wa", "99-aro-test")}
	cf := &stubClientFactory{}
	r := newReconciler(c, sf, cf.factory())

	if _, err := r.Reconcile(context.Background(), reconcile.Request{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !sf.fetched {
		t.Fatal("fetcher was not called")
	}
	if !cf.called {
		t.Fatal("client factory was not called")
	}
	// Verify URI parts were threaded through correctly.
	if cf.gotVaultURL != "https://aro-test-vault.vault.azure.net" {
		t.Errorf("vault URL = %q, want https://aro-test-vault.vault.azure.net", cf.gotVaultURL)
	}
	if sf.gotName != "dynamic-workaround-catalog" {
		t.Errorf("secret name = %q, want dynamic-workaround-catalog", sf.gotName)
	}
	if sf.gotVersion != "" {
		t.Errorf("secret version = %q, want empty (latest)", sf.gotVersion)
	}
	if !mcExists(t, c, "99-aro-test") {
		t.Fatal("expected MachineConfig 99-aro-test to be applied")
	}

	// Verify the catalog version annotation made it onto the MC. This is the
	// breadcrumb we promised support engineers; treat it as part of the API.
	mc := &mcv1.MachineConfig{}
	if err := c.Get(context.Background(), types.NamespacedName{Name: "99-aro-test"}, mc); err != nil {
		t.Fatalf("get applied MC: %v", err)
	}
	if mc.Annotations["aro.openshift.io/dynamic-workaround-catalog-version"] != "test-1" {
		t.Errorf("catalog-version annotation = %q, want test-1",
			mc.Annotations["aro.openshift.io/dynamic-workaround-catalog-version"])
	}
	if mc.Labels[CatalogManagedByLabel] != "true" {
		t.Errorf("managed-by label = %q, want true", mc.Labels[CatalogManagedByLabel])
	}
}

func TestReconcilePinnedVersion(t *testing.T) {
	// A URI with an explicit version segment should pin the GetSecret call
	// to that version. This is the production "rollback" workflow.
	pinnedURI := "https://aro-test-vault.vault.azure.net/secrets/dynamic-workaround-catalog/v123abc"
	aro := fixtureAROCluster(arov1alpha1.OperatorFlags{
		operator.DynamicWorkaroundCatalogEnabled:   operator.FlagTrue,
		operator.DynamicWorkaroundCatalogSecretURI: pinnedURI,
		operator.DynamicWorkaroundPredicates:       predicateAlways,
	})
	cv := fixtureClusterVersion("4.17.0")
	c := ctrlfake.NewClientBuilder().WithObjects(aro, cv).Build()

	sf := &stubFetcher{catalog: catalogWith("test-wa", "99-aro-test")}
	cf := &stubClientFactory{}
	r := newReconciler(c, sf, cf.factory())

	if _, err := r.Reconcile(context.Background(), reconcile.Request{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sf.gotVersion != "v123abc" {
		t.Errorf("secret version = %q, want v123abc", sf.gotVersion)
	}
}

func TestReconcileSkipsNonMatchingPredicate(t *testing.T) {
	aro := fixtureAROCluster(arov1alpha1.OperatorFlags{
		operator.DynamicWorkaroundCatalogEnabled:   operator.FlagTrue,
		operator.DynamicWorkaroundCatalogSecretURI: testSecretURI,
		// Predicate requires a region we don't have.
		operator.DynamicWorkaroundPredicates: `{"test-wa":"region == \"westus\""}`,
	})
	cv := fixtureClusterVersion("4.17.0")
	c := ctrlfake.NewClientBuilder().WithObjects(aro, cv).Build()

	sf := &stubFetcher{catalog: catalogWith("test-wa", "99-aro-test")}
	cf := &stubClientFactory{}
	r := newReconciler(c, sf, cf.factory())

	if _, err := r.Reconcile(context.Background(), reconcile.Request{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mcExists(t, c, "99-aro-test") {
		t.Fatal("MC was applied despite predicate mismatch")
	}
}

func TestReconcileSkipsWorkaroundWithoutPredicate(t *testing.T) {
	// Workaround in the catalog but NOT in the predicates flag → cluster
	// has not opted in → no MachineConfig applied. This is the default
	// posture for fresh clusters that have not yet had a predicate set.
	aro := fixtureAROCluster(arov1alpha1.OperatorFlags{
		operator.DynamicWorkaroundCatalogEnabled:   operator.FlagTrue,
		operator.DynamicWorkaroundCatalogSecretURI: testSecretURI,
		operator.DynamicWorkaroundPredicates:       `{"other-wa":"true"}`,
	})
	cv := fixtureClusterVersion("4.17.0")
	c := ctrlfake.NewClientBuilder().WithObjects(aro, cv).Build()

	sf := &stubFetcher{catalog: catalogWith("test-wa", "99-aro-test")}
	cf := &stubClientFactory{}
	r := newReconciler(c, sf, cf.factory())

	if _, err := r.Reconcile(context.Background(), reconcile.Request{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mcExists(t, c, "99-aro-test") {
		t.Fatal("MC was applied for a workaround with no per-cluster predicate")
	}
}

func TestReconcileCleansUpStaleEntries(t *testing.T) {
	// Pre-existing managed MC ("99-old") should be deleted because the
	// current catalog only produces "99-current".
	old := staleMC("99-old")
	aro := fixtureAROCluster(arov1alpha1.OperatorFlags{
		operator.DynamicWorkaroundCatalogEnabled:   operator.FlagTrue,
		operator.DynamicWorkaroundCatalogSecretURI: testSecretURI,
		operator.DynamicWorkaroundPredicates:       `{"current":"true"}`,
	})
	cv := fixtureClusterVersion("4.17.0")
	c := ctrlfake.NewClientBuilder().WithObjects(aro, cv, old).Build()

	sf := &stubFetcher{catalog: catalogWith("current", "99-current")}
	cf := &stubClientFactory{}
	r := newReconciler(c, sf, cf.factory())

	if _, err := r.Reconcile(context.Background(), reconcile.Request{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mcExists(t, c, "99-old") {
		t.Error("stale MC was not cleaned up")
	}
	if !mcExists(t, c, "99-current") {
		t.Error("new MC was not applied")
	}
}

func TestReconcileFetchFailureKeepsExistingMCs(t *testing.T) {
	// Critical safety property: a Key Vault outage must NOT cause existing
	// mitigations to be torn down. Pre-seed a managed MC, then force the
	// fetcher to fail and assert the MC survives.
	existing := staleMC("99-aro-existing")
	aro := fixtureAROCluster(arov1alpha1.OperatorFlags{
		operator.DynamicWorkaroundCatalogEnabled:   operator.FlagTrue,
		operator.DynamicWorkaroundCatalogSecretURI: testSecretURI,
		operator.DynamicWorkaroundPredicates:       predicateAlways,
	})
	cv := fixtureClusterVersion("4.17.0")
	c := ctrlfake.NewClientBuilder().WithObjects(aro, cv, existing).Build()

	sf := &stubFetcher{err: errors.New("simulated outage")}
	cf := &stubClientFactory{}
	r := newReconciler(c, sf, cf.factory())

	res, err := r.Reconcile(context.Background(), reconcile.Request{})
	if err != nil {
		t.Fatalf("expected fetch error to be swallowed, got %v", err)
	}
	if res.RequeueAfter == 0 {
		t.Error("expected RequeueAfter even on fetch failure")
	}
	if !mcExists(t, c, "99-aro-existing") {
		t.Fatal("existing managed MC was deleted on fetch failure — this would tear down mitigations during a Key Vault outage")
	}
}

func TestReconcileClientFactoryFailureKeepsExistingMCs(t *testing.T) {
	// Mirror of the above for the credential / KV-client construction path:
	// if we can't even build the client (e.g. cred refresh failure), we
	// must not delete what is already on the cluster.
	existing := staleMC("99-aro-existing")
	aro := fixtureAROCluster(arov1alpha1.OperatorFlags{
		operator.DynamicWorkaroundCatalogEnabled:   operator.FlagTrue,
		operator.DynamicWorkaroundCatalogSecretURI: testSecretURI,
		operator.DynamicWorkaroundPredicates:       predicateAlways,
	})
	cv := fixtureClusterVersion("4.17.0")
	c := ctrlfake.NewClientBuilder().WithObjects(aro, cv, existing).Build()

	sf := &stubFetcher{}
	cf := &stubClientFactory{returnErr: errors.New("creds expired")}
	r := newReconciler(c, sf, cf.factory())

	if _, err := r.Reconcile(context.Background(), reconcile.Request{}); err != nil {
		t.Fatalf("expected client-factory error to be swallowed, got %v", err)
	}
	if sf.fetched {
		t.Error("fetcher should not have been called when client factory failed")
	}
	if !mcExists(t, c, "99-aro-existing") {
		t.Fatal("existing managed MC was deleted on client-factory failure")
	}
}

func TestParsePollInterval(t *testing.T) {
	tests := []struct {
		raw  string
		want time.Duration
	}{
		{"", defaultPollInterval},
		{"garbage", defaultPollInterval},
		{"0s", defaultPollInterval},
		{"-1m", defaultPollInterval},
		{"10s", minPollInterval}, // clamped up
		{"30s", minPollInterval}, // clamped up
		{"1m", minPollInterval},  // boundary, kept
		{"5m", 5 * time.Minute},  // typical
		{"1h", time.Hour},        // long
	}
	for _, tt := range tests {
		got := parsePollInterval(tt.raw)
		if got != tt.want {
			t.Errorf("parsePollInterval(%q) = %s, want %s", tt.raw, got, tt.want)
		}
	}
}
