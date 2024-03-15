package containerinstall

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/containers/podman/v4/pkg/bindings/containers"
	"github.com/containers/podman/v4/pkg/bindings/images"
	"github.com/containers/podman/v4/pkg/bindings/secrets"
	"github.com/containers/podman/v4/pkg/specgen"
	"github.com/opencontainers/runtime-spec/specs-go"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/util/steps"
)

var (
	devEnvVars = []string{
		"AZURE_FP_CLIENT_ID",
		"AZURE_RP_CLIENT_ID",
		"AZURE_RP_CLIENT_SECRET",
		"AZURE_SUBSCRIPTION_ID",
		"AZURE_TENANT_ID",
		"DOMAIN_NAME",
		"KEYVAULT_PREFIX",
		"LOCATION",
		"PROXY_HOSTNAME",
		"PULL_SECRET",
		"RESOURCEGROUP",
	}
)

func (m *manager) Install(ctx context.Context, sub *api.SubscriptionDocument, doc *api.OpenShiftClusterDocument, version *api.OpenShiftVersion) error {
	s := []steps.Step{
		steps.Action(func(context.Context) error {
			options := (&images.PullOptions{}).
				WithQuiet(true).
				WithPolicy("always").
				WithUsername(m.pullSecret.Username).
				WithPassword(m.pullSecret.Password)

			_, err := images.Pull(m.conn, version.Properties.InstallerPullspec, options)
			return err
		}),
		steps.Action(func(context.Context) error { return m.createSecrets(ctx, doc, sub) }),
		steps.Action(func(context.Context) error { return m.startContainer(ctx, version) }),
		steps.Condition(m.containerFinished, 60*time.Minute, false),
		steps.Action(m.cleanupContainers),
	}

	_, err := steps.Run(ctx, m.log, 10*time.Second, s, nil)
	if err != nil {
		return err
	}
	if !m.success {
		return fmt.Errorf("failed to install cluster")
	}
	return nil
}

func (m *manager) putSecret(secretName string) specgen.Secret {
	return specgen.Secret{
		Source: m.clusterUUID + "-" + secretName,
		Target: "/.azure/" + secretName,
		Mode:   0o644,
	}
}

func (m *manager) startContainer(ctx context.Context, version *api.OpenShiftVersion) error {
	s := specgen.NewSpecGenerator(version.Properties.InstallerPullspec, false)
	s.Name = "installer-" + m.clusterUUID

	s.Secrets = []specgen.Secret{
		m.putSecret("99_aro.json"),
		m.putSecret("99_sub.json"),
		m.putSecret("proxy.crt"),
		m.putSecret("proxy-client.crt"),
		m.putSecret("proxy-client.key"),
	}

	s.Env = map[string]string{
		"ARO_RP_MODE":               "development",
		"ARO_UUID":                  m.clusterUUID,
		"OPENSHIFT_INSTALL_INVOKER": "hive",
		"OPENSHIFT_INSTALL_RELEASE_IMAGE_OVERRIDE": version.Properties.OpenShiftPullspec,
	}

	for _, envvar := range devEnvVars {
		s.Env["ARO_"+envvar] = os.Getenv(envvar)
	}

	s.Mounts = append(s.Mounts, specs.Mount{
		Destination: "/.azure",
		Type:        "tmpfs",
		Source:      "",
	})
	s.WorkDir = "/.azure"
	s.Entrypoint = []string{"/bin/bash", "-c", "/bin/openshift-install create manifests && /bin/openshift-install create cluster"}

	_, err := runContainer(m.conn, m.log, s)
	return err
}

func (m *manager) containerFinished(context.Context) (bool, bool, error) {
	containerName := "installer-" + m.clusterUUID
	inspectData, err := containers.Inspect(m.conn, containerName, nil)
	if err != nil {
		return false, false, err
	}

	if inspectData.State.Status == "exited" || inspectData.State.Status == "stopped" {
		if inspectData.State.ExitCode != 0 {
			getContainerLogs(m.conn, m.log, containerName)
			return true, false, fmt.Errorf("container exited with %d", inspectData.State.ExitCode)
		}
		m.success = true
		return true, false, nil
	}
	return false, true, nil
}

func (m *manager) createSecrets(ctx context.Context, doc *api.OpenShiftClusterDocument, sub *api.SubscriptionDocument) error {
	encCluster, err := json.Marshal(doc.OpenShiftCluster)
	if err != nil {
		return err
	}
	_, err = secrets.Create(
		m.conn, bytes.NewBuffer(encCluster),
		(&secrets.CreateOptions{}).WithName(m.clusterUUID+"-99_aro.json"))
	if err != nil {
		return err
	}

	encSub, err := json.Marshal(sub.Subscription)
	if err != nil {
		return err
	}
	_, err = secrets.Create(
		m.conn, bytes.NewBuffer(encSub),
		(&secrets.CreateOptions{}).WithName(m.clusterUUID+"-99_sub.json"))
	if err != nil {
		return err
	}

	basepath := os.Getenv("ARO_CHECKOUT_PATH")
	if basepath == "" {
		// This assumes we are running from an ARO-RP checkout in development
		var err error
		_, curmod, _, _ := runtime.Caller(0)
		basepath, err = filepath.Abs(filepath.Join(filepath.Dir(curmod), "../.."))
		if err != nil {
			return err
		}
	}

	err = m.secretFromFile(filepath.Join(basepath, "secrets/proxy.crt"), "proxy.crt")
	if err != nil {
		return err
	}

	err = m.secretFromFile(filepath.Join(basepath, "secrets/proxy-client.crt"), "proxy-client.crt")
	if err != nil {
		return err
	}

	err = m.secretFromFile(filepath.Join(basepath, "secrets/proxy-client.key"), "proxy-client.key")
	if err != nil {
		return err
	}

	return nil
}

func (m *manager) secretFromFile(from, name string) error {
	f, err := os.Open(from)
	if err != nil {
		return err
	}

	_, err = secrets.Create(
		m.conn, f,
		(&secrets.CreateOptions{}).WithName(m.clusterUUID+"-"+name))
	return err
}

func (m *manager) cleanupContainers(ctx context.Context) error {
	containerName := "installer-" + m.clusterUUID

	if !m.success {
		m.log.Infof("cleaning up failed container %s", containerName)
		getContainerLogs(m.conn, m.log, containerName)
	}

	_, err := containers.Remove(
		m.conn, containerName,
		(&containers.RemoveOptions{}).WithForce(true).WithIgnore(true))
	if err != nil {
		m.log.Errorf("unable to remove container: %v", err)
	}

	for _, secretName := range []string{"99_aro.json", "99_sub.json", "proxy.crt", "proxy-client.crt", "proxy-client.key"} {
		err = secrets.Remove(m.conn, m.clusterUUID+"-"+secretName)
		if err != nil {
			m.log.Debugf("unable to remove secret %s: %v", m.clusterUUID+"-"+secretName, err)
		}
	}
	return nil
}
