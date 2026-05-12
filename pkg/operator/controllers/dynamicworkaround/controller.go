package dynamicworkaround

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"

	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	configv1 "github.com/openshift/api/config/v1"
	mcv1 "github.com/openshift/machine-config-operator/pkg/apis/machineconfiguration.openshift.io/v1"

	"github.com/Azure/ARO-RP/pkg/operator"
	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	"github.com/Azure/ARO-RP/pkg/operator/predicates"
	"github.com/Azure/ARO-RP/pkg/util/azureclient"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/azuresdk/azsecrets"
	"github.com/Azure/ARO-RP/pkg/util/clienthelper"
	"github.com/Azure/ARO-RP/pkg/util/version"
)

const (
	// defaultPollInterval is the requeue cadence used when the operator flag
	// is missing or unparseable. 5 minutes balances responsiveness against
	// load on the Key Vault endpoint.
	defaultPollInterval = 5 * time.Minute

	// minPollInterval clamps absurdly low values an operator might set by
	// mistake, e.g. "10s". A faster-than-1-minute poll is never useful for
	// MachineConfig-class fixes (MCO reconciles much slower than that).
	minPollInterval = 1 * time.Minute

	// kvCallTimeout caps how long a single Key Vault GetSecret may take.
	// The azsecrets SDK has its own retry policy on top of this.
	kvCallTimeout = 30 * time.Second
)

// SecretsClientFactory builds an azsecrets.Client for a given Key Vault URL,
// using the AROEnvironment to derive cloud-specific endpoints and credential
// options. Injectable so tests can hand in a stub that returns a mock client
// without touching real Azure credentials.
type SecretsClientFactory func(env *azureclient.AROEnvironment, vaultURL string) (azsecrets.Client, error)

// NewDefaultSecretsClientFactory returns the production factory. It uses
// azidentity.NewDefaultAzureCredential, which transparently picks up the
// AZURE_CLIENT_ID / AZURE_CLIENT_SECRET / AZURE_TENANT_ID env vars that the
// master operator pod already gets from the azure-cloud-credentials Secret
// (see pkg/operator/deploy/staticresources/master/deployment.yaml.tmpl), or
// the workload-identity federated token when in MIWI mode.
func NewDefaultSecretsClientFactory() SecretsClientFactory {
	return func(env *azureclient.AROEnvironment, vaultURL string) (azsecrets.Client, error) {
		cred, err := azidentity.NewDefaultAzureCredential(env.DefaultAzureCredentialOptions())
		if err != nil {
			return nil, fmt.Errorf("build Azure credential: %w", err)
		}
		c, err := azsecrets.NewClient(vaultURL, cred, env.ArmClientOptions().ClientOptions)
		if err != nil {
			return nil, fmt.Errorf("build Key Vault client for %s: %w", vaultURL, err)
		}
		return c, nil
	}
}

// Reconciler watches the ARO Cluster CR, periodically reads the dynamic
// workaround catalog from a Key Vault secret, and reconciles the set of
// catalog-managed MachineConfigs to match.
//
// Lifecycle of a workaround:
//
//	catalog entry appears → predicate matches → MachineConfig is created via Ensure
//	catalog entry persists → MachineConfig is re-Ensured (idempotent)
//	catalog entry disappears OR predicate stops matching → MachineConfig is deleted
//	Enabled flag goes false → every catalog-managed MachineConfig is deleted
type Reconciler struct {
	log           *logrus.Entry
	client        client.Client
	ch            clienthelper.Interface
	fetcher       Fetcher
	clientFactory SecretsClientFactory
}

// NewReconciler wires up the controller. fetcher and clientFactory are
// injectable to keep tests hermetic; production code should pass
// NewKeyVaultFetcher() and NewDefaultSecretsClientFactory().
func NewReconciler(log *logrus.Entry, c client.Client, fetcher Fetcher, factory SecretsClientFactory) *Reconciler {
	return &Reconciler{
		log:           log,
		client:        c,
		ch:            clienthelper.NewWithClient(log, c),
		fetcher:       fetcher,
		clientFactory: factory,
	}
}

