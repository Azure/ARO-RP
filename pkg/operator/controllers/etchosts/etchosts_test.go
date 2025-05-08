package etchosts

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGenerateEtcHostsAROConf(t *testing.T) {
	cases := []struct {
		name     string
		input    etcHostsAROConfTemplateData
		expected string
	}{
		{
			name: "generate aro.conf data",
			input: etcHostsAROConfTemplateData{
				ClusterDomain:            "test.com",
				APIIntIP:                 "10.10.10.10",
				GatewayDomains:           []string{"test2.com", "test3.com"},
				GatewayPrivateEndpointIP: "20.20.20.20",
			},
			expected: "10.10.10.10\tapi.test.com api-int.test.com\n20.20.20.20\ttest2.com test3.com\n",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			actual, _ := GenerateEtcHostsAROConf(tc.input.ClusterDomain, tc.input.APIIntIP,
				tc.input.GatewayDomains, tc.input.GatewayPrivateEndpointIP)
			assert.Equal(t, tc.expected, string(actual))
		})
	}
}

func TestGenerateEtcHostsAROScript(t *testing.T) {
	cases := []struct {
		name     string
		input    etcHostsAROScriptTemplateData
		expected string
	}{
		{
			name:     "generate aro-etchosts-resolver.sh",
			expected: "#!/bin/bash\nset -uo pipefail\n\ntrap 'jobs -p | xargs kill || true; wait; exit 0' TERM\n\nOPENSHIFT_MARKER=\"openshift-aro-etchosts-resolver\"\nHOSTS_FILE=\"/etc/hosts\"\nCONFIG_FILE=\"/etc/hosts.d/aro.conf\"\nTEMP_FILE=\"/etc/hosts.d/aro.tmp\"\n\n# Make a temporary file with the old hosts file's data.\nif ! cp -f \"${HOSTS_FILE}\" \"${TEMP_FILE}\"; then\n  echo \"Failed to preserve hosts file. Exiting.\"\n  exit 1\nfi\n\nif ! sed --silent \"/# ${OPENSHIFT_MARKER}/d; w ${TEMP_FILE}\" \"${HOSTS_FILE}\"; then\n  # Only continue rebuilding the hosts entries if its original content is preserved\n  sleep 60 & wait\n  continue\nfi\n\nwhile IFS= read -r line; do\n    echo \"${line} # ${OPENSHIFT_MARKER}\" >> \"${TEMP_FILE}\"\ndone < \"${CONFIG_FILE}\"\n\n# Replace /etc/hosts with our modified version if needed\ncmp \"${TEMP_FILE}\" \"${HOSTS_FILE}\" || cp -f \"${TEMP_FILE}\" \"${HOSTS_FILE}\"\n# TEMP_FILE is not removed to avoid file create/delete and attributes copy churn\n",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			actual, _ := GenerateEtcHostsAROScript()
			assert.Equal(t, tc.expected, string(actual))
		})
	}
}

func TestGenerateEtcHostsAROUnit(t *testing.T) {
	cases := []struct {
		name     string
		input    etcHostsAROScriptTemplateData
		expected string
	}{
		{
			name:     "generate aro-etchosts-resolver.service",
			expected: "[Unit]\nDescription=One shot service that appends static domains to etchosts\nBefore=network-online.target\n\n[Service]\n# ExecStart will copy the hosts defined in /etc/hosts.d/aro.conf to /etc/hosts\nExecStart=/bin/bash /usr/local/bin/aro-etchosts-resolver.sh\n\n[Install]\nWantedBy=multi-user.target\n",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			actual, _ := GenerateEtcHostsAROUnit()
			assert.Equal(t, tc.expected, actual)
		})
	}
}
