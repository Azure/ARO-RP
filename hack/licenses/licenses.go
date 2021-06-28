package main

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"bytes"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

var (
	goLicense = []byte(`// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.`)

	pythonLicense = []byte(`# Copyright (c) Microsoft Corporation.
# Licensed under the Apache License 2.0.`)
)

func applyGoLicense() error {
	return filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		switch path {
		case "pkg/client", "vendor", ".git":
			return filepath.SkipDir
		}

		if !strings.HasSuffix(path, ".go") {
			return nil
		}

		b, err := ioutil.ReadFile(path)
		if err != nil {
			return err
		}

		if bytes.Contains(b, []byte("DO NOT EDIT.")) {
			return nil
		}

		if !bytes.Contains(b, goLicense) {
			i := bytes.Index(b, []byte("package "))
			i += bytes.Index(b[i:], []byte("\n"))

			var bb []byte
			bb = append(bb, b[:i]...)
			bb = append(bb, []byte("\n\n")...)
			bb = append(bb, goLicense...)
			bb = append(bb, b[i:]...)

			err = ioutil.WriteFile(path, bb, 0666)
			if err != nil {
				return err
			}
		}

		return nil
	})
}

func applyPythonLicense() error {
	return filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if strings.HasPrefix(path, "pyenv") {
			return filepath.SkipDir
		}

		switch path {
		case "python/client", "vendor":
			return filepath.SkipDir
		}

		if !strings.HasSuffix(path, ".py") {
			return nil
		}

		b, err := ioutil.ReadFile(path)
		if err != nil {
			return err
		}

		if !bytes.Contains(b, pythonLicense) {
			var bb []byte

			if bytes.HasPrefix(b, []byte("#!")) {
				i := bytes.Index(b, []byte("\n"))

				bb = append(bb, b[:i]...)
				bb = append(bb, []byte("\n\n")...)
				bb = append(bb, pythonLicense...)
				bb = append(bb, b[i:]...)

			} else {
				bb = append(bb, pythonLicense...)
				bb = append(bb, []byte("\n\n")...)
				bb = append(bb, b...)
			}

			err = ioutil.WriteFile(path, bb, 0666)
			if err != nil {
				return err
			}
		}

		return nil
	})
}

func run() error {
	err := applyGoLicense()
	if err != nil {
		return err
	}

	return applyPythonLicense()
}

func main() {
	if err := run(); err != nil {
		panic(err)
	}
}
