package miseadapter

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"net/http"
	"sort"
	"testing"

	"github.com/go-test/deep"
)

func Test_createRequest(t *testing.T) {
	miseAddress := "http://localhost:5000"

	translatedRequest, err := http.NewRequestWithContext(context.Background(), http.MethodPost, miseAddress, nil)
	if err != nil {
		t.Fatal(err)
	}

	translatedRequest.Header = http.Header{
		"Original-Uri":                    []string{"http://1.2.3.4/view"},
		"Original-Method":                 []string{http.MethodGet},
		"X-Forwarded-For":                 []string{"http://2.3.4.5"},
		"Authorization":                   []string{"Bearer token"},
		"Return-All-Actor-Token-Claims":   []string{"1"},
		"Return-All-Subject-Token-Claims": []string{"1"},
	}

	translatedRequestWithSpecificClaims, err := http.NewRequestWithContext(context.Background(), http.MethodPost, miseAddress, nil)
	if err != nil {
		t.Fatal(err)
	}

	translatedRequestWithSpecificClaims.Header = http.Header{
		"Original-Uri":                   []string{"http://1.2.3.4/view"},
		"Original-Method":                []string{http.MethodGet},
		"X-Forwarded-For":                []string{"http://2.3.4.5"},
		"Authorization":                  []string{"Bearer token"},
		"Return-Actor-Token-Claim-Tid":   []string{"1"},
		"Return-Subject-Token-Claim-Tid": []string{"1"},
	}

	type args struct {
		ctx         context.Context
		miseAddress string
		input       Input
	}
	tests := []struct {
		name    string
		args    args
		want    *http.Request
		wantErr bool
	}{
		{
			name: "Input is translated",
			args: args{
				ctx:         context.Background(),
				miseAddress: miseAddress,
				input: Input{
					OriginalUri:            "http://1.2.3.4/view",
					OriginalMethod:         http.MethodGet,
					OriginalIPAddress:      "http://2.3.4.5",
					AuthorizationHeader:    "Bearer token",
					ReturnAllActorClaims:   true,
					ReturnAllSubjectClaims: true,
				},
			},
			want:    translatedRequest,
			wantErr: false,
		},
		{
			name: "Input is translated with specific claims",
			args: args{
				ctx:         context.Background(),
				miseAddress: miseAddress,
				input: Input{
					OriginalUri:           "http://1.2.3.4/view",
					OriginalMethod:        http.MethodGet,
					OriginalIPAddress:     "http://2.3.4.5",
					AuthorizationHeader:   "Bearer token",
					ActorClaimsToReturn:   []string{"tid"},
					SubjectClaimsToReturn: []string{"tid"},
				},
			},
			want:    translatedRequestWithSpecificClaims,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := createRequest(tt.args.ctx, tt.args.miseAddress, tt.args.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("createRequest() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if diff := deep.Equal(tt.want, got); diff != nil {
				t.Errorf("-want/+got:\n%s", diff)
				return
			}
		})
	}
}

