package containerinstall

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/Azure/go-autorest/autorest/to"
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

type auths struct {
	Auths map[string]map[string]interface{} `json:"auths,omitempty"`
}

func (m *manager) Install(ctx context.Context, sub *api.SubscriptionDocument, doc *api.OpenShiftClusterDocument, version *api.OpenShiftVersion) error {
	m.sub = sub
	m.doc = doc
	m.version = version

	pullSecrets := &auths{}
	err := json.Unmarshal([]byte(os.Getenv("PULL_SECRET")), pullSecrets)
	if err != nil {
		return err
	}

	auth, ok := pullSecrets.Auths[m.env.ACRDomain()]
	if !ok {
		return fmt.Errorf("missing %s key in PULL_SECRET", m.env.ACRDomain())
	}

	token, ok := auth["auth"]
	if !ok {
		return errors.New("maformed auth token")
	}

	decoded, err := base64.StdEncoding.DecodeString(token.(string))
	if err != nil {
		return err
	}

	split := strings.Split(string(decoded), ":")
	if len(split) != 2 {
		return fmt.Errorf("not username:pass in %s config", m.env.ACRDomain())
	}

	s := []steps.Step{
		steps.Action(func(context.Context) error {
			options := &images.PullOptions{
				Quiet:    to.BoolPtr(true),
				Policy:   to.StringPtr("always"),
				Username: to.StringPtr(split[0]),
				Password: to.StringPtr(split[1]),
			}

			return pullContainer(m.conn, m.version.Properties.InstallerPullspec, options)
		}),
		steps.Action(m.writeFiles),
		steps.Action(m.startContainer),
		steps.Condition(m.containerFinished, 60*time.Minute, false),
		steps.Action(m.cleanupContainers),
	}

	_, err = steps.Run(ctx, m.log, 10*time.Second, s, nil)
	if err != nil {
		return err
	}
	if !m.success {
		return fmt.Errorf("failed to install cluster")
	}
	return nil
}

func (m *manager) putSecret(secretName string) specgen.Secret {
	uid := uint32(os.Getuid())
	gid := uint32(os.Getgid())
	return specgen.Secret{
		Source: m.doc.ID + "-" + secretName,
		Target: "/.azure/" + secretName,
		UID:    uid,
		GID:    gid,
		Mode:   0o644,
	}
}

func (m *manager) startContainer(ctx context.Context) error {
	s := specgen.NewSpecGenerator(m.version.Properties.InstallerPullspec, false)
	s.Name = "installer-" + m.doc.ID
	s.User = fmt.Sprintf("%d", os.Getuid())

	s.Secrets = []specgen.Secret{
		m.putSecret("99_aro.json"),
		m.putSecret("99_sub.json"),
		m.putSecret("proxy.crt"),
		m.putSecret("proxy-client.crt"),
		m.putSecret("proxy-client.key"),
	}

	s.Env = map[string]string{
		"ARO_RP_MODE":               "development",
		"ARO_UUID":                  m.doc.ID,
		"OPENSHIFT_INSTALL_INVOKER": "hive",
		"OPENSHIFT_INSTALL_RELEASE_IMAGE_OVERRIDE": m.version.Properties.OpenShiftPullspec,
	}

	for _, i := range devEnvVars {
		s.Env["ARO_"+i] = os.Getenv(i)
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

func (m *manager) containerFinished(context.Context) (bool, error) {
	containerName := "installer-" + m.doc.ID
	inspectData, err := containers.Inspect(m.conn, containerName, nil)
	if err != nil {
		return false, err
	}

	if inspectData.State.Status == "exited" || inspectData.State.Status == "stopped" {
		if inspectData.State.ExitCode != 0 {
			getContainerLogs(m.conn, m.log, containerName)
			return true, fmt.Errorf("container exited with %d", inspectData.State.ExitCode)
		} else {
			m.success = true
			return true, nil
		}
	}
	return false, nil
}

func (m *manager) writeFiles(ctx context.Context) error {
	encCluster, err := json.Marshal(m.doc.OpenShiftCluster)
	if err != nil {
		return err
	}
	_, err = secrets.Create(m.conn, bytes.NewBuffer(encCluster), &secrets.CreateOptions{Name: to.StringPtr(m.doc.ID + "-99_aro.json")})
	if err != nil {
		return err
	}

	encSub, err := json.Marshal(m.sub.Subscription)
	if err != nil {
		return err
	}
	_, err = secrets.Create(m.conn, bytes.NewBuffer(encSub), &secrets.CreateOptions{Name: to.StringPtr(m.doc.ID + "-99_sub.json")})
	if err != nil {
		return err
	}

	if m.isDevelopment {
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

		err = m.secretFile(filepath.Join(basepath, "secrets/proxy.crt"), "proxy.crt")
		if err != nil {
			return err
		}

		err = m.secretFile(filepath.Join(basepath, "secrets/proxy-client.crt"), "proxy-client.crt")
		if err != nil {
			return err
		}

		err = m.secretFile(filepath.Join(basepath, "secrets/proxy-client.key"), "proxy-client.key")
		if err != nil {
			return err
		}
	}

	return nil
}

func (m *manager) secretFile(from, name string) error {
	f, err := os.Open(from)
	if err != nil {
		return err
	}

	_, err = secrets.Create(m.conn, f, &secrets.CreateOptions{Name: to.StringPtr(m.doc.ID + "-" + name)})
	return err
}

func (m *manager) cleanupContainers(ctx context.Context) error {
	containerName := "installer-" + m.doc.ID

	if !m.success {
		m.log.Info("cleaning up failed container %s", containerName)
		getContainerLogs(m.conn, m.log, containerName)
	}

	_, err := containers.Remove(m.conn, containerName, &containers.RemoveOptions{Force: to.BoolPtr(true), Ignore: to.BoolPtr(true)})
	if err != nil {
		m.log.Errorf("unable to remove container: %v", err)
	}

	for _, secretName := range []string{"99_aro.json", "99_sub.json", "proxy.crt", "proxy-client.crt", "proxy-client.key"} {
		err = secrets.Remove(m.conn, m.doc.ID+"-"+secretName)
		if err != nil {
			m.log.Errorf("unable to remove secret %s: %v", m.doc.ID+"-"+secretName, err)
		}
	}
	return nil
}
