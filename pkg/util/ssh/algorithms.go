package ssh

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	cryptossh "golang.org/x/crypto/ssh"
)

// These lists define the SSH algorithms used for portal SSH and bootstrap node
// diagnostics. They are aligned with FIPS 140-3 / SP 800-131A requirements
// ssh-rsa (SHA-1) is excluded: it is not FIPS 140-3 compliant and will
// panic under Go's FIPS mode. Clusters that only advertise ssh-rsa will
// fail to connect cleanly rather than panic.
// https://learn.microsoft.com/en-us/azure/governance/policy/samples/guest-configuration-baseline-linux
// https://liquid.microsoft.com/Web/Views/View/873720#Zrex-3A-2F-2Fsecurityconfigbaselines-2FRequirements-2Fbl-2E00250-2F

func KexAlgorithms() []string {
	return []string{
		cryptossh.KeyExchangeECDHP256,
		cryptossh.KeyExchangeECDHP384,
		cryptossh.KeyExchangeECDHP521,
		cryptossh.KeyExchangeDH14SHA256,
	}
}

func HostKeyAlgorithms() []string {
	return []string{
		cryptossh.KeyAlgoRSASHA512,
		cryptossh.KeyAlgoRSASHA256,
		cryptossh.KeyAlgoECDSA256,
		cryptossh.KeyAlgoECDSA384,
		cryptossh.KeyAlgoECDSA521,
	}
}

func Ciphers() []string {
	return []string{
		cryptossh.CipherAES256CTR,
		cryptossh.CipherAES192CTR,
		cryptossh.CipherAES128CTR,
	}
}

func MACs() []string {
	return []string{
		cryptossh.HMACSHA256ETM,
		cryptossh.HMACSHA512ETM,
		cryptossh.HMACSHA256,
		cryptossh.HMACSHA512,
	}
}

func PublicKeyAlgorithms() []string {
	return []string{
		cryptossh.KeyAlgoECDSA256,
		cryptossh.KeyAlgoECDSA384,
		cryptossh.KeyAlgoECDSA521,
		cryptossh.KeyAlgoRSASHA256,
		cryptossh.KeyAlgoRSASHA512,
	}
}
