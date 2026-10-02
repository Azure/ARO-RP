package encryption

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/util/azureclient/azuresdk/azsecrets"
)

const (
	// defaultMinRefreshInterval bounds how often a failure to open can cause
	// the keys to be re-enumerated. A document that no key can open is not
	// necessarily rare, so this must be long enough that a stream of them
	// cannot become a stream of Key Vault requests.
	defaultMinRefreshInterval = 5 * time.Minute

	// refreshTimeout bounds a single re-enumeration. Open has no context of its
	// own, and its callers should not be made to wait indefinitely on Key
	// Vault.
	refreshTimeout = 30 * time.Second
)

var errNoOpeners = errors.New("no decryption keys are available")

// A keySet is the keys a multi holds at a point in time. It is replaced
// wholesale rather than mutated, so that Open needs no lock.
type keySet struct {
	sealer  AEAD
	openers []AEAD
}

// A keyLoader enumerates the encryption keys currently in force.
type keyLoader interface {
	load(ctx context.Context) (*keySet, error)
}

type multi struct {
	loader keyLoader

	log                *logrus.Entry
	minRefreshInterval time.Duration
	now                func() time.Time

	keys atomic.Pointer[keySet]

	refreshMu     sync.Mutex
	lastRefreshed time.Time // guarded by refreshMu
}

var _ AEAD = (*multi)(nil)

// An Option configures optional behaviour of an AEAD returned by NewMulti.
type Option func(*multi)

// WithLogger sets the logger on which key refreshes are reported. Refreshes are
// silent by default.
func WithLogger(log *logrus.Entry) Option {
	return func(m *multi) { m.log = log }
}

// WithMinRefreshInterval overrides how often a failure to open may cause the
// keys to be re-enumerated. Non-positive durations are ignored, since they
// would let every failed open reach Key Vault, and the failures this exists to
// answer arrive in bursts.
func WithMinRefreshInterval(d time.Duration) Option {
	return func(m *multi) {
		if d > 0 {
			m.minRefreshInterval = d
		}
	}
}

func NewMulti(ctx context.Context, serviceKeyvault azsecrets.Client, secretName, legacySecretName string, opts ...Option) (AEAD, error) {
	return newMulti(ctx, &keyVaultLoader{
		keyvault:         serviceKeyvault,
		secretName:       secretName,
		legacySecretName: legacySecretName,
	}, opts...)
}

func newMulti(ctx context.Context, loader keyLoader, opts ...Option) (*multi, error) {
	m := &multi{
		loader:             loader,
		minRefreshInterval: defaultMinRefreshInterval,
		now:                time.Now,
	}

	for _, opt := range opts {
		opt(m)
	}

	keys, err := m.loader.load(ctx)
	if err != nil {
		return nil, err
	}
	if keys == nil {
		return nil, errNoOpeners
	}

	m.keys.Store(keys)

	// lastRefreshed is deliberately left at its zero value rather than set to
	// now. Recording the construction load here would rate-limit the first
	// failure-triggered refresh for minRefreshInterval, and a process which
	// starts just before a new key version is written would then be unable to
	// recover for that whole window — which is the case this type exists for.
	// The cost of leaving it zero is one extra enumeration if the first
	// document opened is genuinely undecryptable.

	return m, nil
}

func (c *multi) Open(input []byte) ([]byte, error) {
	keys := c.keys.Load()

	b, err := open(keys, input)
	if err == nil {
		return b, nil
	}

	// A failure to open may mean nothing worse than that this process's keys
	// are stale. The services which hold a multi are long-lived and enumerate
	// the secret versions once, at start-up, so a version created since then is
	// absent until the process is restarted. Re-enumerate and try once more.
	c.refresh()

	// Re-read the keys rather than asking whether this call was the one that
	// replaced them. A refresh performed by a concurrent caller is just as
	// useful as one performed here, and under the burst of failures that a
	// rotation produces most callers will find the keys already replaced by the
	// time they reach this point. Comparing the pointer, rather than trusting
	// that a refresh happened, also avoids a pointless second attempt when the
	// rate limit meant no re-enumeration took place at all.
	refreshed := c.keys.Load()
	if refreshed == keys {
		return nil, err
	}

	b, refreshedErr := open(refreshed, input)
	if refreshedErr != nil {
		// Report the original error. The input cannot be opened by any key this
		// process holds, and describing that in terms of the first attempt is
		// the more faithful account.
		return nil, err
	}

	return b, nil
}

