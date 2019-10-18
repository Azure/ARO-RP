package restconfig

import (
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// RestConfig returns the Kubernetes *rest.Config for a kubeconfig
func RestConfig(b []byte) (*rest.Config, error) {
	config, err := clientcmd.Load(b)
	if err != nil {
		return nil, err
	}

	return clientcmd.NewDefaultClientConfig(*config, &clientcmd.ConfigOverrides{}).ClientConfig()
}
