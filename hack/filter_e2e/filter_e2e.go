package main

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

// Use this by piping output to be filtered. It will be written back to standard out with any redactions applied.

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

const secretReplacement = "[REDACTED]"

func main() {
	remove, found := os.LookupEnv("E2E_VAR_NAMES_TO_REMOVE")
	if !found || remove == "" {
		fmt.Println("Error: must provide comma-separated, non-empy env variable E2E_VAR_NAMES_TO_REMOVE")
		os.Exit(1)
	}

	// Build a slice of variable names and their replacements e.g. ["secret", "redacted", "secret2", "redacted"]
	var replacements = []string{`\"`, `"`}
	for _, name := range strings.Split(remove, ",") {
		// Only replace env variables that exist
		value, found := os.LookupEnv(name)
		if found && value != "" {
			replacements = append(replacements, value, secretReplacement)
		}
	}
	r := strings.NewReplacer(replacements...)

	// Scan each line of piped in input
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		line := scanner.Text()
		line = r.Replace(line)
		fmt.Println(line)
	}
}
