package permissions

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/Azure/ARO-RP/pkg/util/azureclient/authz/remotepdp"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/azuresdk/azcore"
	"github.com/Azure/ARO-RP/pkg/util/token"
)

type permissionsValidatorPDP struct {
	log                     *logrus.Entry
	oid                     *string
	resourceManagerEndpoint string

	pdpClient remotepdp.RemotePDPClient

	// This represents the Subject for CheckAccess.  Could be either FP or SP.
	checkAccessSubjectInfoCred azcore.TokenCredential
}

func NewPermissionsValidatorWithPDP(log *logrus.Entry, pdpClient remotepdp.RemotePDPClient, checkAccessSubjectInfoCred azcore.TokenCredential, resourceManagerEndpoint string) PermissionsValidator {
	return &permissionsValidatorPDP{
		log:                        log,
		pdpClient:                  pdpClient,
		resourceManagerEndpoint:    resourceManagerEndpoint,
		checkAccessSubjectInfoCred: checkAccessSubjectInfoCred,
	}
}

func (p *permissionsValidatorPDP) ValidateActions(ctx context.Context, r *azure.Resource, actions []string) error {
	timeout := 65 * time.Second // checkAccess refreshes data every min. This allows ~3 retries.

	timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	return wait.PollImmediateUntil(timeout, func() (bool, error) {
		return p.usingCheckAccessV2(ctx, r, actions)
	}, timeoutCtx.Done())
}

// usingCheckAccessV2 uses the new RBAC checkAccessV2 API
func (p permissionsValidatorPDP) usingCheckAccessV2(
	ctx context.Context,
	resource *azure.Resource,
	actions []string) (bool, error) {
	// TODO remove this when fully migrated to CheckAccess
	p.log.Debug("retry validateActions with CheckAccessV2")

	// reusing oid during retries
	if p.oid == nil {
		scope := p.resourceManagerEndpoint + "/.default"
		t, err := p.checkAccessSubjectInfoCred.GetToken(ctx, policy.TokenRequestOptions{Scopes: []string{scope}})
		if err != nil {
			return false, err
		}

		oid, err := token.GetObjectId(t.Token)
		if err != nil {
			return false, err
		}
		p.oid = &oid
	}

	authReq := createAuthorizationRequest(*p.oid, resource.String(), actions...)
	results, err := p.pdpClient.CheckAccess(ctx, authReq)
	if err != nil {
		return false, err
	}

	if results == nil {
		p.log.Info("nil response returned from CheckAccessV2")
		return false, nil
	}

	for _, result := range results.Value {
		if result.AccessDecision != remotepdp.Allowed {
			p.log.Infof("%s has no access to %s", *p.oid, result.ActionId)
			return false, nil
		}
	}

	return true, nil
}

func createAuthorizationRequest(subject, resourceId string, actions ...string) remotepdp.AuthorizationRequest {
	actionInfos := []remotepdp.ActionInfo{}
	for _, action := range actions {
		actionInfos = append(actionInfos, remotepdp.ActionInfo{Id: action})
	}

	return remotepdp.AuthorizationRequest{
		Subject: remotepdp.SubjectInfo{
			Attributes: remotepdp.SubjectAttributes{
				ObjectId: subject,
			},
		},
		Actions: actionInfos,
		Resource: remotepdp.ResourceInfo{
			Id: resourceId,
		},
	}
}
