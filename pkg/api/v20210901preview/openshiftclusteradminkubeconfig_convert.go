package v20210901preview

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"encoding/base64"
	"strings"

	"github.com/ghodss/yaml"
	"github.com/ugorji/go/codec"

	"github.com/Azure/ARO-RP/pkg/api"
)

type openShiftClusterAdminKubeconfigConverter struct{}

type kubeconfig struct {
	api.MissingFields
	Clusters []struct {
		api.MissingFields
		Cluster struct {
			api.MissingFields
			Server string `json:"server"`
			CAData string `json:"certificate-authority-data,omitempty"`
		} `json:"cluster"`
	} `json:"clusters"`
}

// openShiftClusterAdminKubeconfigConverter returns a new external representation
// of the internal object, reading from the subset of the internal object's
// fields that appear in the external representation.  ToExternal does not
// modify its argument; there is no pointer aliasing between the passed and
// returned objects.
func (*openShiftClusterAdminKubeconfigConverter) ToExternal(oc *api.OpenShiftCluster) interface{} {

	config := kubeconfig{}
	content, err := base64.StdEncoding.DecodeString(string(oc.Properties.UserAdminKubeconfig))
	if err != nil {
		panic(err) //TODO
	}
	jsondata, err := yaml.YAMLToJSON(content)
	if err != nil {
		panic(err) //TODO
	}
	err = codec.NewDecoderBytes(jsondata, new(codec.JsonHandle)).Decode(&config)
	if err != nil {
		panic(err) //TODO
	}
	for i := range config.Clusters {
		config.Clusters[i].Cluster.Server = strings.Replace(config.Clusters[i].Cluster.Server, "https://api-int.", "https://api.", 1)
		config.Clusters[i].Cluster.CAData = ""
	}
	err = codec.NewEncoderBytes(&jsondata, new(codec.JsonHandle)).Encode(config)
	if err != nil {
		panic(err) //TODO
	}
	output, err := yaml.JSONToYAML(jsondata)
	if err != nil {
		panic(err) //TODO
	}

	return &OpenShiftClusterAdminKubeconfig{
		Kubeconfig: api.SecureBytes(output),
	}
}
