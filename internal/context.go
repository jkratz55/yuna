package internal

type ContextKey uint

const (
	ContextKeyLogger ContextKey = iota
	ContextKeyRestyTemplatedPath
	ContextKeyPrincipal
)
