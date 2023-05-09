package permissions

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	mgmtauthorization "github.com/Azure/azure-sdk-for-go/services/preview/authorization/mgmt/2018-09-01-preview/authorization"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/authz/remotepdp"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/authorization"
	"github.com/Azure/ARO-RP/pkg/util/feature"
)

type PermissionsValidator interface {
	ValidateActions(ctx context.Context, r *azure.Resource, actions []string) error
}

func NewPermissionsValidator(_env env.Interface, log *logrus.Entry, authorizer autorest.Authorizer, oc *api.OpenShiftCluster, sub *api.SubscriptionDocument) (PermissionsValidator, error) {
	if feature.IsRegisteredForFeature(
		sub.Subscription.Properties,
		api.FeatureFlagCheckAccessTestToggle,
	) {
		// TODO remove after successfully migrating to CheckAccess
		log.Info("CheckAccess Feature is set")
		fpClientCred, err := _env.FPNewClientCertificateCredential(sub.Subscription.Properties.TenantID)
		if err != nil {
			return nil, err
		}

		aroEnv := _env.Environment()
		return NewPermissionsValidatorWithPDP(
			log, remotepdp.NewRemotePDPClient(
				fmt.Sprintf(aroEnv.Endpoint, _env.Location()),
				aroEnv.OAuthScope,
				fpClientCred,
			), fpClientCred, _env.Environment().ResourceManagerEndpoint), nil
	}

	return NewPermissionsValidatorWithPermissionsClient(log, authorization.NewPermissionsClient(_env.Environment(), sub.ID, authorizer)), nil
}

// canDoAction returns true if a given action is granted by a set of permissions
func canDoAction(ps []mgmtauthorization.Permission, a string) (bool, error) {
	for _, p := range ps {
		var matched bool

		for _, action := range *p.Actions {
			action := regexp.QuoteMeta(action)
			action = "(?i)^" + strings.ReplaceAll(action, `\*`, ".*") + "$"
			rx, err := regexp.Compile(action)
			if err != nil {
				return false, err
			}
			if rx.MatchString(a) {
				matched = true
				break
			}
		}
		if !matched {
			continue
		}

		for _, notAction := range *p.NotActions {
			notAction := regexp.QuoteMeta(notAction)
			notAction = "(?i)^" + strings.ReplaceAll(notAction, `\*`, ".*") + "$"
			rx, err := regexp.Compile(notAction)
			if err != nil {
				return false, err
			}
			if rx.MatchString(a) {
				matched = false
				break
			}
		}
		if matched {
			return true, nil
		}
	}

	return false, nil
}
