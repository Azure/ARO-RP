package ocm

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"encoding/json"
	"strings"

	log "github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api"
)

func ValidateOCMFromSystemData(systemDataHeaderStr string, validClientIDs []string) bool {
	var systemData *api.SystemData
	err := json.Unmarshal([]byte(systemDataHeaderStr), &systemData)
	if err != nil {
		log.Errorf("Unable to unmarshal systemData: %q", err)
		return false
	}

	if systemData.LastModifiedByType != api.CreatedByTypeApplication {
		log.Errorf("Only applications are authorized, received: %q", systemData.LastModifiedByType)
		return false
	}

	for _, validClientId := range validClientIDs {
		if strings.EqualFold(systemData.LastModifiedBy, validClientId) {
			return true
		}
	}
	return false
}
