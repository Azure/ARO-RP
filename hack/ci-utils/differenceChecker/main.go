package main

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"crypto/sha256"
	"fmt"
	"io/fs"
	"log"
	"os"
	"os/exec"
	"strings"
)

var ignoredDirectories map[string]bool = map[string]bool{"vendor": true, "pyenv": true, ".git": true}

func main() {
	if len(os.Args) < 2 {
		fmt.Printf("usage : \n%s command -arg1 -arg2\n", os.Args[0])
		os.Exit(1)
	}
	fmt.Println(os.Args)
	hashTreeBefore, err := readDir(".")

	if err != nil {
		log.Println(err)
	}

	cmd := exec.Command(os.Args[1], os.Args[2:]...)
	bytes, err := cmd.CombinedOutput()
	if err != nil || cmd.ProcessState.ExitCode() != 0 {
		log.Println(string(bytes))
		log.Fatal(err)
	}

	hashTreeAfter, err := readDir(".")
	if err != nil {
		log.Fatal(err)
	}

	if !equivalentTrees(hashTreeBefore, hashTreeAfter) {
		os.Exit(1)
	}
}

func equivalentTrees(first, second hashTree) bool {
	isEquivalent := true
	if len(first.tree) != len(second.tree) {
		isEquivalent = false
	}

	for key, bytes := range first.tree {
		secondBytes, ok := second.tree[key]
		delete(second.tree, key)
		if !ok {
			log.Printf("%s has been deleted after make generate", key)
			isEquivalent = false
			continue
		}
		if bytes != secondBytes {
			log.Printf("%s has been modified after make generate", key)
			isEquivalent = false
			continue
		}
	}
	for key := range second.tree {
		log.Printf("%s has been added after make generate", key)
	}

	return isEquivalent
}

func readDir(name string) (hashTree, error) {
	fileSystem := os.DirFS(name)
	hashTree := &hashTree{tree: make(map[string][sha256.Size]byte)}
	err := fs.WalkDir(fileSystem, ".", hashTree.update)
	if err != nil {
		return *hashTree, err
	}
	return *hashTree, nil
}

type hashTree struct {
	tree map[string][sha256.Size]byte
}

func (h *hashTree) update(path string, file fs.DirEntry, pathError error) error {
	if pathError != nil {
		return pathError
	}

	if file.IsDir() || !file.Type().IsRegular() {
		if ignoredDirectories[strings.TrimPrefix(path, "./")] {
			return fs.SkipDir
		}
		return nil
	}

	bytes, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	hash := sha256.Sum256(bytes)
	h.tree[path] = hash
	return nil
}
