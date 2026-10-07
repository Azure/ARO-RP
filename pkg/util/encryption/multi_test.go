package encryption

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"reflect"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"go.uber.org/mock/gomock"

	mock_encryption "github.com/Azure/ARO-RP/pkg/util/mocks/encryption"
	utilerror "github.com/Azure/ARO-RP/test/util/error"
)

// loaderFunc adapts a function to the keyLoader interface.
type loaderFunc func(ctx context.Context) (*keySet, error)

func (f loaderFunc) load(ctx context.Context) (*keySet, error) {
	return f(ctx)
}

// newTestMulti returns a multi holding the given openers, with refreshes rate
// limited out, so that a failure to open does not reach the loader.
func newTestMulti(t *testing.T, openers ...AEAD) *multi {
	t.Helper()

	now := time.Unix(0, 0)

	m := &multi{
		loader: loaderFunc(func(context.Context) (*keySet, error) {
			t.Error("keyLoader.load() called, want no call")
			return nil, errors.New("unexpected refresh")
		}),
		minRefreshInterval: time.Hour,
		now:                func() time.Time { return now },
	}
	m.keys.Store(&keySet{openers: openers})
	m.lastRefreshed = now

	return m
}

func TestOpen(t *testing.T) {
	mockInput := []byte("fakeInput")

	type test struct {
		name       string
		mocks      func(firstOpener *mock_encryption.MockAEAD, secondOpener *mock_encryption.MockAEAD)
		wantResult []byte
		wantErr    string
	}

	for _, tt := range []*test{
		{
			name: "first opener succeeds, do not try second",
			mocks: func(firstOpener *mock_encryption.MockAEAD, secondOpener *mock_encryption.MockAEAD) {
				firstOpener.EXPECT().Open(mockInput).Return([]byte("result from the first opener"), nil)
			},
			wantResult: []byte("result from the first opener"),
		},
		{
			name: "first opener errors, but second succeeds",
			mocks: func(firstOpener *mock_encryption.MockAEAD, secondOpener *mock_encryption.MockAEAD) {
				firstOpener.EXPECT().Open(mockInput).Return(nil, errors.New("fake error from the first opener"))
				secondOpener.EXPECT().Open(mockInput).Return([]byte("result from the second opener"), nil)
			},
			wantResult: []byte("result from the second opener"),
		},
		{
			name: "all openers error",
			mocks: func(firstOpener *mock_encryption.MockAEAD, secondOpener *mock_encryption.MockAEAD) {
				firstOpener.EXPECT().Open(mockInput).Return(nil, errors.New("fake error from the first opener"))
				secondOpener.EXPECT().Open(mockInput).Return(nil, errors.New("fake error from the second opener"))
			},
			wantErr: "fake error from the second opener",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)

			firstOpener := mock_encryption.NewMockAEAD(controller)
			secondOpener := mock_encryption.NewMockAEAD(controller)

			multi := newTestMulti(t, firstOpener, secondOpener)

			tt.mocks(firstOpener, secondOpener)

			b, err := multi.Open(mockInput)
			utilerror.AssertErrorMessage(t, err, tt.wantErr)
			if b != nil && !reflect.DeepEqual(tt.wantResult, b) ||
				b == nil && tt.wantResult != nil {
				t.Error(b)
			}
		})
	}
}

func TestOpenWithNoOpeners(t *testing.T) {
	multi := newTestMulti(t)

	b, err := multi.Open([]byte("fakeInput"))
	if !errors.Is(err, errNoOpeners) {
		t.Errorf("multi.Open() error = %v, want %v", err, errNoOpeners)
	}
	if b != nil {
		t.Errorf("multi.Open() = %v, want nil", b)
	}
}

