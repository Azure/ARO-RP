package clientauthorizer

type all struct{}

func NewAll() ClientAuthorizer {
	return &all{}
}

func (all) IsAuthorized([]byte) bool {
	return true
}

func (all) IsReady() bool {
	return true
}
