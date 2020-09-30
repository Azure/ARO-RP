//go:generate go run .

package main

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"bufio"
	"encoding/json"
	"flag"
	"io/ioutil"
	"os"
	"regexp"
)

type cgmanifest struct {
	Registrations []*registration `json:"registration,omitempty"`
	Version       int             `json:"version,omitempty"`
}

type registration struct {
	Component *typedComponent `json:"component,omitempty"`
}

type typedComponent struct {
	Type string       `json:"type,omitempty"`
	Go   *goComponent `json:"go,omitempty"`
}

type goComponent struct {
	Version string `json:"version,omitempty"`
	Name    string `json:"name,omitempty"`
}

var rx = regexp.MustCompile(`^# (?:.* => )?([^ ]+) ([^ ]+)$`)

func run() error {
	file, err := os.Open("../../vendor/modules.txt")
	if err != nil {
		return err
	}
	defer file.Close()

	cgmanifest := &cgmanifest{
		Version: 1,
	}

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		m := rx.FindStringSubmatch(scanner.Text())
		if m == nil {
			continue
		}

		cgmanifest.Registrations = append(cgmanifest.Registrations, &registration{
			Component: &typedComponent{
				Type: "go",
				Go: &goComponent{
					Name:    m[1],
					Version: m[2],
				},
			},
		})
	}
	if err := scanner.Err(); err != nil {
		return err
	}

	b, err := json.MarshalIndent(cgmanifest, "", "    ")
	if err != nil {
		return err
	}
	b = append(b, byte('\n'))

	return ioutil.WriteFile("../../cgmanifest.json", b, 0666)
}

func main() {
	flag.Parse()

	if err := run(); err != nil {
		panic(err)
	}
}