// TestOpenRefreshesStaleKeys covers the case this refresh exists for: the
// process was started before a key was rotated, so the key which sealed the
// input is not among the ones it enumerated at start-up.
func TestOpenRefreshesStaleKeys(t *testing.T) {
	mockInput := []byte("fakeInput")
	staleErr := fmt.Errorf("%w: fake error from the stale opener", ErrKeyMismatch)

	for _, tt := range []struct {
		name       string
		load       func(fresh AEAD) (*keySet, error)
		mocks      func(stale *mock_encryption.MockAEAD, fresh *mock_encryption.MockAEAD)
		wantResult []byte
		wantErr    string
		wantLoads  int
	}{
		{
			name: "refreshed key opens the input",
			load: func(fresh AEAD) (*keySet, error) {
				return &keySet{openers: []AEAD{fresh}}, nil
			},
			mocks: func(stale *mock_encryption.MockAEAD, fresh *mock_encryption.MockAEAD) {
				stale.EXPECT().Open(mockInput).Return(nil, staleErr)
				fresh.EXPECT().Open(mockInput).Return([]byte("result from the fresh opener"), nil)
			},
			wantResult: []byte("result from the fresh opener"),
			wantLoads:  1,
		},
		{
			name: "refreshed key does not open the input either",
			load: func(fresh AEAD) (*keySet, error) {
				return &keySet{openers: []AEAD{fresh}}, nil
			},
			mocks: func(stale *mock_encryption.MockAEAD, fresh *mock_encryption.MockAEAD) {
				stale.EXPECT().Open(mockInput).Return(nil, staleErr)
				fresh.EXPECT().Open(mockInput).Return(nil, errors.New("fake error from the fresh opener"))
			},
			// The original error is reported, not the one from the retry.
			wantErr:   "encryption: key mismatch: fake error from the stale opener",
			wantLoads: 1,
		},
		{
			name: "refresh fails",
			load: func(fresh AEAD) (*keySet, error) {
				return nil, errors.New("fake error from key vault")
			},
			mocks: func(stale *mock_encryption.MockAEAD, fresh *mock_encryption.MockAEAD) {
				stale.EXPECT().Open(mockInput).Return(nil, staleErr)
			},
			wantErr:   "encryption: key mismatch: fake error from the stale opener",
			wantLoads: 1,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)

			stale := mock_encryption.NewMockAEAD(controller)
			fresh := mock_encryption.NewMockAEAD(controller)
			tt.mocks(stale, fresh)

			loads := 0
			now := time.Unix(0, 0)

			m := &multi{
				loader: loaderFunc(func(context.Context) (*keySet, error) {
					loads++
					return tt.load(fresh)
				}),
				minRefreshInterval: time.Hour,
				now:                func() time.Time { return now },
			}
			m.keys.Store(&keySet{openers: []AEAD{stale}})
			m.lastRefreshed = now.Add(-time.Hour)

			b, err := m.Open(mockInput)
			utilerror.AssertErrorMessage(t, err, tt.wantErr)
			if !reflect.DeepEqual(b, tt.wantResult) {
				t.Errorf("multi.Open(%q) = %q, want %q", mockInput, b, tt.wantResult)
			}
			if loads != tt.wantLoads {
				t.Errorf("keyLoader.load() called %d times, want %d", loads, tt.wantLoads)
			}
		})
	}
}

// TestRefreshIsRateLimited checks that a run of inputs which no key can open
// cannot become a run of Key Vault requests.
func TestRefreshIsRateLimited(t *testing.T) {
	loads := 0
	now := time.Unix(0, 0)

	m := &multi{
		loader: loaderFunc(func(context.Context) (*keySet, error) {
			loads++
			return &keySet{}, nil
		}),
		minRefreshInterval: time.Hour,
		now:                func() time.Time { return now },
	}
	m.keys.Store(&keySet{})
	m.lastRefreshed = now.Add(-time.Hour)

	m.refresh()
	if loads != 1 {
		t.Errorf("keyLoader.load() called %d times after one refresh, want 1", loads)
	}

	m.refresh()
	if loads != 1 {
		t.Errorf("keyLoader.load() called %d times after an immediate second refresh, want 1", loads)
	}

	now = now.Add(time.Hour)

	m.refresh()
	if loads != 2 {
		t.Errorf("keyLoader.load() called %d times after minRefreshInterval elapsed, want 2", loads)
	}
}

