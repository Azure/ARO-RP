package main

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptrace"
	"os"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/jongio/azidext/go/azidext"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/database"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/metrics/noop"
	"github.com/Azure/ARO-RP/pkg/util/encryption"
	"github.com/Azure/ARO-RP/pkg/util/keyvault"
	utillog "github.com/Azure/ARO-RP/pkg/util/log"
)

const (
	DatabaseName        = "DATABASE_NAME"
	DatabaseAccountName = "DATABASE_ACCOUNT_NAME"
	KeyVaultPrefix      = "KEYVAULT_PREFIX"
)

var logger = utillog.GetLogger()

type transport struct {
	current *http.Request
}

func (t *transport) WroteRequest(wri httptrace.WroteRequestInfo) {
	log := logger.WithField("wri", wri).WithError(wri.Err).WithField("t", t)
	log.Infof("WroteRequest")
}

func (t *transport) RoundTrip(req *http.Request) (*http.Response, error) {
	log := logger.WithField("t", t)
	t.current = req
	log.Infof("RoundTrip start")
	fmt.Fprintf(os.Stderr, "%s %s %s\n", req.Method, req.URL, req.Proto)
	for k, v := range req.Header {
		for x, vv := range v {
			if x > 0 {
				k = strings.Repeat(" ", len(k))
			}
			fmt.Fprintf(os.Stderr, "%s: %s\n", k, vv)
		}
	}
	fmt.Fprintf(os.Stderr, "---\n%s\n---\n", req.Body)
	resp, err := http.DefaultTransport.RoundTrip(req)
	log.WithError(err).Infof("RoundTrip stop")
	if resp != nil {
		fmt.Fprintf(os.Stderr, "%d.%d %s\n", resp.ProtoMajor, resp.ProtoMinor, resp.Status)
		for k, v := range resp.Header {
			for x, vv := range v {
				if x > 0 {
					k = strings.Repeat(" ", len(k))
				}
				fmt.Fprintf(os.Stderr, "%s: %s\n", k, vv)
			}
		}
		fmt.Fprintf(os.Stderr, "---\n%s\n---\n", resp.Body)
	}
	return resp, err
}

// GotConn prints whether the connection has been used previously
// for the current request.
func (t *transport) GotConn(info httptrace.GotConnInfo) {
	log := logger.WithFields(logrus.Fields{
		"idle":       info.IdleTime,
		"reused":     info.Reused,
		"wasidle":    info.WasIdle,
		"localaddr":  info.Conn.LocalAddr(),
		"remoteaddr": info.Conn.RemoteAddr(),
		"transport":  t,
	})
	log.Infof("Connection reused")
}

func newTrace(ctx context.Context) context.Context {
	t := &transport{}
	log := logger.WithContext(ctx)

	trace := &httptrace.ClientTrace{
		GotConn:      t.GotConn,
		WroteRequest: t.WroteRequest,
		GotFirstResponseByte: func() {
			log.Infoln("GotFirstResponseByte")
		},
		DNSStart: func(di httptrace.DNSStartInfo) {
			log.WithField("host", di.Host).Infof("DNSStart")
		},
		DNSDone: func(di httptrace.DNSDoneInfo) {
			log.WithError(di.Err).WithField("coalesced", di.Coalesced).WithField("addrs", di.Addrs).Infof("DNSDone")
		},
		// WroteHeaderField: func(key string, value []string) {
		// 	log.WithField("key", key).WithField("value", value).Infof("WroteHeaderField")
		// },
		// WroteHeaders: func() {
		// 	log.Info("WroteHeaders")
		// },
		ConnectStart: func(network, addr string) {
			log.WithField("network", network).WithField("addr", addr).Info("ConnectStart")
		},
		ConnectDone: func(network, addr string, err error) {
			log.WithField("network", network).WithField("addr", addr).WithError(err).Info("ConnectDone")
		},
	}
	ctx = httptrace.WithClientTrace(ctx, trace)
	return ctx
}

