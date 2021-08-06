package banner

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"strings"
	"testing"

	consolev1 "github.com/openshift/api/console/v1"
	consolefake "github.com/openshift/client-go/console/clientset/versioned/fake"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"

	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	arofake "github.com/Azure/ARO-RP/pkg/operator/clientset/versioned/fake"
	utillog "github.com/Azure/ARO-RP/pkg/util/log"
)

func TestBannerReconcile(t *testing.T) {
	r := Reconciler{log: utillog.GetLogger()}
	for _, tt := range []struct {
		name            string
		oldCN           consolev1.ConsoleNotification
		bannerSetting   string
		expectBanner    bool
		expectedMessage string
		expectedErr     error
		featureFlag     bool
	}{
		{
			name:          "Wrong banner setting",
			bannerSetting: "WRONG",
			expectBanner:  false,
			expectedErr:   fmt.Errorf("wrong banner setting '%s'", "WRONG"),
			featureFlag:   true,
		},
		{
			name:          "No banner when feature disabled",
			bannerSetting: "ContactSupport",
			expectBanner:  false,
			featureFlag:   false,
		},
		{
			name: "Banner not deleted when feature disabled",
			oldCN: consolev1.ConsoleNotification{
				ObjectMeta: metav1.ObjectMeta{
					Name: BannerName,
				},
				Spec: consolev1.ConsoleNotificationSpec{
					Text:            "OLD BANNER TEXT",
					Location:        consolev1.BannerTop,
					Color:           "#000",
					BackgroundColor: "#ff0",
				},
			},
			bannerSetting:   "",
			expectBanner:    true,
			expectedMessage: "OLD BANNER TEXT",
			featureFlag:     false,
		},
		{
			name: "Banner not modified when feature disabled",
			oldCN: consolev1.ConsoleNotification{
				ObjectMeta: metav1.ObjectMeta{
					Name: BannerName,
				},
				Spec: consolev1.ConsoleNotificationSpec{
					Text:            "OLD BANNER TEXT",
					Location:        consolev1.BannerTop,
					Color:           "#000",
					BackgroundColor: "#ff0",
				},
			},
			bannerSetting:   "ContactSupport",
			expectBanner:    true,
			expectedMessage: "OLD BANNER TEXT",
			featureFlag:     false,
		},
		{
			name:          "No banner from empty",
			bannerSetting: "",
			expectBanner:  false,
			featureFlag:   true,
		},
		{
			name:            "Support banner from empty",
			bannerSetting:   "ContactSupport",
			expectBanner:    true,
			expectedMessage: "We have noticed an issue regarding your cluster requiring an action on your part. Please contact support with your cluster resource ID: FAKE_RESOURCE_ID",
			featureFlag:     true,
		},
		{
			name: "Delete existing ConsoleNotification",
			oldCN: consolev1.ConsoleNotification{
				ObjectMeta: metav1.ObjectMeta{
					Name: BannerName,
				},
				Spec: consolev1.ConsoleNotificationSpec{
					Text:            "OLD BANNER TEXT",
					Location:        consolev1.BannerTop,
					Color:           "#000",
					BackgroundColor: "#ff0",
				},
			},
			bannerSetting: "",
			expectBanner:  false,
			featureFlag:   true,
		},
		{
			name: "Support banner from existing CN",
			oldCN: consolev1.ConsoleNotification{
				ObjectMeta: metav1.ObjectMeta{
					Name: BannerName,
				},
				Spec: consolev1.ConsoleNotificationSpec{
					Text:            "OLD BANNER TEXT",
					Location:        consolev1.BannerTop,
					Color:           "#000",
					BackgroundColor: "#ff0",
				},
			},
			bannerSetting:   "ContactSupport",
			expectBanner:    true,
			expectedMessage: "We have noticed an issue regarding your cluster requiring an action on your part. Please contact support with your cluster resource ID: FAKE_RESOURCE_ID",
			featureFlag:     true,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			instance := arov1alpha1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: "cluster",
				},
				Spec: arov1alpha1.ClusterSpec{
					ResourceID: "FAKE_RESOURCE_ID",
					Banner: arov1alpha1.Banner{
						Content: arov1alpha1.BannerContent(tt.bannerSetting),
					},
					Features: arov1alpha1.FeaturesSpec{
						ReconcileBanner: tt.featureFlag,
					},
				},
			}

			r.arocli = arofake.NewSimpleClientset(&instance)
			r.consolecli = consolefake.NewSimpleClientset(&tt.oldCN)

			// function under test
			_, err := r.Reconcile(context.Background(), ctrl.Request{})

			if err != nil {
				if tt.expectedErr == nil {
					t.Errorf("Got unexpected Error when running Reconcile: %s", err)
				}
				if !strings.EqualFold(err.Error(), tt.expectedErr.Error()) {
					t.Errorf("Expected Error %s, got %s when processing %s testcase", tt.expectedErr.Error(), err.Error(), tt.name)
				}
				return
			}
			resultBanner, err := r.consolecli.ConsoleV1().ConsoleNotifications().Get(context.Background(), BannerName, metav1.GetOptions{})
			if tt.expectBanner {
				if err != nil {
					t.Errorf("Got Error when checking ConsoleNotification: %s", err)
				}
				if !strings.EqualFold(resultBanner.Spec.Text, tt.expectedMessage) {
					t.Errorf("Expected banner with text '%s', got '%s'", tt.expectedMessage, resultBanner.Spec.Text)
				}
			} else {
				if err != nil && !kerrors.IsNotFound(err) {
					t.Errorf("Got Error when checking for empty ConsoleNotification: %s", err)
				}
				if err == nil || !kerrors.IsNotFound(err) || resultBanner != nil {
					t.Errorf("Expected not to get a ConsoleNotification, but it exists")
				}
			}

		})
	}
}
