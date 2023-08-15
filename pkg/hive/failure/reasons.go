package failure

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import "regexp"

type InstallFailingReason struct {
	Name          string
	Reason        string
	Message       string
	SearchRegexes []*regexp.Regexp
}

var Reasons = []InstallFailingReason{
	// Order within this array determines precedence. Earlier entries will take
	// priority over later ones.
	AzureRequestDisallowedByPolicy,
	AzureInvalidTemplateDeployment,
}

var AzureRequestDisallowedByPolicy = InstallFailingReason{
	Name:    "AzureRequestDisallowedByPolicy",
	Reason:  "AzureRequestDisallowedByPolicy",
	Message: "Cluster Deployment was disallowed by policy.  Please see install log for more information.",
	SearchRegexes: []*regexp.Regexp{
		regexp.MustCompile(`"code":\w?"InvalidTemplateDeployment".*"code":\w?"RequestDisallowedByPolicy"`),
	},
}

var AzureInvalidTemplateDeployment = InstallFailingReason{
	Name:    "AzureInvalidTemplateDeployment",
	Reason:  "AzureInvalidTemplateDeployment",
	Message: "The template deployment failed. Please see install log for more information.",
	SearchRegexes: []*regexp.Regexp{
		regexp.MustCompile(`"code":\w?"InvalidTemplateDeployment"`),
	},
}
