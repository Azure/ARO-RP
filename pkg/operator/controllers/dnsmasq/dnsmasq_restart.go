package dnsmasq

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"bytes"
	"text/template"

	"github.com/Azure/go-autorest/autorest/to"
	ign2types "github.com/coreos/ignition/config/v2_2/types"
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

func restartScriptIgnFile(data []byte) ign2types.File {
	return ign2types.File{
		Node: ign2types.Node{
			Filesystem: "root",
			Overwrite:  to.BoolPtr(true),
			Path:       "/etc/NetworkManager/dispatcher.d/" + restartScriptFileName,
			User: &ign2types.NodeUser{
				Name: "root",
			},
		},
		FileEmbedded1: ign2types.FileEmbedded1{
			Contents: ign2types.FileContents{
				Source: dataurl.EncodeBytes(data),
			},
			Mode: to.IntPtr(0744),
		},
	}
}
