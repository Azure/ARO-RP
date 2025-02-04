package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"reflect"
	"strings"
	"testing"

	hivev1 "github.com/openshift/hive/apis/hive/v1"
	"go.uber.org/mock/gomock"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/metrics/noop"
	mock_env "github.com/Azure/ARO-RP/pkg/util/mocks/env"
	mock_hive "github.com/Azure/ARO-RP/pkg/util/mocks/hive"
)

func Test_listAdminHiveSyncSet(t *testing.T) {
	fakeUUID := "00000000-0000-0000-0000-000000000000"
	ctx := context.Background()
	syncsetTest := hivev1.SyncSetList{
		Items: []hivev1.SyncSet{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "syncset-1",
					Namespace: "aro-" + fakeUUID,
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "syncset-2",
					Namespace: "aro-" + fakeUUID,
				},
			},
		},
	}
	selectorSyncSetTest := hivev1.SelectorSyncSetList{
		Items: []hivev1.SelectorSyncSet{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "selectorsyncset-1",
					Namespace: "",
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "selectorsyncset-2",
					Namespace: "",
				},
			},
		},
	}
	type test struct {
		name           string
		namespace      string
		syncsetname    string
		isSyncSet      bool
		hiveEnabled    bool
		mocks          func(*test, *mock_hive.MockSyncSetManager)
		wantStatusCode int
		wantResponse   []byte
		wantError      string
	}

	for _, tt := range []*test{
		{
			name:           "selectorSyncSets are not namespaced",
			namespace:      "aro-" + fakeUUID,
			syncsetname:    "syncsetTest",
			isSyncSet:      false,
			hiveEnabled:    true,
			mocks:          func(tt *test, s *mock_hive.MockSyncSetManager) {},
			wantStatusCode: 400,
			wantError:      "400: InvalidRequestContent: : namespace should be null for listing selectorsyncsets",
		},
		{
			name:           "SyncSets must be namespaced",
			namespace:      "",
			syncsetname:    "syncsetTest",
			isSyncSet:      true,
			hiveEnabled:    true,
			mocks:          func(tt *test, s *mock_hive.MockSyncSetManager) {},
			wantStatusCode: 400,
			wantError:      "400: InvalidRequestContent: : namespace cannot be null for listing syncsets",
		},
		{
			name:        "list Selectorsyncsets",
			namespace:   "",
			syncsetname: "selectorSyncSetTest",
			mocks: func(tt *test, s *mock_hive.MockSyncSetManager) {
				s.EXPECT().
					List(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Eq(reflect.TypeOf(hivev1.SelectorSyncSetList{}))).
					Return(&selectorSyncSetTest, nil).Times(1)
			},
			isSyncSet:      false,
			hiveEnabled:    true,
			wantStatusCode: 200,
			wantError:      "",
			wantResponse:   []byte(`{"items":[{"metadata":{"name":"selectorsyncset-1","namespace":""}},{"metadata":{"name":"selectorsyncset-2","namespace":""}}]}`),
		},
		{
			name:        "list syncsets",
			namespace:   "aro-" + fakeUUID,
			syncsetname: "syncSetTest",
			mocks: func(tt *test, s *mock_hive.MockSyncSetManager) {
				s.EXPECT().
					List(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Eq(reflect.TypeOf(hivev1.SyncSetList{}))).
					Return(&syncsetTest, nil).Times(1)
			},
			isSyncSet:      true,
			hiveEnabled:    true,
			wantStatusCode: 200,
			wantError:      "",
			wantResponse:   []byte(`{"items":[{"metadata":{"name":"syncset-1","namespace":"aro-00000000-0000-0000-0000-000000000000"}},{"metadata":{"name":"syncset-2","namespace":"aro-00000000-0000-0000-0000-000000000000"}}]}`),
		},
		{
			name:           "Hive is not enabled selector/syncsets",
			namespace:      "aro-" + fakeUUID,
			syncsetname:    "syncSetTest",
			mocks:          nil,
			isSyncSet:      false,
			hiveEnabled:    false,
			wantStatusCode: 500,
			wantError:      "500: InternalServerError: : hive is not enabled",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ti := newTestInfra(t).WithOpenShiftClusters().WithSubscriptions()
			defer ti.done()

			_env := ti.env.(*mock_env.MockInterface)
			var f *frontend
			var err error
			if tt.hiveEnabled {
				s := mock_hive.NewMockSyncSetManager(ti.controller)
				tt.mocks(tt, s)
				f, err = NewFrontend(ctx, ti.audit, ti.log, _env, ti.dbGroup, api.APIs, &noop.Noop{}, &noop.Noop{}, nil, nil, s, nil, nil, nil, nil)
			} else {
				f, err = NewFrontend(ctx, ti.audit, ti.log, _env, ti.dbGroup, api.APIs, &noop.Noop{}, &noop.Noop{}, nil, nil, nil, nil, nil, nil, nil)
			}
			if err != nil {
				t.Fatal(err)
			}

			hiveSyncSet, err := f._listAdminHiveSyncSet(ctx, tt.namespace, tt.syncsetname, tt.isSyncSet)
			cloudErr, isCloudErr := err.(*api.CloudError)
			if tt.wantError != "" && isCloudErr && cloudErr != nil {
				if tt.wantError != cloudErr.Error() {
					t.Fatalf("got %q but wanted %q", cloudErr.Error(), tt.wantError)
				}
				if tt.wantStatusCode != 0 && tt.wantStatusCode != cloudErr.StatusCode {
					t.Fatalf("got %q but wanted %q", cloudErr.Error(), tt.wantError)
				}
			}

			if !strings.EqualFold(string(hiveSyncSet), string(tt.wantResponse)) {
				t.Fatalf("got %q and expected %q", hiveSyncSet, tt.wantResponse)
			}
		})
	}
}
