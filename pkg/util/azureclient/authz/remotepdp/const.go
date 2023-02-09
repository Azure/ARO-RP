package remotepdp

const (
	modulename = "aro-pdpclient"
	// version is the semantic version of this module
	version = "0.0.1" //nolint
)

// AccessDecision possible returned values
const (
	Allowed    AccessDecision = "Allowed"
	NotAllowed AccessDecision = "NotAllowed"
	Denied     AccessDecision = "Denied"
)
