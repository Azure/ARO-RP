package pullsecret

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Extract()", func() {
	It("correctly decodes a pullsecret", func() {
		pullSecret := "{\"auths\": {\"example.com\": {\"auth\": \"dGVzdHVzZXI6dGVzdHBhc3M=\"}}}"

		correctlyExtracted, err := Extract(pullSecret, "example.com")
		Expect(err).ToNot(HaveOccurred())
		Expect(correctlyExtracted).To(Equal(&UserPass{Username: "testuser", Password: "testpass"}))
	})

	It("errors if no pullsecret for that name exists", func() {
		pullSecret := "{\"auths\": {\"example.com\": {\"auth\": \"dGVzdHVzZXI6dGVzdHBhc3M=\"}}}"

		_, err := Extract(pullSecret, "missingexample.com")
		Expect(err).To(MatchError("missing 'missingexample.com' key in pullsecret"))
	})

	It("errors if the json is invalid", func() {
		_, err := Extract("\"", "example.com")
		Expect(err).To(MatchError("malformed pullsecret (invalid JSON)"))
	})

	It("errors if the base64 is invalid", func() {
		pullSecret := "{\"auths\": {\"example.com\": {\"auth\": \"5\"}}}"

		_, err := Extract(pullSecret, "example.com")
		Expect(err).To(MatchError("malformed auth token"))
	})

	It("errors if the base64 does not contain a username and password", func() {
		pullSecret := "{\"auths\": {\"example.com\": {\"auth\": \"c29tZXRoaW5nZWxzZQ==\"}}}"

		_, err := Extract(pullSecret, "example.com")
		Expect(err).To(MatchError("auth token not in format of username:password"))
	})

	It("errors if pullsecret has no auth key for domain", func() {
		pullSecret := "{\"auths\": {\"example.com\": {\"p\": \"d\"}}}"

		_, err := Extract(pullSecret, "example.com")
		Expect(err).To(MatchError("malformed pullsecret (no auth key)"))
	})
})

func TestPullSecret(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "PullSecret Suite")
}
