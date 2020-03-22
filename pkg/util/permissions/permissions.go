package permissions

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"regexp"
	"strings"

	mgmtauthorization "github.com/Azure/azure-sdk-for-go/services/preview/authorization/mgmt/2018-09-01-preview/authorization"
)

// CanDoAction returns true if a given action is granted by a set of permissions
func CanDoAction(ps []mgmtauthorization.Permission, a string) (bool, error) {
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
