package main

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"bytes"
	"context"
	"embed"
	"fmt"
	"text/template"

	"github.com/sirupsen/logrus"

	utillog "github.com/Azure/ARO-RP/pkg/util/log"
)

//go:embed staticresources
var embeddedFiles embed.FS

func run(ctx context.Context, log *logrus.Entry) error {
	templatesRoot, err := template.ParseFS(embeddedFiles, "staticresources/*.yaml")
	if err != nil {
		return err
	}

	for x := range 100 {
		data := map[string]string{
			"NamespaceName": fmt.Sprintf("test-%d", x),
		}

		buff := &bytes.Buffer{}
		err = templatesRoot.ExecuteTemplate(buff, "namespace.yaml", data)
		if err != nil {
			return err
		}

		fmt.Print(buff.String())
		fmt.Println("---")

		for i := range 100 {
			buff.Reset()
			data["DeploymentNumber"] = fmt.Sprintf("%d", i)
			err = templatesRoot.ExecuteTemplate(buff, "deployment.yaml", data)
			if err != nil {
				return err
			}

			fmt.Print(buff.String())
			fmt.Println("---")
		}
	}
	return nil
}

func main() {
	log := utillog.GetLogger()

	if err := run(context.Background(), log); err != nil {
		log.Fatal(err)
	}
}
