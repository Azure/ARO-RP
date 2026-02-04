package main

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"log"
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

func main() {
	ignoredGo := flag.String("ignored-go", "", "comma separated list of ignored directories")
	ignoredPython := flag.String("ignored-python", "", "comma separated list of ignored directories")
	dirs := flag.String("dirs", ".", "comma separated list of dirs to parse")
	validateOnly := flag.Bool("validate", false, "only validates and does not modify (for CI)")
	flag.Parse()

	if *validateOnly {
		err := validateLicenses(*ignoredGo, *ignoredPython, *dirs)
		if err != nil {
			os.Exit(1)
		}
		os.Exit(0)
	} else {
		if err := run(); err != nil {
			log.Fatalf("error while checking for licences. err: %s ", err)
		}
	}
}

func run() error {
	err := applyGoLicense()
	if err != nil {
		return err
	}

	return applyPythonLicense()
}

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

		b, err := os.ReadFile(path)
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

			err = os.WriteFile(path, bb, 0o666)
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
		case "python/client", "vendor", "python/az/aro/azext_aro/aaz":
			return filepath.SkipDir
		}

		if !strings.HasSuffix(path, ".py") {
			return nil
		}

		b, err := os.ReadFile(path)
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

			err = os.WriteFile(path, bb, 0o666)
			if err != nil {
				return err
			}
		}

		return nil
	})
}

// returns the lists of files that don't have a license but should
func validateGoLicenses(ignored map[string]bool, dirs []string) []string {
	unlicensedFiles := make([]string, 0)
	for _, dir := range dirs {
		filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if ignored[path] {
				return filepath.SkipDir
			}

			if !strings.HasSuffix(path, ".go") {
				return nil
			}

			b, err := os.ReadFile(path)
			if err != nil {
				return err
			}

			if bytes.Contains(b, []byte("DO NOT EDIT.")) {
				return nil
			}

			if !bytes.Contains(b, goLicense) {
				unlicensedFiles = append(unlicensedFiles, path)
			}
			return nil
		})
	}

	return unlicensedFiles
}

// returns the lists of files that don't have a license but should
func validatePythonLicenses(ignored map[string]bool, dirs []string) []string {
	unlicensedFiles := make([]string, 0)

	for _, dir := range dirs {
		filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if strings.HasPrefix(path, "pyenv") {
				return filepath.SkipDir
			}

			if ignored[path] {
				return filepath.SkipDir
			}

			if !strings.HasSuffix(path, ".py") {
				return nil
			}

			b, err := os.ReadFile(path)
			if err != nil {
				return err
			}

			if !bytes.Contains(b, pythonLicense) {
				unlicensedFiles = append(unlicensedFiles, path)
			}
			return nil
		})
	}

	return unlicensedFiles
}

func parseIgnored(commaSeparated string) map[string]bool {
	ignored := strings.Split(commaSeparated, ",")

	result := make(map[string]bool)
	for _, v := range ignored {
		result[v] = true
	}
	return result
}

func validateLicenses(ignoredGo, ignoredPython, dirs string) error {
	ignoredMapGo := parseIgnored(ignoredGo)
	ignoredMapPython := parseIgnored(ignoredPython)

	unlicensedGo := validateGoLicenses(ignoredMapGo, strings.Split(dirs, ","))

	unlicensedPython := validatePythonLicenses(ignoredMapPython, strings.Split(dirs, ","))

	for _, v := range unlicensedGo {
		fmt.Printf("%s does not have a license\n", v)
	}

	for _, v := range unlicensedPython {
		fmt.Printf("%s does not have a license\n", v)
	}

	if len(unlicensedGo) > 0 || len(unlicensedPython) > 0 {
		return errors.New("validation failed")
	}
	return nil
}
