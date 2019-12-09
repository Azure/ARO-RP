package middleware

type contextKey int

const (
	ContextKeyLog contextKey = iota
	ContextKeyOriginalPath
	ContextKeyBody
)