func run(ctx context.Context, log *logrus.Entry) error {
	if len(os.Args) != 2 {
		return fmt.Errorf("usage: %s resourceid", os.Args[0])
	}

	t := &transport{}

	ctx = newTrace(ctx)

	_env, err := env.NewCore(ctx, log)
	if err != nil {
		log.WithError(err).WithContext(ctx).WithField("fn", "env.NewCore").WithField("resourceid", os.Args[1]).Error("Failed to create environment")
		return err
	}

	tokenCredential, err := azidentity.NewAzureCLICredential(nil)
	if err != nil {
		log.WithError(err).WithContext(ctx).WithField("fn", "azidentity.NewAzureCLICredential").WithField("resourceid", os.Args[1]).Error("Failed to create token credential")
		return err
	}

	scopes := []string{_env.Environment().ResourceManagerScope}
	authorizer := azidext.NewTokenCredentialAdapter(tokenCredential, scopes)

	msiKVAuthorizer, err := _env.NewMSIAuthorizer(env.MSIContextRP, _env.Environment().KeyVaultScope)
	if err != nil {
		log.WithError(err).WithContext(ctx).WithField("fn", "_env.NewMSIAuthorizer").WithField("resourceid", os.Args[1]).Error("Failed to create MSI authorizer")
		return err
	}

	if err := env.ValidateVars(KeyVaultPrefix); err != nil {
		log.WithError(err).WithContext(ctx).WithField("fn", "keyvault.URI").WithField("resourceid", os.Args[1]).Error("Failed to create keyvault client")
		return err
	}
	keyVaultPrefix := os.Getenv(KeyVaultPrefix)
	serviceKeyvaultURI := keyvault.URI(_env, env.ServiceKeyvaultSuffix, keyVaultPrefix)
	serviceKeyvault := keyvault.NewManager(msiKVAuthorizer, serviceKeyvaultURI)

	aead, err := encryption.NewMulti(ctx, serviceKeyvault, env.EncryptionSecretV2Name, env.EncryptionSecretName)
	if err != nil {
		log.WithError(err).WithContext(ctx).WithField("fn", "encryption.NewMulti").WithField("resourceid", os.Args[1]).Error("Failed to create AEAD")
		return err
	}

	if err := env.ValidateVars(DatabaseAccountName); err != nil {
		return err
	}

	dbAccountName := os.Getenv(DatabaseAccountName)
	dbAuthorizer, err := database.NewMasterKeyAuthorizer(ctx, _env, authorizer, dbAccountName)
	if err != nil {
		log.WithError(err).WithContext(ctx).WithField("fn", "database.NewMasterKeyAuthorizer").WithField("resourceid", os.Args[1]).Error("Failed to create master key authorizer")
		return err
	}

	dbc, err := database.NewDatabaseClientWithTransport(log.WithField("component", "database"), _env, dbAuthorizer, &noop.Noop{}, aead, dbAccountName, t)
	if err != nil {
		log.WithError(err).WithContext(ctx).WithField("fn", "database.NewDatabaseClient").WithField("resourceid", os.Args[1]).Error("Failed to create database client")
		return err
	}

	dbName, err := DBName(_env.IsLocalDevelopmentMode())
	if err != nil {
		return err
	}

	openShiftClusters, err := database.NewOpenShiftClusters(ctx, dbc, dbName)
	if err != nil {
		log.WithError(err).WithContext(ctx).WithField("fn", "database.NewOpenShiftClusters").WithField("resourceid", os.Args[1]).Error("Failed to connect to database")
		return err
	}

	doc, err := openShiftClusters.Get(ctx, strings.ToLower(os.Args[1]))
	if err != nil {
		log.WithError(err).WithContext(ctx).WithField("fn", "openShiftClusters.Get").WithField("resourceid", os.Args[1]).Error("Failed to get cluster")
		return err
	}

	return json.NewEncoder(os.Stdout).Encode(doc)
}

func main() {
	log := utillog.GetLogger()

	if err := run(context.Background(), log); err != nil {
		log.Fatal(err)
	}
}

func DBName(isLocalDevelopmentMode bool) (string, error) {
	if !isLocalDevelopmentMode {
		return "ARO", nil
	}

	if err := env.ValidateVars(DatabaseName); err != nil {
		return "", fmt.Errorf("%v (development mode)", err.Error())
	}

	return os.Getenv(DatabaseName), nil
}
