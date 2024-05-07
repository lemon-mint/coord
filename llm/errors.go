package llm

import "errors"

var (
	ErrUnknown         = errors.New("unknown error")
	ErrNoResponse      = errors.New("no response")
	ErrInvalidRequest  = errors.New("invalid request")
	ErrInvalidResponse = errors.New("invalid response")
	ErrAuthentication  = errors.New("authentication error")
	ErrPermission      = errors.New("permission error")
	ErrNotFound        = errors.New("not found")
	ErrRateLimit       = errors.New("rate limit error")
	ErrOverloaded      = errors.New("overloaded")
	ErrInternalServer  = errors.New("internal server error")
)
