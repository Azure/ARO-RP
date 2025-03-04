package env

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"sync"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/util/azureclient/azuresdk/azsecrets"
)

type CertificateRefresher interface {
	Start(context.Context) error
	GetCertificates() (*rsa.PrivateKey, []*x509.Certificate)
}

type refreshingCertificate struct {
	lock      sync.RWMutex
	certs     []*x509.Certificate
	key       *rsa.PrivateKey
	logger    *logrus.Entry
	kv        azsecrets.Client
	certName  string
	newTicker func() (tick <-chan time.Time, stop func())
}

func newCertificateRefresher(logger *logrus.Entry, interval time.Duration, kv azsecrets.Client, certificateName string) CertificateRefresher {
	return &refreshingCertificate{
		logger:   logger,
		kv:       kv,
		certName: certificateName,
		newTicker: func() (tick <-chan time.Time, stop func()) {
			ticker := time.NewTicker(interval)
			return ticker.C, func() { ticker.Stop() }
		},
	}
}

func (r *refreshingCertificate) Start(ctx context.Context) error {
	// initial pull to get the certificate start
	err := r.fetchCertificateOnce(ctx)
	if err != nil {
		return err
	}

	r.fetchCertificate(ctx)

	return nil
}

// GetCertificates loads the certificate from the synced store safe to use concurently
func (r *refreshingCertificate) GetCertificates() (*rsa.PrivateKey, []*x509.Certificate) {
	r.lock.RLock()
	defer r.lock.RUnlock()

	return r.key, r.certs
}

// fetchCertificateOnce access keyvault via preset getter and download new set
// of certificates.
// in case of failure error is returned and old certificate is left in the
// synced store
func (r *refreshingCertificate) fetchCertificateOnce(ctx context.Context) error {
	certificate, err := r.kv.GetSecret(ctx, r.certName, "", nil)
	if err != nil {
		return err
	}

	key, certs, err := azsecrets.ParseSecretAsCertificate(certificate)
	if err != nil {
		return err
	}

	r.lock.Lock()
	defer r.lock.Unlock()
	r.key = key
	r.certs = certs

	return nil
}

// fetchCertificate starts goroutine to poll certificates
func (r *refreshingCertificate) fetchCertificate(ctx context.Context) {
	tick, stop := r.newTicker()

	go func() {
		defer stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-tick:
				err := r.fetchCertificateOnce(ctx)
				if err != nil {
					r.logger.Errorf("cannot pull certificate leaving old one, %s", err.Error())
				}
			}
		}
	}()
}
