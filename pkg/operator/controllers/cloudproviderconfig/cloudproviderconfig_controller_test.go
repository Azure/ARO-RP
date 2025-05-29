package cloudproviderconfig

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/json"
	"io"
	"testing"

	"github.com/sirupsen/logrus"
	logtest "github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	ctrl "sigs.k8s.io/controller-runtime"
	ctrlfake "sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/Azure/ARO-RP/pkg/operator"
	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	"github.com/Azure/ARO-RP/pkg/operator/controllers/base"
	_ "github.com/Azure/ARO-RP/pkg/util/scheme"
)

var (
	cmMetadata = metav1.ObjectMeta{Name: "cloud-provider-config", Namespace: "openshift-config"}
)

func TestReconcileCloudProviderConfig(t *testing.T) {
	type test struct {
		name        string
		configMap   *corev1.ConfigMap
		expectedLog *logrus.Entry
	}
	cpcNull := &azCloudProviderConfig{}
	jsonStringByteNull, _ := json.Marshal(&cpcNull)

	falsePointer := false
	cpcFalse := &azCloudProviderConfig{DisableOutboundSNAT: &falsePointer}
	jsonStringByteFalse, _ := json.Marshal(&cpcFalse)

	truePointer := true
	cpcTrue := &azCloudProviderConfig{DisableOutboundSNAT: &truePointer}
	jsonStringByteTrue, _ := json.Marshal(&cpcTrue)

	for _, tt := range []*test{
		{
			name:        "ConfigMap openshift-config/cloud-provider-config does not exist",
			expectedLog: &logrus.Entry{Level: logrus.ErrorLevel, Message: "configmaps \"cloud-provider-config\" not found"},
		},
		{
			name: "ConfigMap openshift-config/cloud-provider-config doesn't have config field",
			configMap: &corev1.ConfigMap{
				ObjectMeta: cmMetadata,
				Data: map[string]string{
					"notconfig": `{}`,
				},
			},
			expectedLog: &logrus.Entry{Level: logrus.ErrorLevel, Message: "field config in ConfigMap openshift-config/cloud-provider-config is missing"},
		},
		{
			name: "ConfigMap openshift-config/cloud-provider-config updated from disableOutboundSNAT: null",
			configMap: &corev1.ConfigMap{
				ObjectMeta: cmMetadata,
				Data: map[string]string{
					"config": string(jsonStringByteNull),
				},
			},
			expectedLog: &logrus.Entry{Level: logrus.InfoLevel, Message: "updating openshift-config/cloud-provider-config disableOutboundSNAT from nil to true"},
		},
		{
			name: "ConfigMap openshift-config/cloud-provider-config updated from disableOutboundSNAT: false",
			configMap: &corev1.ConfigMap{
				ObjectMeta: cmMetadata,
				Data: map[string]string{
					"config": string(jsonStringByteFalse),
				},
			},
			expectedLog: &logrus.Entry{Level: logrus.InfoLevel, Message: "updating openshift-config/cloud-provider-config disableOutboundSNAT from false to true"},
		},
		{
			name: "ConfigMap openshift-config/cloud-provider-config not updated from disableOutboundSNAT: true",
			configMap: &corev1.ConfigMap{
				ObjectMeta: cmMetadata,
				Data: map[string]string{
					"config": string(jsonStringByteTrue),
				},
			},
			expectedLog: &logrus.Entry{Level: logrus.DebugLevel, Message: "openshift-config/cloud-provider-config disableOutboundSNAT is set to true no changes needed"},
		},
	} {
		ctx := context.Background()

		logger := &logrus.Logger{
			Out:       io.Discard,
			Formatter: new(logrus.TextFormatter),
			Hooks:     make(logrus.LevelHooks),
			Level:     logrus.TraceLevel,
		}
		var hook = logtest.NewLocal(logger)

		instance := &arov1alpha1.Cluster{
			ObjectMeta: metav1.ObjectMeta{
				Name: arov1alpha1.SingletonClusterName,
			},
			Spec: arov1alpha1.ClusterSpec{
				OperatorFlags: arov1alpha1.OperatorFlags{
					operator.CloudProviderConfigEnabled: operator.FlagTrue,
				},
			},
		}

		clientBuilder := ctrlfake.NewClientBuilder().WithObjects(instance)
		if tt.configMap != nil {
			clientBuilder.WithObjects(tt.configMap)
		}

		r := &CloudProviderConfigReconciler{
			AROController: base.AROController{
				Log:    logrus.NewEntry(logger),
				Client: clientBuilder.Build(),
				Name:   ControllerName,
			},
		}
		request := ctrl.Request{}
		request.Name = "cloud-provider-config"
		request.Namespace = "openshift-config"

		_, err := r.Reconcile(ctx, request)
		if err != nil {
			logger.Log(logrus.ErrorLevel, err)
		}

		actualLog := hook.LastEntry()
		if actualLog == nil {
			assert.Equal(t, tt.expectedLog, actualLog)
		} else {
			assert.Equal(t, tt.expectedLog.Level.String(), actualLog.Level.String())
			assert.Equal(t, tt.expectedLog.Message, actualLog.Message)
		}
	}
}
