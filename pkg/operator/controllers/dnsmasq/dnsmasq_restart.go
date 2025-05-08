package dnsmasq

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"bytes"
	"text/template"

	"github.com/Azure/go-autorest/autorest/to"
	ign3types "github.com/coreos/ignition/v2/config/v3_2/types"
	"github.com/vincent-petithory/dataurl"
)

const restartScriptFileName = "99-dnsmasq-restart"

func nmDispatcherRestartDnsmasq() ([]byte, error) {
	t := template.Must(template.New(restartScriptFileName).Parse(restartScript))
	buf := &bytes.Buffer{}

	err := t.ExecuteTemplate(buf, restartScriptFileName, nil)
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func restartScriptIgnFile(data []byte) ign3types.File {
	return ign3types.File{
		Node: ign3types.Node{
			Overwrite: to.BoolPtr(true),
			Path:      "/etc/NetworkManager/dispatcher.d/" + restartScriptFileName,
			User: ign3types.NodeUser{
				Name: to.StringPtr("root"),
			},
		},
		FileEmbedded1: ign3types.FileEmbedded1{
			Contents: ign3types.Resource{
				Source: to.StringPtr(dataurl.EncodeBytes(data)),
			},
			Mode: to.IntPtr(0744),
		},
	}
}
