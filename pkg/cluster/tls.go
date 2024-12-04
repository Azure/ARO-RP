package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/Azure/go-autorest/logger"
	configv1 "github.com/openshift/api/config/v1"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/util/dns"
	"github.com/Azure/ARO-RP/pkg/util/keyvault"
)

const (
	OneCertPublicIssuerName = "OneCertV2-PublicCA"
)

type writer struct {
	*manager
	logLevel logger.LevelType
}

func (w writer) Writeln(level logger.LevelType, message string) {
	w.Writef(level, "%s\n", message)
}
func (w writer) Writef(level logger.LevelType, format string, a ...interface{}) {
	if w.logLevel >= level {
		w.log.Log(logrus.InfoLevel, entryHeader(level), fmt.Sprintf(format, a...))
	}
}
func (w writer) WriteRequest(req *http.Request, filter logger.Filter) {
	if req == nil {
		return
	}
	b := &bytes.Buffer{}
	fmt.Fprintf(b, "%s REQUEST: %s %s\n", entryHeader(logger.LogInfo), req.Method, processURL(filter, req.URL))
	// dump headers
	for k, v := range req.Header {
		if ok, mv := processHeader(filter, k, v); ok {
			fmt.Fprintf(b, "%s: %s\n", k, strings.Join(mv, ","))
		}
	}
	if req.Body != nil && !strings.Contains(req.Header.Get("Content-Type"), "application/octet-stream") {
		// dump body
		body, err := ioutil.ReadAll(req.Body)
		if err == nil {
			fmt.Fprintln(b, string(processBody(filter, body)))
			if nc, ok := req.Body.(io.Seeker); ok {
				// rewind to the beginning
				nc.Seek(0, io.SeekStart)
			} else {
				// recreate the body
				req.Body = ioutil.NopCloser(bytes.NewReader(body))
			}
		} else {
			fmt.Fprintf(b, "failed to read body: %v\n", err)
		}
	}
	w.Writeln(logger.LogInfo, b.String())
}
func (w writer) WriteResponse(resp *http.Response, filter logger.Filter) {
	if resp == nil {
		return
	}
	b := &bytes.Buffer{}
	fmt.Fprintf(b, "%s RESPONSE: %d %s\n", entryHeader(logger.LogInfo), resp.StatusCode, processURL(filter, resp.Request.URL))
	// dump headers
	for k, v := range resp.Header {
		if ok, mv := processHeader(filter, k, v); ok {
			fmt.Fprintf(b, "%s: %s\n", k, strings.Join(mv, ","))
		}
	}
	if resp.Body != nil && !strings.Contains(resp.Header.Get("Content-Type"), "application/octet-stream") {
		// dump body
		defer resp.Body.Close()
		body, err := ioutil.ReadAll(resp.Body)
		if err == nil {
			fmt.Fprintln(b, string(processBody(filter, body)))
			resp.Body = ioutil.NopCloser(bytes.NewReader(body))
		} else {
			fmt.Fprintf(b, "failed to read body: %v\n", err)
		}
	}
	w.Writeln(logger.LogInfo, b.String())
}

func processURL(f logger.Filter, u *url.URL) string {
	if f.URL == nil {
		return u.String()
	}
	return f.URL(u)
}

func processHeader(f logger.Filter, k string, val []string) (bool, []string) {
	if f.Header == nil {
		return true, val
	}
	return f.Header(k, val)
}

func processBody(f logger.Filter, b []byte) []byte {
	if f.Body == nil {
		return b
	}
	return f.Body(b)
}

func entryHeader(level logger.LevelType) string {
	// this format provides a fixed number of digits so the size of the timestamp is constant
	return fmt.Sprintf("(%s) %s:", time.Now().Format("2006-01-02T15:04:05.0000000Z07:00"), level.String())
}

