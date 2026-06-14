package service

import "time"

// pick* return the body value if set (non-nil), else the existing value.
// Mirrors Java's `body.getX() != null ? body.getX() : existing.getX()`.

func pickStr(body, existing *string) *string {
	if body != nil {
		return body
	}
	return existing
}

func pickF64(body, existing *float64) *float64 {
	if body != nil {
		return body
	}
	return existing
}

func pickTime(body, existing *time.Time) *time.Time {
	if body != nil {
		return body
	}
	return existing
}
