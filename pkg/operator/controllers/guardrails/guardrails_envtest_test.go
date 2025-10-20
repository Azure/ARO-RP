package guardrails

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/Azure/ARO-RP/pkg/operator"
	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	"github.com/Azure/ARO-RP/pkg/util/clienthelper"
	testlog "github.com/Azure/ARO-RP/test/util/log"
)

func SetupEnvtestDefaultBinaryAssetsDirectory() (string, error) {
	var baseDir string

	// find the base data directory
	switch runtime.GOOS {
	case "windows":
		baseDir = os.Getenv("LocalAppData")
		if baseDir == "" {
			return "", errors.New("%LocalAppData% is not defined")
		}
	case "darwin":
		homeDir := os.Getenv("HOME")
		if homeDir == "" {
			return "", errors.New("$HOME is not defined")
		}
		baseDir = filepath.Join(homeDir, "Library/Application Support")
	default:
		baseDir = os.Getenv("XDG_DATA_HOME")
		if baseDir == "" {
			homeDir := os.Getenv("HOME")
			if homeDir == "" {
				return "", errors.New("neither $XDG_DATA_HOME nor $HOME are defined")
			}
			baseDir = filepath.Join(homeDir, ".local/share")
		}
	}

	// append our program-specific dir to it (OSX has a slightly different
	// convention so try to follow that).
	switch runtime.GOOS {
	case "darwin", "ios":
		return filepath.Join(baseDir, "io.kubebuilder.envtest", "k8s", "1.25.0-darwin-amd64"), nil
	default:
		return filepath.Join(baseDir, "kubebuilder-envtest", "k8s", "1.25.0-linux-amd64"), nil
	}
}

var setupTime = time.Second * 70

var _ = Describe("Guardrails", Ordered, Serial, func() {
	var r *Reconciler
	var restConfig *rest.Config
	var k8sClient client.Client
	var _ch clienthelper.Interface
	var testEnv *envtest.Environment

	var log *logrus.Entry
	var hook *test.Hook

	BeforeAll(func(ctx SpecContext) {
		if os.Getenv("USE_ENVTEST") == "" {
			Skip("Not running tests using envtest, set USE_ENVTEST to enable")
		}

		var err error
		hook, log = testlog.New()

		dir, err := SetupEnvtestDefaultBinaryAssetsDirectory()
		Expect(err).ToNot(HaveOccurred())

		testEnv = &envtest.Environment{
			// aro.openshift.io CRDs
			CRDDirectoryPaths:     []string{filepath.Join("..", "..", "deploy", "staticresources")},
			BinaryAssetsDirectory: dir,
		}
		restConfig, err = testEnv.Start()
		Expect(err).ToNot(HaveOccurred())

		k8sClient, err = client.New(restConfig, client.Options{})
		Expect(err).ToNot(HaveOccurred())

		_ch = clienthelper.NewWithClient(log, k8sClient)

		cluster := &arov1alpha1.Cluster{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Cluster",
				APIVersion: "aro.openshift.io/v1alpha1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: arov1alpha1.SingletonClusterName,
			},
			Spec: arov1alpha1.ClusterSpec{
				OperatorFlags: arov1alpha1.OperatorFlags{
					operator.GuardrailsEnabled:       operator.FlagTrue,
					operator.GuardrailsDeployManaged: operator.FlagTrue,
					controllerPullSpec:               "wonderfulPullspec",
				},
				ACRDomain: "acrtest.example.com",
			},
		}

		err = _ch.Ensure(ctx, cluster)
		Expect(err).ToNot(HaveOccurred())
	}, NodeTimeout(setupTime))

	BeforeEach(func(ctx SpecContext) {
		// TODO: do some cleanup here
		r = NewReconciler(log, k8sClient)

		r.skipGatekeeperReadinessCheck = true
		r.skipPolicyDeployment = true
	})

	AfterAll(func() {
		testEnv.Stop()

		fmt.Println("logs:")
		for _, i := range hook.AllEntries() {
			fmt.Println(i)
		}
	})

	It("will create the deployment when run", func(ctx SpecContext) {
		_, err := r.Reconcile(ctx, reconcile.Request{})
		Expect(err).ToNot(HaveOccurred())

	}, SpecTimeout(setupTime+10*time.Second))

})

func TestGuardrails(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Guardrails Suite")
}
