package validate

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/api/validate/dynamic"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/authz/remotepdp"
	"github.com/Azure/ARO-RP/pkg/util/feature"
)

func CreatePDPClient(_env env.Interface, log *logrus.Entry, oc *api.OpenShiftCluster, sub *api.Subscription) (*dynamic.PDPChecker, error) {
	if feature.IsRegisteredForFeature(
		sub.Properties,
		api.FeatureFlagCheckAccessTestToggle,
	) {
		// TODO remove after successfully migrating to CheckAccess
		log.Info("CheckAccess Feature is set")
		var err error
		fpClientCred, err := _env.FPNewClientCertificateCredential(sub.Properties.TenantID)
		if err != nil {
			return nil, err
		}

		aroEnv := _env.Environment()
		return dynamic.NewPDPChecker(
			remotepdp.NewRemotePDPClient(
				fmt.Sprintf(aroEnv.Endpoint, _env.Location()),
				aroEnv.OAuthScope,
				fpClientCred,
			),
			fpClientCred,
		), nil
	}

	return nil, nil
}