// TestOpenRetriesAfterAConcurrentRefresh covers the caller which finds the keys
// already being refreshed by another goroutine. It must still retry against the
// refreshed keys: the rotation this fix exists for produces a burst of
// simultaneous failures, and only one of them performs the re-enumeration.
func TestOpenRetriesAfterAConcurrentRefresh(t *testing.T) {
	const callers = 8

	controller := gomock.NewController(t)

	// The loader is held until every caller has failed against the stale keys.
	// Each has then taken its snapshot of the key set before the refresh
	// replaces it, so each must find the keys replaced by whichever caller
	// performed the load, and all but that one must do so without loading.
	var failed int64
	allFailed := make(chan struct{})

	stale := mock_encryption.NewMockAEAD(controller)
	stale.EXPECT().Open(gomock.Any()).DoAndReturn(func([]byte) ([]byte, error) {
		if atomic.AddInt64(&failed, 1) == callers {
			close(allFailed)
		}
		return nil, fmt.Errorf("%w: chacha20poly1305: message authentication failed", ErrKeyMismatch)
	}).Times(callers)

	fresh := mock_encryption.NewMockAEAD(controller)
	fresh.EXPECT().Open([]byte("sealed")).Return([]byte("opened"), nil).Times(callers)

	now := time.Unix(0, 0)

	release := make(chan struct{})
	var loads int64

	m := &multi{
		loader: loaderFunc(func(context.Context) (*keySet, error) {
			atomic.AddInt64(&loads, 1)
			<-release
			return &keySet{openers: []AEAD{fresh}}, nil
		}),
		minRefreshInterval: time.Hour,
		now:                func() time.Time { return now },
	}
	m.keys.Store(&keySet{openers: []AEAD{stale}})
	m.lastRefreshed = now.Add(-time.Hour)

	var wg sync.WaitGroup
	results := make([][]byte, callers)
	errs := make([]error, callers)

	for i := range callers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			results[i], errs[i] = m.Open([]byte("sealed"))
		}()
	}

	<-allFailed
	close(release)
	wg.Wait()

	for i := range callers {
		if errs[i] != nil {
			t.Errorf("caller %d: multi.Open() returned unexpected error: %v", i, errs[i])
		}
		if !reflect.DeepEqual(results[i], []byte("opened")) {
			t.Errorf("caller %d: multi.Open() = %q, want %q", i, results[i], []byte("opened"))
		}
	}

	if got := atomic.LoadInt64(&loads); got != 1 {
		t.Errorf("keyLoader.load() called %d times, want 1", got)
	}
}

func TestNewMultiReportsLoadFailure(t *testing.T) {
	wantErr := errors.New("fake error from key vault")

	_, err := newMulti(context.Background(), loaderFunc(func(context.Context) (*keySet, error) {
		return nil, wantErr
	}))
	if !errors.Is(err, wantErr) {
		t.Errorf("newMulti() error = %v, want %v", err, wantErr)
	}
}

// A process which starts shortly before a new key version is written must be
// able to recover on its first failure. Recording the construction load as a
// refresh would rate-limit that first attempt for minRefreshInterval, leaving
// the process unable to open anything sealed with the new version for the whole
// window — the case this type exists to answer.
func TestOpenRefreshesOnTheFirstFailureAfterConstruction(t *testing.T) {
	controller := gomock.NewController(t)

	stale := mock_encryption.NewMockAEAD(controller)
	stale.EXPECT().Open(gomock.Any()).Return(nil, fmt.Errorf("%w: chacha20poly1305: message authentication failed", ErrKeyMismatch)).AnyTimes()

	fresh := mock_encryption.NewMockAEAD(controller)
	fresh.EXPECT().Open([]byte("sealed")).Return([]byte("opened"), nil).AnyTimes()

	now := time.Unix(0, 0)
	loads := 0

	m, err := newMulti(t.Context(), loaderFunc(func(context.Context) (*keySet, error) {
		loads++
		if loads == 1 {
			return &keySet{openers: []AEAD{stale}}, nil
		}
		return &keySet{openers: []AEAD{fresh}}, nil
	}), func(m *multi) { m.now = func() time.Time { return now } })
	if err != nil {
		t.Fatal(err)
	}

	// No time has passed since construction, so a refresh recorded there would
	// suppress this one.
	got, err := m.Open([]byte("sealed"))
	if err != nil {
		t.Errorf("multi.Open() returned unexpected error: %v", err)
	}
	if !reflect.DeepEqual(got, []byte("opened")) {
		t.Errorf("multi.Open() = %q, want %q", got, []byte("opened"))
	}
	if loads != 2 {
		t.Errorf("keyLoader.load() called %d times, want 2: construction, then the first failure", loads)
	}
}