func open(keys *keySet, input []byte) ([]byte, error) {
	if keys == nil || len(keys.openers) == 0 {
		return nil, errNoOpeners
	}

	var err error
	for _, opener := range keys.openers {
		var b []byte
		if b, err = opener.Open(input); err == nil {
			return b, nil
		}
	}

	return nil, err
}

// refresh re-enumerates the keys and replaces them. It does nothing if it has
// already run within the last minRefreshInterval.
//
// It reports nothing. Callers observe the outcome by re-reading c.keys, so that
// a caller which arrives while another is already refreshing benefits from that
// refresh rather than being told its own attempt did nothing.
//
// The lock is held across the load, so a caller arriving mid-refresh waits for
// it. That wait is bounded by refreshTimeout and is the point of the exercise:
// the keys it is waiting for are the ones that will let it succeed. Returning
// early instead would hand it a stale key set and a failure it need not have
// had.
func (c *multi) refresh() {
	c.refreshMu.Lock()
	defer c.refreshMu.Unlock()

	// Another caller may have refreshed while this one waited for the lock.
	if c.now().Sub(c.lastRefreshed) < c.minRefreshInterval {
		return
	}

	// Recorded before the attempt rather than after it, so that a Key Vault
	// which is slow or failing is not asked again any sooner than one which
	// answered.
	c.lastRefreshed = c.now()

	ctx, cancel := context.WithTimeout(context.Background(), refreshTimeout)
	defer cancel()

	keys, err := c.loader.load(ctx)
	if err == nil && keys == nil {
		err = errNoOpeners
	}
	if err != nil {
		if c.log != nil {
			c.log.Warnf("failed refreshing encryption keys: %v", err)
		}
		return
	}

	was := len(c.keys.Load().openers)
	c.keys.Store(keys)

	if c.log != nil && len(keys.openers) != was {
		c.log.Infof("refreshed encryption keys: %d openers, was %d", len(keys.openers), was)
	}
}

func (c *multi) Seal(input []byte) ([]byte, error) {
	return c.keys.Load().sealer.Seal(input)
}

// A keyVaultLoader enumerates the encryption keys from Key Vault.
type keyVaultLoader struct {
	keyvault         azsecrets.Client
	secretName       string
	legacySecretName string
}

var _ keyLoader = (*keyVaultLoader)(nil)

// load reads the encryption secrets from Key Vault.
//
// The sealer is the current version of secretName. The openers are every
// version of both secretName and legacySecretName, so that data written by a
// process holding a different version can still be read.
func (l *keyVaultLoader) load(ctx context.Context) (*keySet, error) {
	rawKey, err := l.keyvault.GetSecret(ctx, l.secretName, "", nil)
	if err != nil {
		return nil, err
	}

	key, err := azsecrets.ExtractBase64Value(rawKey)
	if err != nil {
		return nil, err
	}

	sealer, err := NewAES256SHA512(ctx, key)
	if err != nil {
		return nil, err
	}

	keys := &keySet{sealer: sealer}

	for _, x := range []struct {
		secretName  string
		aeadFactory func(context.Context, []byte) (AEAD, error)
	}{
		{l.secretName, NewAES256SHA512},
		{l.legacySecretName, NewXChaCha20Poly1305},
	} {
		openers, err := l.openersFor(ctx, x.secretName, x.aeadFactory)
		if err != nil {
			return nil, err
		}

		keys.openers = append(keys.openers, openers...)
	}

	return keys, nil
}

// openersFor builds one AEAD per version of the named secret.
func (l *keyVaultLoader) openersFor(ctx context.Context, secretName string, aeadFactory func(context.Context, []byte) (AEAD, error)) ([]AEAD, error) {
	var keys [][]byte

	pager := l.keyvault.NewListSecretPropertiesVersionsPager(secretName, nil)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, properties := range page.Value {
			if properties != nil && properties.ID != nil && properties.Attributes != nil {
				raw, err := l.keyvault.GetSecret(ctx, (*properties.ID).Name(), (*properties.ID).Version(), nil)
				if err != nil {
					return nil, err
				}

				version, err := azsecrets.ExtractBase64Value(raw)
				if err != nil {
					return nil, err
				}

				keys = append(keys, version)
			}
		}
	}

	openers := make([]AEAD, 0, len(keys))
	for _, key := range keys {
		aead, err := aeadFactory(ctx, key)
		if err != nil {
			return nil, err
		}

		openers = append(openers, aead)
	}

	return openers, nil
}
