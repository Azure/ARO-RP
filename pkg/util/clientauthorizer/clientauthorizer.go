package clientauthorizer

type ClientAuthorizer interface {
	IsAuthorized([]byte) bool
	IsReady() bool
}
