package install

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/json"

	"github.com/ghodss/yaml"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"
)

func (i *Installer) fixCloudConfig(ctx context.Context) error {
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		cm, err := i.kubernetescli.CoreV1().ConfigMaps("openshift-config").Get("cloud-provider-config", metav1.GetOptions{})
		if err != nil {
			return err
		}

		var config map[string]interface{}
		err = json.Unmarshal([]byte(cm.Data["config"]), &config)
		if err != nil {
			return err
		}

		if _, ok := config["aadClientSecret"]; ok && config["aadClientSecret"].(string) != "" {
			i.log.Info("skip fixCloudConfig")
			return nil
		}

		s, err := i.kubernetescli.CoreV1().Secrets("kube-system").Get("azure-cloud-provider", metav1.GetOptions{})
		if err != nil {
			return err
		}

		// merge secret contents over configmap
		err = yaml.Unmarshal(s.Data["cloud-config"], &config)
		if err != nil {
			return err
		}

		b, err := json.MarshalIndent(config, "", "\t")
		if err != nil {
			return err
		}

		cm.Data["config"] = string(b)
		_, err = i.kubernetescli.CoreV1().ConfigMaps("openshift-config").Update(cm)
		if err != nil {
			i.log.Error(err)
			return err
		}

		return nil
	})
}
