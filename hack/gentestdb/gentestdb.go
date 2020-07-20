package main

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"flag"
	"os"
	"path"
	"path/filepath"
	"strings"
	"text/template"
)

func main() {

	template := flag.String("template", "", "path to template")
	flag.Parse()

	if *template == "" {
		panic(*template)
	}

	templatePath, err := filepath.Abs(*template)
	if err != nil {
		panic(err)
	}

	err = Generate(templatePath)

	if err != nil {
		panic(err)
	}

}

type db struct {
	Name              string
	DocumentInterface string
	UsesPartitions    bool
	UsesLeases        bool
}

func Generate(pathName string) error {

	databaseKinds := []db{
		{
			Name:              "OpenShiftClusters",
			DocumentInterface: "OpenShiftClusterDocument",
			UsesPartitions:    true,
			UsesLeases:        true,
		},
		{
			Name:              "Billing",
			DocumentInterface: "BillingDocument",
			UsesPartitions:    false,
			UsesLeases:        false,
		},
		{
			Name:              "Subscriptions",
			DocumentInterface: "SubscriptionDocument",
			UsesPartitions:    false,
			UsesLeases:        true,
		},
		{
			Name:              "AsyncOperations",
			DocumentInterface: "AsyncOperationDocument",
			UsesPartitions:    false,
			UsesLeases:        false,
		},
	}

	dir, filename := path.Split(pathName)
	tmpl, err := template.New(filename).ParseFiles(pathName)

	if err != nil {
		return err
	}

	for _, f := range databaseKinds {

		writer, err := os.Create(dir + "/zz_generated_" + strings.ToLower(f.Name) + ".go")
		if err != nil {
			return err
		}

		err = tmpl.Execute(writer, f)
		if err != nil {
			return err
		}

	}

	return nil

}
