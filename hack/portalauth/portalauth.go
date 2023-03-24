package main

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/securecookie"
	"github.com/gorilla/sessions"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/util/keyvault"
	utillog "github.com/Azure/ARO-RP/pkg/util/log"
)

const (
	SessionName        = "session"
	SessionKeyExpires  = "expires"
	SessionKeyUsername = "user_name"
	SessionKeyGroups   = "groups"
)

func run(ctx context.Context, log *logrus.Entry) error {
	username := flag.String("username", "testuser", "username of the portal user")
	groups := flag.String("groups", "", "comma-separated list of groups the user is in")

	flag.Parse()

	_env, err := env.NewCore(ctx, log)
	if err != nil {
		return err
	}

	msiKVAuthorizer, err := _env.NewMSIAuthorizer(env.MSIContextRP, _env.Environment().ResourceIdentifiers.KeyVault)
	if err != nil {
		return err
	}

	portalKeyvaultURI, err := keyvault.URI(_env, env.PortalKeyvaultSuffix)
	if err != nil {
		return err
	}

	portalKeyvault := keyvault.NewManager(msiKVAuthorizer, portalKeyvaultURI)

	sessionKey, err := portalKeyvault.GetBase64Secret(ctx, env.PortalServerSessionKeySecretName, "")
	if err != nil {
		return err
	}

	store := sessions.NewCookieStore(sessionKey)

	store.MaxAge(0)
	store.Options.Secure = true
	store.Options.HttpOnly = true
	store.Options.SameSite = http.SameSiteLaxMode

	session := sessions.NewSession(store, SessionName)
	opts := *store.Options
	session.Options = &opts

	session.Values[SessionKeyUsername] = username
	session.Values[SessionKeyGroups] = strings.Split(*groups, ",")
	session.Values[SessionKeyExpires] = time.Now().Add(time.Hour).Unix()

	encoded, err := securecookie.EncodeMulti(session.Name(), session.Values,
		store.Codecs...)
	if err != nil {
		return err
	}

	// Print session variable to stdout
	fmt.Printf("%s", encoded)

	return nil
}

func main() {
	log := utillog.GetLogger()

	if err := run(context.Background(), log); err != nil {
		log.Fatal(err)
	}
}
