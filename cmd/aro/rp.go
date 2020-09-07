package main

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/Azure/go-autorest/tracing"
	uuid "github.com/satori/go.uuid"
	"github.com/sirupsen/logrus"
	"k8s.io/client-go/tools/metrics"

	"github.com/Azure/ARO-RP/pkg/api"
	_ "github.com/Azure/ARO-RP/pkg/api/admin"
	_ "github.com/Azure/ARO-RP/pkg/api/v20191231preview"
	_ "github.com/Azure/ARO-RP/pkg/api/v20200430"
	_ "github.com/Azure/ARO-RP/pkg/api/v20201031preview"
	"github.com/Azure/ARO-RP/pkg/backend"
	"github.com/Azure/ARO-RP/pkg/database"
	"github.com/Azure/ARO-RP/pkg/deploy/generator"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/frontend"
	"github.com/Azure/ARO-RP/pkg/frontend/adminactions"
	"github.com/Azure/ARO-RP/pkg/metrics/statsd"
	"github.com/Azure/ARO-RP/pkg/metrics/statsd/azure"
	"github.com/Azure/ARO-RP/pkg/metrics/statsd/k8s"
	"github.com/Azure/ARO-RP/pkg/proxy"
	"github.com/Azure/ARO-RP/pkg/util/clientauthorizer"
	"github.com/Azure/ARO-RP/pkg/util/encryption"
	"github.com/Azure/ARO-RP/pkg/util/fakearm"
)

func rp(ctx context.Context, log *logrus.Entry) error {
	uuid := uuid.NewV4().String()
	log.Printf("uuid %s", uuid)

	_env, err := env.NewEnv(ctx, log)
	if err != nil {
		return err
	}

	var keys []string
	if _env.Type() == env.Dev {
		keys = []string{
			"PULL_SECRET",
		}
	} else {
		keys = []string{
			"ACR_RESOURCE_ID",
			"ADMIN_API_CLIENT_CERT_COMMON_NAME",
			"MDM_ACCOUNT",
			"MDM_NAMESPACE",
		}

		if _, found := os.LookupEnv("PULL_SECRET"); found {
			return fmt.Errorf(`environment variable "PULL_SECRET" set`)
		}
	}
	for _, key := range keys {
		if _, found := os.LookupEnv(key); !found {
			return fmt.Errorf("environment variable %q unset", key)
		}
	}

	m, err := statsd.New(ctx, log.WithField("component", "metrics"), _env, os.Getenv("MDM_ACCOUNT"), os.Getenv("MDM_NAMESPACE"))
	if err != nil {
		return err
	}

	tracing.Register(azure.New(m))
	metrics.Register(k8s.NewLatency(m), k8s.NewResult(m))

	kv, err := env.NewServiceKeyvault(ctx, _env)
	if err != nil {
		return err
	}

	dbKey, err := kv.GetSecret(ctx, env.EncryptionSecretName)
	if err != nil {
		return err
	}

	cipher, err := encryption.NewXChaCha20Poly1305(ctx, dbKey)
	if err != nil {
		return err
	}

	dbc, err := database.NewDatabaseClient(ctx, log.WithField("component", "database"), _env, m, cipher)
	if err != nil {
		return err
	}

	dbasyncoperations, err := database.NewAsyncOperations(_env, dbc)
	if err != nil {
		return err
	}

	dbbilling, err := database.NewBilling(ctx, _env, dbc)
	if err != nil {
		return err
	}

	dbopenshiftclusters, err := database.NewOpenShiftClusters(ctx, _env, dbc, uuid)
	if err != nil {
		return err
	}

	go database.EmitQueueLengthMetrics(ctx, log.WithField("component", "database"), dbopenshiftclusters, m)

	dbsubscriptions, err := database.NewSubscriptions(ctx, _env, dbc, uuid)
	if err != nil {
		return err
	}

	feKey, err := kv.GetSecret(ctx, env.FrontendEncryptionSecretName)
	if err != nil {
		return err
	}

	feCipher, err := encryption.NewXChaCha20Poly1305(ctx, feKey)
	if err != nil {
		return err
	}

	dialer, err := proxy.NewDialer(_env)
	if err != nil {
		return err
	}

	var armClientAuthorizer, adminClientAuthorizer clientauthorizer.ClientAuthorizer
	var l net.Listener
	if _env.Type() == env.Dev {
		armClientAuthorizer = clientauthorizer.NewAll()
		adminClientAuthorizer = clientauthorizer.NewAll()

		// in dev mode there is no authentication, so for safety we only listen on
		// localhost
		l, err = net.Listen("tcp", "localhost:8443")
		if err != nil {
			return err
		}

	} else {
		armClientAuthorizer = clientauthorizer.NewARM(log)
		adminClientAuthorizer, err = clientauthorizer.NewAdmin(
			log,
			"/etc/aro-rp/admin-ca-bundle.pem",
			os.Getenv("ADMIN_API_CLIENT_CERT_COMMON_NAME"),
		)
		if err != nil {
			return err
		}

		l, err = net.Listen("tcp", ":8443")
		if err != nil {
			return err
		}
	}

	key, certs, err := kv.GetCertificateSecret(ctx, env.RPServerSecretName)
	if err != nil {
		return err
	}

	gl, err := env.NewClustersGenevaLogging(ctx, kv)
	if err != nil {
		return err
	}

	clustersKeyvaultURI, err := env.GetVaultURI(ctx, _env, generator.ClustersKeyVaultTagValue)
	if err != nil {
		return err
	}

	l = frontend.TLSListener(l, key, certs)

	fp, err := env.NewFPAuthorizer(ctx, _env, kv)
	if err != nil {
		return err
	}

	fakearm, err := fakearm.New(_env, fp)
	if err != nil {
		return err
	}

	f, err := frontend.NewFrontend(ctx, log.WithField("component", "frontend"), _env, fp, dialer, dbasyncoperations, dbopenshiftclusters, dbsubscriptions, l, api.APIs, m, feCipher, adminactions.New, armClientAuthorizer, adminClientAuthorizer)
	if err != nil {
		return err
	}

	b, err := backend.NewBackend(ctx, log.WithField("component", "backend"), _env, fp, gl, dialer, fakearm, dbasyncoperations, dbbilling, dbopenshiftclusters, dbsubscriptions, cipher, m, clustersKeyvaultURI)
	if err != nil {
		return err
	}

	// This part of the code orchestrates shutdown sequence. When sigterm is
	// received, it will trigger backend to stop accepting new documents and
	// finish old ones. Frontend will stop advertising itself to the loadbalancer.
	// When shutdown completes for frontend and backend "/healthz" endpoint
	// will go dark and external observer will know that shutdown sequence is finished
	sigterm := make(chan os.Signal, 1)
	stop := make(chan struct{})
	doneF := make(chan struct{})
	doneB := make(chan struct{})
	signal.Notify(sigterm, syscall.SIGTERM)

	log.Print("listening")
	go b.Run(ctx, stop, doneB)
	go f.Run(ctx, stop, doneF)

	<-sigterm
	log.Print("received SIGTERM")
	close(stop)
	<-doneB
	<-doneF

	return nil
}
