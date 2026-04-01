package dns

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"bytes"
	"text/template"

	ign3types "github.com/coreos/ignition/v2/config/v3_2/types"
	"github.com/vincent-petithory/dataurl"

	"github.com/Azure/ARO-RP/pkg/util/pointerutils"
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
			Overwrite: pointerutils.ToPtr(true),
			Path:      "/etc/NetworkManager/dispatcher.d/" + restartScriptFileName,
			User: ign3types.NodeUser{
				Name: pointerutils.ToPtr("root"),
			},
		},
		FileEmbedded1: ign3types.FileEmbedded1{
			Contents: ign3types.Resource{
				Source: pointerutils.ToPtr(dataurl.EncodeBytes(data)),
			},
			Mode: pointerutils.ToPtr(0o744),
		},
	}
}
