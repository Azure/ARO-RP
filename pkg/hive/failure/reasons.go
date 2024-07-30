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
	AzureZonalAllocationFailed,
}

var AzureRequestDisallowedByPolicy = InstallFailingReason{
	Name:    "AzureRequestDisallowedByPolicy",
	Reason:  "AzureRequestDisallowedByPolicy",
	Message: "Deployment failed due to RequestDisallowedByPolicy. Please see details for more information.",
	SearchRegexes: []*regexp.Regexp{
		regexp.MustCompile(`"code":\w?"InvalidTemplateDeployment".*"code":\w?"RequestDisallowedByPolicy"`),
	},
}

var AzureInvalidTemplateDeployment = InstallFailingReason{
	Name:    "AzureInvalidTemplateDeployment",
	Reason:  "AzureInvalidTemplateDeployment",
	Message: "Deployment failed. Please see details for more information.",
	SearchRegexes: []*regexp.Regexp{
		regexp.MustCompile(`"code":\w?"InvalidTemplateDeployment"`),
	},
}

var AzureZonalAllocationFailed = InstallFailingReason{
	Name:    "AzureZonalAllocationFailed",
	Reason:  "AzureZonalAllocationFailed",
	Message: "Deployment failed. Please see details for more information.",
	SearchRegexes: []*regexp.Regexp{
		regexp.MustCompile(`"code\W*":\W*"ZonalAllocationFailed\W*"`),
	},
}
