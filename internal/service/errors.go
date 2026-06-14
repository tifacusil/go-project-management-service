package service

import "fmt"

// Kind classifies a service error so handlers map it to an HTTP status
// without importing net/http into the service layer.
type Kind int

const (
	KindInvalid  Kind = iota // 400 — bad input / business rule violation
	KindNotFound             // 404 — resource missing
	KindConflict             // 400 — duplicate (Java baseline returns 400, not 409)
)

// Error is a service-layer error carrying a Kind + human message.
type Error struct {
	Kind Kind
	Msg  string
}

func (e *Error) Error() string { return e.Msg }

func invalid(format string, a ...any) *Error {
	return &Error{Kind: KindInvalid, Msg: fmt.Sprintf(format, a...)}
}

func notFound(format string, a ...any) *Error {
	return &Error{Kind: KindNotFound, Msg: fmt.Sprintf(format, a...)}
}

func conflict(format string, a ...any) *Error {
	return &Error{Kind: KindConflict, Msg: fmt.Sprintf(format, a...)}
}