// Reconcile is invoked on changes to the ARO Cluster CR and on a poll timer.
// It is intentionally tolerant of transient failures: any error during fetch
// or predicate evaluation is logged and the reconcile is re-queued. We do not
// return an error from Reconcile in those cases because we don't want the
// controller-runtime backoff to mask the steady-state poll interval.
func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	instance := &arov1alpha1.Cluster{}
	if err := r.client.Get(ctx, types.NamespacedName{Name: arov1alpha1.SingletonClusterName}, instance); err != nil {
		return reconcile.Result{}, err
	}

	flags := instance.Spec.OperatorFlags
	pollAfter := parsePollInterval(flags.GetWithDefault(operator.DynamicWorkaroundCatalogPollInterval, ""))

	if !flags.GetSimpleBoolean(operator.DynamicWorkaroundCatalogEnabled) {
		r.log.Debug("dynamic workaround catalog disabled; ensuring no catalog-managed MachineConfigs remain")
		if err := r.cleanupAll(ctx, nil); err != nil {
			r.log.Errorf("cleanup failed while disabled: %v", err)
		}
		return reconcile.Result{RequeueAfter: pollAfter}, nil
	}

	secretURI := flags.GetWithDefault(operator.DynamicWorkaroundCatalogSecretURI, "")
	if secretURI == "" {
		r.log.Warn("catalog enabled but no secret URI set; treating as disabled")
		if err := r.cleanupAll(ctx, nil); err != nil {
			r.log.Errorf("cleanup failed with empty secret URI: %v", err)
		}
		return reconcile.Result{RequeueAfter: pollAfter}, nil
	}

	// Parse per-cluster predicates BEFORE doing any network work. A malformed
	// predicates flag is a config error and must not be allowed to tear down
	// existing managed MCs — log loudly and skip this cycle.
	preds, err := parsePredicates(flags.GetWithDefault(operator.DynamicWorkaroundPredicates, ""))
	if err != nil {
		r.log.Errorf("invalid predicates flag: %v", err)
		return reconcile.Result{RequeueAfter: pollAfter}, nil
	}

	vaultURL, secretName, secretVersion, err := parseSecretURI(secretURI)
	if err != nil {
		// A malformed URI is a config error, not a transient failure — log
		// loudly but don't tear down existing mitigations.
		r.log.Errorf("invalid catalog secret URI %q: %v", secretURI, err)
		return reconcile.Result{RequeueAfter: pollAfter}, nil
	}

	facts, err := r.gatherClusterFacts(ctx, instance)
	if err != nil {
		r.log.Errorf("gather cluster facts: %v", err)
		// Without facts we can't safely evaluate predicates. Skip applying
		// anything this cycle but DON'T cleanup — that would flap on a
		// transient Network/ClusterVersion read error.
		return reconcile.Result{RequeueAfter: pollAfter}, nil
	}

	azEnv, err := azureclient.EnvironmentFromName(instance.Spec.AZEnvironment)
	if err != nil {
		r.log.Errorf("resolve AZ environment %q: %v", instance.Spec.AZEnvironment, err)
		return reconcile.Result{RequeueAfter: pollAfter}, nil
	}

	secretsClient, err := r.clientFactory(&azEnv, vaultURL)
	if err != nil {
		r.log.Errorf("build Key Vault client: %v", err)
		return reconcile.Result{RequeueAfter: pollAfter}, nil
	}

	fetchCtx, cancel := context.WithTimeout(ctx, kvCallTimeout)
	defer cancel()
	catalog, err := r.fetcher.Fetch(fetchCtx, secretsClient, secretName, secretVersion)
	if err != nil {
		r.log.Errorf("fetch catalog from %s: %v", secretURI, err)
		// Don't tear down working mitigations because the catalog endpoint is
		// briefly unreachable.
		return reconcile.Result{RequeueAfter: pollAfter}, nil
	}

	r.log.WithField("catalogVersion", catalog.CatalogVersion).
		Infof("evaluating %d workarounds (predicates configured for %d)", len(catalog.Workarounds), len(preds))

	applied, err := r.applyMatching(ctx, catalog, preds, facts)
	if err != nil {
		r.log.Errorf("apply workarounds: %v", err)
	}

	if err := r.cleanupAll(ctx, applied); err != nil {
		r.log.Errorf("cleanup stale workarounds: %v", err)
	}

	return reconcile.Result{RequeueAfter: pollAfter}, nil
}

