package matcher

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"fmt"
	"reflect"

	"github.com/Azure/ARO-RP/pkg/api"
)

// OpenShiftClusterDocument compares api.OpenShiftClusterDocument objects
// ignoring dynamic fields such as UUIDs
type OpenShiftClusterDocument api.OpenShiftClusterDocument

func (m *OpenShiftClusterDocument) Matches(x interface{}) bool {
	doc, ok := x.(*api.OpenShiftClusterDocument)
	if !ok {
		return false
	}

	id, asyncOperationID := doc.ID, doc.AsyncOperationID
	doc.ID, doc.AsyncOperationID = m.ID, m.AsyncOperationID

	defer func() {
		doc.ID, doc.AsyncOperationID = id, asyncOperationID
	}()

	return reflect.DeepEqual((*api.OpenShiftClusterDocument)(m), doc)
}

func (m *OpenShiftClusterDocument) String() string {
	return fmt.Sprintf("is equal to %v without comparing IDs", (*api.OpenShiftClusterDocument)(m))
}

// AsyncOperationDocument compares api.AsyncOperationDocument objects
// ignoring dynamic fields such as UUIDs
type AsyncOperationDocument api.AsyncOperationDocument

func (m *AsyncOperationDocument) Matches(x interface{}) bool {
	doc, ok := x.(*api.AsyncOperationDocument)
	if !ok {
		return false
	}

	id, asyncOperationName, asyncOperationID, asyncOperationStartTime :=
		doc.ID, doc.AsyncOperation.Name, doc.AsyncOperation.ID, doc.AsyncOperation.StartTime
	doc.ID, doc.AsyncOperation.Name, doc.AsyncOperation.ID, doc.AsyncOperation.StartTime =
		m.ID, m.AsyncOperation.Name, m.AsyncOperation.ID, m.AsyncOperation.StartTime

	defer func() {
		doc.ID, doc.AsyncOperation.Name, doc.AsyncOperation.ID, doc.AsyncOperation.StartTime =
			id, asyncOperationName, asyncOperationID, asyncOperationStartTime
	}()

	return reflect.DeepEqual((*api.AsyncOperationDocument)(m), doc)
}

func (m *AsyncOperationDocument) String() string {
	return fmt.Sprintf("is equal to %v without comparing IDs", (*api.AsyncOperationDocument)(m))
}
