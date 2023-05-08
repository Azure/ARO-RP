package dynamic

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"

	"github.com/Azure/ARO-RP/pkg/util/azureclient/authz/remotepdp"
	"github.com/Azure/ARO-RP/pkg/util/token"
)

type PDPChecker struct {
	pdp remotepdp.RemotePDPClient

	// This represents the Subject for CheckAccess.  Could be either FP or SP.
	checkAccessSubjectInfoCred azcore.TokenCredential
}

func NewPDPChecker(pdp remotepdp.RemotePDPClient, checkAccessSubjectInfoCred azcore.TokenCredential) *PDPChecker {
	return &PDPChecker{
		pdp:                        pdp,
		checkAccessSubjectInfoCred: checkAccessSubjectInfoCred,
	}
}

// usingCheckAccessV2 uses the new RBAC checkAccessV2 API
func (c closure) usingCheckAccessV2() (bool, error) {
	// TODO remove this when fully migrated to CheckAccess
	c.dv.log.Debug("retry validateActions with CheckAccessV2")

	// reusing oid during retries
	if c.oid == nil {
		scope := c.dv.azEnv.ResourceManagerEndpoint + "/.default"
		t, err := c.dv.pdpChecker.checkAccessSubjectInfoCred.GetToken(c.ctx, policy.TokenRequestOptions{Scopes: []string{scope}})
		if err != nil {
			return false, err
		}

		oid, err := token.GetObjectId(t.Token)
		if err != nil {
			return false, err
		}
		c.oid = &oid
	}

	authReq := createAuthorizationRequest(*c.oid, c.resource.String(), c.actions...)
	results, err := c.dv.pdpChecker.pdp.CheckAccess(c.ctx, authReq)
	if err != nil {
		return false, err
	}

	if results == nil {
		c.dv.log.Info("nil response returned from CheckAccessV2")
		return false, nil
	}

	for _, result := range results.Value {
		if result.AccessDecision != remotepdp.Allowed {
			c.dv.log.Infof("%s has no access to %s", *c.oid, result.ActionId)
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