func (m *manager) createCertificates(ctx context.Context) error {
	managedDomain, err := dns.ManagedDomain(m.env, m.doc.OpenShiftCluster.Properties.ClusterProfile.Domain)
	if err != nil {
		return err
	}

	if managedDomain == "" {
		return nil
	}

	certs := []struct {
		certificateName string
		commonName      string
	}{
		{
			certificateName: m.APICertName(),
			commonName:      "api." + managedDomain,
		},
		{
			certificateName: m.IngressCertName(),
			commonName:      "*.apps." + managedDomain,
		},
	}

	logger.Instance = writer{manager: m, logLevel: logger.LogInfo}

	for _, c := range certs {
		m.log.Printf("creating certificate %s", c.certificateName)
		err = m.env.ClusterKeyvault().CreateSignedCertificate(ctx, OneCertPublicIssuerName, c.certificateName, c.commonName, keyvault.EkuServerAuth)
		if err != nil {
			return err
		}
	}

	for _, c := range certs {
		m.log.Printf("waiting for certificate %s", c.certificateName)
		wg := sync.WaitGroup{}
		for i := 0; i < 100; i++ {
			wg.Add(1)
			go func() {
				err = m.env.ClusterKeyvault().WaitForCertificateOperation(ctx, c.certificateName)
				if err != nil {
					m.log.Errorf("error when waiting for certificate %s: %s", c.certificateName, err.Error())
				}
				wg.Done()
			}()
		}
		wg.Wait()
		return api.NewCloudError(
			http.StatusBadRequest,
			"unimplemented-test",
			"target-test",
			"test",
		)
		//err = m.env.ClusterKeyvault().WaitForCertificateOperation(ctx, c.certificateName)
		//if err != nil {
		//	m.log.Errorf("error when waiting for certificate %s: %s", c.certificateName, err.Error())
		//	return err
		//}
	}

	return nil
}

func (m *manager) configureAPIServerCertificate(ctx context.Context) error {
	if m.env.FeatureIsSet(env.FeatureDisableSignedCertificates) {
		return nil
	}

	managedDomain, err := dns.ManagedDomain(m.env, m.doc.OpenShiftCluster.Properties.ClusterProfile.Domain)
	if err != nil {
		return err
	}

	if managedDomain == "" {
		return nil
	}

	for _, namespace := range []string{"openshift-config", "openshift-azure-operator"} {
		err = EnsureTLSSecretFromKeyvault(ctx, m.env.ClusterKeyvault(), m.ch, types.NamespacedName{Name: m.APICertName(), Namespace: namespace}, m.APICertName())
		if err != nil {
			return err
		}
	}

	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		apiserver, err := m.configcli.ConfigV1().APIServers().Get(ctx, "cluster", metav1.GetOptions{})
		if err != nil {
			return err
		}

		apiserver.Spec.ServingCerts.NamedCertificates = []configv1.APIServerNamedServingCert{
			{
				Names: []string{
					"api." + managedDomain,
				},
				ServingCertificate: configv1.SecretNameReference{
					Name: m.APICertName(),
				},
			},
		}

		_, err = m.configcli.ConfigV1().APIServers().Update(ctx, apiserver, metav1.UpdateOptions{})
		return err
	})
}

func (m *manager) configureIngressCertificate(ctx context.Context) error {
	if m.env.FeatureIsSet(env.FeatureDisableSignedCertificates) {
		return nil
	}

	managedDomain, err := dns.ManagedDomain(m.env, m.doc.OpenShiftCluster.Properties.ClusterProfile.Domain)
	if err != nil {
		return err
	}

	if managedDomain == "" {
		return nil
	}

	for _, namespace := range []string{"openshift-ingress", "openshift-azure-operator"} {
		err = EnsureTLSSecretFromKeyvault(ctx, m.env.ClusterKeyvault(), m.ch, types.NamespacedName{Namespace: namespace, Name: m.IngressCertName()}, m.IngressCertName())
		if err != nil {
			return err
		}
	}

	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		ic, err := m.operatorcli.OperatorV1().IngressControllers("openshift-ingress-operator").Get(ctx, "default", metav1.GetOptions{})
		if err != nil {
			return err
		}

		ic.Spec.DefaultCertificate = &corev1.LocalObjectReference{
			Name: m.IngressCertName(),
		}

		_, err = m.operatorcli.OperatorV1().IngressControllers("openshift-ingress-operator").Update(ctx, ic, metav1.UpdateOptions{})
		return err
	})
}
