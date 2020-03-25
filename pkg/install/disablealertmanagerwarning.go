package install

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"github.com/ghodss/yaml"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"
)

// disableAlertManagerWarning is a hack to disable the
// AlertmanagerReceiversNotConfigured warning added in 4.3.8.
func (i *Installer) disableAlertManagerWarning(ctx context.Context) error {
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		s, err := i.kubernetescli.CoreV1().Secrets("openshift-monitoring").Get("alertmanager-main", metav1.GetOptions{})
		if err != nil {
			return err
		}

		var am map[string]interface{}
		err = yaml.Unmarshal(s.Data["alertmanager.yaml"], &am)
		if err != nil {
			return err
		}

		for _, r := range am["receivers"].([]interface{}) {
			r := r.(map[string]interface{})
			if name, ok := r["name"]; !ok || name != "null" {
				continue
			}

			r["webhook_configs"] = []interface{}{
				map[string]interface{}{"url": "http://localhost:1234/"}, // dummy
			}
		}

		s.Data["alertmanager.yaml"], err = yaml.Marshal(am)
		if err != nil {
			return err
		}

		_, err = i.kubernetescli.CoreV1().Secrets("openshift-monitoring").Update(s)
		return err
	})
}
