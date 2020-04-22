package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"reflect"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/database"
	"github.com/Azure/ARO-RP/pkg/database/cosmosdb"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/metrics/noop"
	"github.com/Azure/ARO-RP/pkg/util/clientauthorizer"
	mock_database "github.com/Azure/ARO-RP/pkg/util/mocks/database"
	utiltls "github.com/Azure/ARO-RP/pkg/util/tls"
	"github.com/Azure/ARO-RP/test/util/listener"
)

func TestPutSubscription(t *testing.T) {
	ctx := context.Background()

	clientkey, clientcerts, err := utiltls.GenerateKeyAndCertificate("client", nil, nil, false, true)
	if err != nil {
		t.Fatal(err)
	}

	serverkey, servercerts, err := utiltls.GenerateKeyAndCertificate("server", nil, nil, false, false)
	if err != nil {
		t.Fatal(err)
	}

	pool := x509.NewCertPool()
	pool.AddCert(servercerts[0])

	cli := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				RootCAs: pool,
				Certificates: []tls.Certificate{
					{
						Certificate: [][]byte{clientcerts[0].Raw},
						PrivateKey:  clientkey,
					},
				},
			},
		},
	}

	mockSubID := "00000000-0000-0000-0000-000000000000"

	type test struct {
		name           string
		request        func(*api.Subscription)
		dbGetDoc       *api.SubscriptionDocument
		dbGetErr       error
		wantDbDoc      *api.SubscriptionDocument
		wantStatusCode int
		wantError      string
	}

	for _, tt := range []*test{
		{
			name: "add a new subscription - registered state",
			request: func(sub *api.Subscription) {
				sub.State = api.SubscriptionStateRegistered
			},
			dbGetErr: &cosmosdb.Error{StatusCode: http.StatusNotFound},
			wantDbDoc: &api.SubscriptionDocument{
				ID: mockSubID,
				Subscription: &api.Subscription{
					State: api.SubscriptionStateRegistered,
				},
			},
			wantStatusCode: http.StatusCreated,
		},
		{
			name: "add a new subscription - warned state",
			request: func(sub *api.Subscription) {
				sub.State = api.SubscriptionStateWarned
			},
			dbGetErr: &cosmosdb.Error{StatusCode: http.StatusNotFound},
			wantDbDoc: &api.SubscriptionDocument{
				ID: mockSubID,
				Subscription: &api.Subscription{
					State: api.SubscriptionStateWarned,
				},
			},
			wantStatusCode: http.StatusCreated,
		},
		{
			name: "add a new subscription - suspended state",
			request: func(sub *api.Subscription) {
				sub.State = api.SubscriptionStateSuspended
			},
			dbGetErr: &cosmosdb.Error{StatusCode: http.StatusNotFound},
			wantDbDoc: &api.SubscriptionDocument{
				ID: mockSubID,
				Subscription: &api.Subscription{
					State: api.SubscriptionStateSuspended,
				},
			},
			wantStatusCode: http.StatusCreated,
		},
		{
			name: "add a new subscription - unregistered state",
			request: func(sub *api.Subscription) {
				sub.State = api.SubscriptionStateUnregistered
			},
			dbGetErr: &cosmosdb.Error{StatusCode: http.StatusNotFound},
			wantDbDoc: &api.SubscriptionDocument{
				ID: mockSubID,
				Subscription: &api.Subscription{
					State: api.SubscriptionStateUnregistered,
				},
			},
			wantStatusCode: http.StatusCreated,
		},
		{
			name: "add a new subscription - deleted state",
			request: func(sub *api.Subscription) {
				sub.State = api.SubscriptionStateDeleted
			},
			dbGetErr: &cosmosdb.Error{StatusCode: http.StatusNotFound},
			wantDbDoc: &api.SubscriptionDocument{
				ID:       mockSubID,
				Deleting: true,
				Subscription: &api.Subscription{
					State: api.SubscriptionStateDeleted,
				},
			},
			wantStatusCode: http.StatusCreated,
		},
		{
			name: "update an existing subscription - registered",
			request: func(sub *api.Subscription) {
				sub.State = api.SubscriptionStateWarned
				sub.Properties = &api.SubscriptionProperties{TenantID: "changed"}
			},
			dbGetDoc: &api.SubscriptionDocument{
				ID: mockSubID,
				Subscription: &api.Subscription{
					State: api.SubscriptionStateRegistered,
				},
			},
			wantDbDoc: &api.SubscriptionDocument{
				ID: mockSubID,
				Subscription: &api.Subscription{
					State:      api.SubscriptionStateWarned,
					Properties: &api.SubscriptionProperties{TenantID: "changed"},
				},
			},
			wantStatusCode: http.StatusOK,
		},
		{
			name: "update an existing subscription - warned state",
			request: func(sub *api.Subscription) {
				sub.State = api.SubscriptionStateSuspended
				sub.Properties = &api.SubscriptionProperties{TenantID: "changed"}
			},
			dbGetDoc: &api.SubscriptionDocument{
				ID: mockSubID,
				Subscription: &api.Subscription{
					State: api.SubscriptionStateWarned,
				},
			},
			wantDbDoc: &api.SubscriptionDocument{
				ID: mockSubID,
				Subscription: &api.Subscription{
					State:      api.SubscriptionStateSuspended,
					Properties: &api.SubscriptionProperties{TenantID: "changed"},
				},
			},
			wantStatusCode: http.StatusOK,
		},
		{
			name: "update an existing subscription - suspended state",
			request: func(sub *api.Subscription) {
				sub.State = api.SubscriptionStateDeleted
				sub.Properties = &api.SubscriptionProperties{TenantID: "changed"}
			},
			dbGetDoc: &api.SubscriptionDocument{
				ID: mockSubID,
				Subscription: &api.Subscription{
					State: api.SubscriptionStateSuspended,
				},
			},
			wantDbDoc: &api.SubscriptionDocument{
				ID:       mockSubID,
				Deleting: true,
				Subscription: &api.Subscription{
					State:      api.SubscriptionStateDeleted,
					Properties: &api.SubscriptionProperties{TenantID: "changed"},
				},
			},
			wantStatusCode: http.StatusOK,
		},
		{
			name: "update an existing subscription - unregistered state",
			request: func(sub *api.Subscription) {
				sub.State = api.SubscriptionStateRegistered
				sub.Properties = &api.SubscriptionProperties{TenantID: "changed"}
			},
			dbGetDoc: &api.SubscriptionDocument{
				ID: mockSubID,
				Subscription: &api.Subscription{
					State: api.SubscriptionStateUnregistered,
				},
			},
			wantDbDoc: &api.SubscriptionDocument{
				ID: mockSubID,
				Subscription: &api.Subscription{
					State:      api.SubscriptionStateRegistered,
					Properties: &api.SubscriptionProperties{TenantID: "changed"},
				},
			},
			wantStatusCode: http.StatusOK,
		},
		{
			name: "update an existing subscription - deleted state",
			request: func(sub *api.Subscription) {
				sub.State = api.SubscriptionStateUnregistered
				sub.Properties = &api.SubscriptionProperties{TenantID: "changed"}
			},
			dbGetDoc: &api.SubscriptionDocument{
				ID:       mockSubID,
				Deleting: true,
				Subscription: &api.Subscription{
					State: api.SubscriptionStateDeleted,
				},
			},
			wantStatusCode: http.StatusBadRequest,
			wantError:      `400: InvalidSubscriptionState: : Request is not allowed in subscription in state 'Deleted'.`,
		},
		{
			name:           "internal error",
			dbGetErr:       errors.New("random error"),
			wantStatusCode: http.StatusInternalServerError,
			wantError:      `500: InternalServerError: : Internal server error.`,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			defer cli.CloseIdleConnections()

			l := listener.NewListener()
			defer l.Close()

			env := &env.Test{
				L:        l,
				TLSKey:   serverkey,
				TLSCerts: servercerts,
			}
			env.SetARMClientAuthorizer(clientauthorizer.NewOne(clientcerts[0].Raw))

			cli.Transport.(*http.Transport).Dial = l.Dial

			controller := gomock.NewController(t)
			defer controller.Finish()

			subscriptions := mock_database.NewMockSubscriptions(controller)
			subscriptions.EXPECT().Get(gomock.Any(), mockSubID).Return(tt.dbGetDoc, tt.dbGetErr)
			if tt.wantDbDoc != nil {
				if tt.dbGetDoc == nil {
					subscriptions.EXPECT().Create(gomock.Any(), tt.wantDbDoc).Return(tt.wantDbDoc, nil)
				} else {
					subscriptions.EXPECT().Update(gomock.Any(), tt.wantDbDoc).Return(tt.wantDbDoc, nil)
				}
			}

			f, err := NewFrontend(ctx, logrus.NewEntry(logrus.StandardLogger()), env, &database.Database{
				Subscriptions: subscriptions,
			}, &noop.Noop{}, nil)
			if err != nil {
				t.Fatal(err)
			}
			enrichFrontendForTest(f.(*frontend), api.APIs, nil, nil, nil)

			go f.Run(ctx, nil, nil)

			buf := &bytes.Buffer{}
			sub := &api.Subscription{}
			if tt.request != nil {
				tt.request(sub)
			}
			err = json.NewEncoder(buf).Encode(sub)
			if err != nil {
				t.Fatal(err)
			}

			req, err := http.NewRequest(http.MethodPut, fmt.Sprintf("https://server/subscriptions/%s?api-version=2.0", mockSubID), buf)
			if err != nil {
				t.Fatal(err)
			}
			req.Header = http.Header{
				"Content-Type": []string{"application/json"},
			}
			resp, err := cli.Do(req)
			if err != nil {
				t.Fatal(err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != tt.wantStatusCode {
				t.Error(resp.StatusCode)
			}

			b, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				t.Fatal(err)
			}

			if tt.wantError == "" {
				var sub *api.Subscription
				err = json.Unmarshal(b, &sub)
				if err != nil {
					t.Fatal(err)
				}

				if !reflect.DeepEqual(sub, tt.wantDbDoc.Subscription) {
					b, _ := json.Marshal(sub)
					t.Error(string(b))
				}
			} else {
				cloudErr := &api.CloudError{StatusCode: resp.StatusCode}
				err = json.Unmarshal(b, &cloudErr)
				if err != nil {
					t.Fatal(err)
				}

				if cloudErr.Error() != tt.wantError {
					t.Error(cloudErr)
				}
			}
		})
	}
}