// applyMatching iterates the catalog and Ensures each workaround whose
// per-cluster predicate (looked up by workaround Name in preds) matches.
// Returns the set of MachineConfig names that should remain on the cluster
// after this pass so the caller can delete the rest.
//
// A workaround that has no entry in preds is treated as "not opted in on this
// cluster" and skipped silently. A single workaround's failure does not stop
// the others — we record the failure and continue, on the theory that a bad
// ignition body or runtime CEL error in one entry should not block a
// security-critical entry later in the list.
func (r *Reconciler) applyMatching(ctx context.Context, catalog *Catalog, preds Predicates, facts ClusterFacts) (map[string]struct{}, error) {
	applied := make(map[string]struct{}, len(catalog.Workarounds))
	var firstErr error

	for i := range catalog.Workarounds {
		w := &catalog.Workarounds[i]
		log := r.log.WithField("workaround", w.Name)

		match, hasPredicate, err := preds.Eval(ctx, w.Name, facts)
		if err != nil {
			log.Errorf("predicate evaluation failed (skipping): %v", err)
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		if !hasPredicate {
			log.Debug("no predicate configured for this workaround on this cluster; skipping")
			continue
		}
		if !match {
			log.Debug("predicate evaluated false; skipping")
			continue
		}

		mc := machineConfigFromCatalog(w, catalog.CatalogVersion)
		if err := r.ch.Ensure(ctx, mc); err != nil {
			log.Errorf("ensure MachineConfig %q: %v", mc.Name, err)
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		log.Infof("applied MachineConfig %q (catalog %s)", mc.Name, catalog.CatalogVersion)
		applied[mc.Name] = struct{}{}
	}
	return applied, firstErr
}

// cleanupAll deletes every MachineConfig that carries our managed-by label
// but whose name is not in `keep`. Passing keep == nil deletes everything
// (used for the disabled / no-URI paths).
//
// We list via an unstructured list rather than the typed mcv1 list to avoid
// pulling the full MachineConfig schema for what is just a label query; this
// also keeps the operator happy on clusters whose MCO CRDs are momentarily
// unavailable.
func (r *Reconciler) cleanupAll(ctx context.Context, keep map[string]struct{}) error {
	list := &unstructured.UnstructuredList{}
	list.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   mcv1.GroupVersion.Group,
		Version: mcv1.GroupVersion.Version,
		Kind:    "MachineConfigList",
	})

	if err := r.client.List(ctx, list, client.MatchingLabels{CatalogManagedByLabel: "true"}); err != nil {
		// On a brand-new cluster the MCO CRD may not yet be installed.
		// Don't promote that to an error.
		if kerrors.IsNotFound(err) {
			return nil
		}
		return fmt.Errorf("list catalog-managed MachineConfigs: %w", err)
	}

	for i := range list.Items {
		name := list.Items[i].GetName()
		if _, want := keep[name]; want {
			continue
		}
		err := r.ch.EnsureDeleted(ctx,
			mcv1.GroupVersion.WithKind("MachineConfig"),
			types.NamespacedName{Name: name},
		)
		if err != nil {
			r.log.Errorf("delete stale MachineConfig %q: %v", name, err)
			// keep going; one stale delete failure should not stop the others
			continue
		}
		r.log.Infof("removed stale catalog-managed MachineConfig %q", name)
	}
	return nil
}

// gatherClusterFacts reads everything the predicate evaluator needs. Returns
// an error only when the *cluster* lookups themselves fail (Network being
// absent is fine — the predicate handles the empty-mode case).
func (r *Reconciler) gatherClusterFacts(ctx context.Context, instance *arov1alpha1.Cluster) (ClusterFacts, error) {
	facts := ClusterFacts{
		Location:            instance.Spec.Location,
		ArchitectureVersion: instance.Spec.ArchitectureVersion,
	}

	cv := &configv1.ClusterVersion{}
	if err := r.client.Get(ctx, types.NamespacedName{Name: "version"}, cv); err != nil {
		return facts, fmt.Errorf("get ClusterVersion: %w", err)
	}
	v, err := version.GetClusterVersion(cv)
	if err != nil {
		// Unknown cluster version is recoverable: predicate matches that need
		// a version return no-match, predicate matches that don't are unaffected.
		r.log.Warnf("could not determine cluster version: %v", err)
	} else {
		facts.ClusterVersion = v
	}

	// IPSec mode lives on the operator Network CR. Missing or malformed:
	// keep facts.IPSecMode as empty string, which the "absent" sentinel matches.
	network := &unstructured.Unstructured{}
	network.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "operator.openshift.io",
		Version: "v1",
		Kind:    "Network",
	})
	if err := r.client.Get(ctx, types.NamespacedName{Name: "cluster"}, network); err != nil {
		if !kerrors.IsNotFound(err) {
			return facts, fmt.Errorf("get Network: %w", err)
		}
	} else {
		mode, found, err := unstructured.NestedString(
			network.Object,
			"spec", "defaultNetwork", "ovnKubernetesConfig", "ipsecConfig", "mode",
		)
		if err != nil {
			r.log.Warnf("could not parse ipsecConfig.mode: %v", err)
		} else if found {
			facts.IPSecMode = mode
		}
	}

	return facts, nil
}

// parsePollInterval reads the configured poll interval, clamping it to
// minPollInterval and falling back to defaultPollInterval on parse failure
// or empty input. Centralised so tests can exercise the boundary cases.
func parsePollInterval(raw string) time.Duration {
	if raw == "" {
		return defaultPollInterval
	}
	d, err := time.ParseDuration(raw)
	if err != nil || d <= 0 {
		return defaultPollInterval
	}
	if d < minPollInterval {
		return minPollInterval
	}
	return d
}

// SetupWithManager registers the reconciler with controller-runtime. We watch
// the ARO Cluster CR for flag changes and re-queue ourselves on a timer via
// RequeueAfter — there's no benefit to also watching MachineConfig because
// the controller is the only writer of catalog-managed MCs.
func (r *Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&arov1alpha1.Cluster{}, builder.WithPredicates(predicate.And(predicates.AROCluster, predicate.GenerationChangedPredicate{}))).
		Named(ControllerName).
		Complete(r)
}