func Test_parseResponseIntoResult(t *testing.T) {
	type args struct {
		response *http.Response
	}

	tests := []struct {
		name    string
		args    args
		want    Result
		wantErr bool
	}{
		{
			name: "parse OK response and claims",
			args: args{
				response: &http.Response{
					StatusCode: http.StatusOK,
					Header: http.Header{
						http.CanonicalHeaderKey("Subject-Token-Claim-tid"): []string{"tid-2"},
						http.CanonicalHeaderKey("Actor-Token-Claim-tid"):   []string{"tid-1"},
					},
				},
			},
			want: Result{
				StatusCode:    http.StatusOK,
				ActorClaims:   map[string][]string{"tid": {"tid-1"}},
				SubjectClaims: map[string][]string{"tid": {"tid-2"}},
			},
		},
		{
			name: "parse OK response and encoded claims",
			args: args{
				response: &http.Response{
					StatusCode: http.StatusOK,
					Header: http.Header{
						http.CanonicalHeaderKey("Subject-Token-Encoded-Claim-season"): []string{"ZnLDvGhsaW5n"},
						http.CanonicalHeaderKey("Actor-Token-Encoded-Claim-season"):   []string{"5pil"},
					},
				},
			},
			want: Result{
				StatusCode:    http.StatusOK,
				ActorClaims:   map[string][]string{"season": {"春"}},
				SubjectClaims: map[string][]string{"season": {"frühling"}},
			},
		},
		{
			name: "parse OK response and encoded claims roles",
			args: args{
				response: &http.Response{
					StatusCode: http.StatusOK,
					Header: http.Header{
						http.CanonicalHeaderKey("Subject-Token-Encoded-Claim-roles"): []string{"ZnLDvGhsaW5n", "5pil"},
					},
				},
			},
			want: Result{
				StatusCode:    http.StatusOK,
				ActorClaims:   map[string][]string{},
				SubjectClaims: map[string][]string{"roles": {"frühling", "春"}},
			},
		},
		{
			name: "parse OK response and not encoded and encoded claims roles",
			args: args{
				response: &http.Response{
					StatusCode: http.StatusOK,
					Header: http.Header{
						http.CanonicalHeaderKey("Subject-Token-Claim-roles"):         []string{"spring"},
						http.CanonicalHeaderKey("Subject-Token-Encoded-Claim-roles"): []string{"ZnLDvGhsaW5n"},
					},
				},
			},
			want: Result{
				StatusCode:    http.StatusOK,
				ActorClaims:   map[string][]string{},
				SubjectClaims: map[string][]string{"roles": {"frühling", "spring"}},
			},
		},
		{
			name: "parse OK response and encoded and not encoded claims roles",
			args: args{
				response: &http.Response{
					StatusCode: http.StatusOK,
					Header: http.Header{
						http.CanonicalHeaderKey("Subject-Token-Encoded-Claim-roles"): []string{"ZnLDvGhsaW5n"},
						http.CanonicalHeaderKey("Subject-Token-Claim-roles"):         []string{"spring"},
					},
				},
			},
			want: Result{
				StatusCode:    http.StatusOK,
				ActorClaims:   map[string][]string{},
				SubjectClaims: map[string][]string{"roles": {"frühling", "spring"}},
			},
		},
		{
			name: "parse OK response and claims with multiple values",
			args: args{
				response: &http.Response{
					StatusCode: http.StatusOK,
					Header: http.Header{
						http.CanonicalHeaderKey("Subject-Token-Claim-roles"): []string{"role1", "role2"},
					},
				},
			},
			want: Result{
				StatusCode:    http.StatusOK,
				ActorClaims:   map[string][]string{},
				SubjectClaims: map[string][]string{"roles": {"role1", "role2"}},
			},
		},
		{
			name: "parse 401 response",
			args: args{
				response: &http.Response{
					StatusCode: http.StatusUnauthorized,
					Header: http.Header{
						http.CanonicalHeaderKey("Error-Description"): []string{"invalid issuer"},
						http.CanonicalHeaderKey("Www-Authenticate"):  []string{"invalid token"},
					},
				},
			},
			want: Result{
				StatusCode:       http.StatusUnauthorized,
				WWWAuthenticate:  []string{"invalid token"},
				ErrorDescription: []string{"invalid issuer"},
				ActorClaims:      map[string][]string{},
				SubjectClaims:    map[string][]string{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, gotErr := parseResponseIntoResult(tt.args.response)

			if tt.wantErr != (gotErr != nil) {
				t.Errorf("wantErr: %v, gotErr: %v", tt.wantErr, gotErr)
			}

			if got.SubjectClaims != nil && got.SubjectClaims["roles"] != nil {
				sort.StringSlice(got.SubjectClaims["roles"]).Sort()
			}

			if diff := deep.Equal(tt.want, got); diff != nil {
				t.Errorf("-want/+got:\n%s", diff)
				return
			}
		})
	}
}