func TestWithMinRefreshIntervalIgnoresNonPositiveDurations(t *testing.T) {
	for _, tt := range []struct {
		name string
		give time.Duration
		want time.Duration
	}{
		{name: "zero is ignored", give: 0, want: defaultMinRefreshInterval},
		{name: "negative is ignored", give: -time.Minute, want: defaultMinRefreshInterval},
		{name: "positive is applied", give: time.Minute, want: time.Minute},
	} {
		t.Run(tt.name, func(t *testing.T) {
			m := &multi{minRefreshInterval: defaultMinRefreshInterval}
			WithMinRefreshInterval(tt.give)(m)

			if m.minRefreshInterval != tt.want {
				t.Errorf("minRefreshInterval = %s, want %s", m.minRefreshInterval, tt.want)
			}
		})
	}
}

// Open refreshes only when a different key could open the input: when the
// input failed authentication, or there were no keys to try. Real ciphers are
// used rather than mocks, because the decision rests on each of them wrapping
// its authentication failures in ErrKeyMismatch, and a cipher which stopped
// doing so would otherwise stop refreshes without failing any test.
func TestOpenRefreshesOnlyWhenAKeyCouldOpenTheInput(t *testing.T) {
	newAES := func(b byte) AEAD {
		t.Helper()
		c, err := NewAES256SHA512(t.Context(), bytes.Repeat([]byte{b}, 64))
		if err != nil {
			t.Fatal(err)
		}
		return c
	}
	newChaCha := func(b byte) AEAD {
		t.Helper()
		c, err := NewXChaCha20Poly1305(t.Context(), bytes.Repeat([]byte{b}, 32))
		if err != nil {
			t.Fatal(err)
		}
		return c
	}
	seal := func(c AEAD) []byte {
		t.Helper()
		b, err := c.Seal([]byte("opened"))
		if err != nil {
			t.Fatal(err)
		}
		return b
	}

	staleAES, freshAES := newAES(1), newAES(2)
	staleChaCha, freshChaCha := newChaCha(1), newChaCha(2)

	for _, tt := range []struct {
		name      string
		stale     []AEAD
		fresh     []AEAD
		input     []byte
		wantErr   string
		wantLoads int
	}{
		{
			name:      "a key mismatch under AES refreshes",
			stale:     []AEAD{staleAES},
			fresh:     []AEAD{freshAES},
			input:     seal(freshAES),
			wantLoads: 1,
		},
		{
			// Openers are held AES first and ChaCha last, as Key Vault loads
			// them, so the error which decides is the ChaCha opener's.
			name:      "a key mismatch under ChaCha refreshes",
			stale:     []AEAD{staleAES, staleChaCha},
			fresh:     []AEAD{staleAES, freshChaCha},
			input:     seal(freshChaCha),
			wantLoads: 1,
		},
		{
			name:      "no keys to try refreshes",
			fresh:     []AEAD{freshAES},
			input:     seal(freshAES),
			wantLoads: 1,
		},
		{
			name:      "an input too short for any key does not refresh",
			stale:     []AEAD{staleAES, staleChaCha},
			fresh:     []AEAD{freshAES},
			input:     make([]byte, 16),
			wantErr:   "encrypted value too short",
			wantLoads: 0,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			loads := 0
			now := time.Unix(0, 0)

			m := &multi{
				loader: loaderFunc(func(context.Context) (*keySet, error) {
					loads++
					return &keySet{openers: tt.fresh}, nil
				}),
				minRefreshInterval: time.Hour,
				now:                func() time.Time { return now },
			}
			m.keys.Store(&keySet{openers: tt.stale})

			got, err := m.Open(tt.input)
			utilerror.AssertErrorMessage(t, err, tt.wantErr)
			if tt.wantErr == "" && !reflect.DeepEqual(got, []byte("opened")) {
				t.Errorf("multi.Open() = %q, want %q", got, []byte("opened"))
			}
			if loads != tt.wantLoads {
				t.Errorf("keyLoader.load() called %d times, want %d", loads, tt.wantLoads)
			}
		})
	}
}

// The loader is an interface seam. One which returned no keys and no error
// would otherwise leave Open dereferencing a nil keySet.
func TestOpenWithoutAKeySetReportsNoOpeners(t *testing.T) {
	_, err := open(nil, []byte("sealed"))

	if !errors.Is(err, errNoOpeners) {
		t.Errorf("open() error = %v, want %v", err, errNoOpeners)
	}
}
