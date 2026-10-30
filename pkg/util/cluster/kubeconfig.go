package cluster

import (
	"context"
	"encoding/base64"
	"fmt"
)

func (c *Cluster) Kubeconfig(ctx context.Context) error {
	kubeconfig, err := c.openshiftclusters.ListAdminCredentials(ctx, c.Config.VnetResourceGroup, c.Config.ClusterName)
	if err != nil {
		return err
	}

	r, err := base64.StdEncoding.DecodeString(kubeconfig)
	if err != nil {
		return err
	}

	fmt.Println(string(r))
	return nil
}
