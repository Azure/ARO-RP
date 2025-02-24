package msgs

// HeartbeatMsg is a message sent to the audit server with various endpoint information.
// This is sent periodically by the client and does not require a user to send it.
type HeartbeatMsg struct {
	// AuditVersion is the version of the audit client.
	AuditVersion string
	// OsVersion is the version of the operating system.
	OsVersion string
	// Language is the language the client is written in.
	Language string
	// Destination is the destination of the audit server.
	Destination string
}
