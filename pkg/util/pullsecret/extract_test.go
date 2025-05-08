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

		correctlyExtracted, err := Extract(pullSecret)
		Expect(err).ToNot(HaveOccurred())
		extractedUserMap, ok := correctlyExtracted["example.com"]
		Expect(ok).To(BeTrue())
		Expect(extractedUserMap).To(Equal(&UserPass{Username: "testuser", Password: "testpass"}))
	})

	It("errors if no pullsecret for that name exists", func() {
		pullSecret := "{\"auths\": {\"example.com\": {\"auth\": \"dGVzdHVzZXI6dGVzdHBhc3M=\"}}}"

		correctlyExtracted, err := Extract(pullSecret)
		Expect(err).ToNot(HaveOccurred())
		_, ok := correctlyExtracted["missingexample.com"]
		Expect(ok).To(BeFalse())
	})

	It("errors if the json is invalid", func() {
		_, err := Extract("\"")
		Expect(err).To(MatchError("malformed pullsecret (invalid JSON)"))
	})

	It("errors if the base64 is invalid", func() {
		pullSecret := "{\"auths\": {\"example.com\": {\"auth\": \"5\"}}}"

		_, err := Extract(pullSecret)
		Expect(err).To(MatchError("malformed auth token for key example.com: invalid Base64"))
	})

	It("errors if the base64 does not contain a username and password", func() {
		pullSecret := "{\"auths\": {\"example.com\": {\"auth\": \"c29tZXRoaW5nZWxzZQ==\"}}}"

		_, err := Extract(pullSecret)
		Expect(err).To(MatchError("malformed auth token for key example.com: not in format of username:password"))
	})

	It("errors if pullsecret has no auth key for domain", func() {
		pullSecret := "{\"auths\": {\"example.com\": {\"p\": \"d\"}}}"

		_, err := Extract(pullSecret)
		Expect(err).To(MatchError("malformed pullsecret (no auth key) for key example.com"))
	})
})

func TestPullSecret(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "PullSecret Suite")
}
