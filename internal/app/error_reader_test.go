package app

import (
	"errors"
	"io"
)

// ErrorReader is a custom io.Reader that returns errors for testing.
type ErrorReader struct {
	data    []byte
	pos     int
	errAt   int // Return error after this many bytes
	err     error
	errSent bool
}

// NewErrorReader creates a reader that returns an error after reading errAt bytes.
func NewErrorReader(data string, errAt int, err error) *ErrorReader {
	return &ErrorReader{
		data:  []byte(data),
		pos:   0,
		errAt: errAt,
		err:   err,
	}
}

func (r *ErrorReader) Read(p []byte) (n int, err error) {
	if r.errSent {
		return 0, io.EOF
	}

	if r.pos >= len(r.data) {
		if r.err != nil && !r.errSent {
			r.errSent = true
			return 0, r.err
		}
		return 0, io.EOF
	}

	remaining := len(r.data) - r.pos
	toRead := len(p)
	if toRead > remaining {
		toRead = remaining
	}

	// Check if we should return error
	if r.errAt >= 0 && r.pos+toRead >= r.errAt {
		// Copy up to errAt, then return error on next read
		toRead = r.errAt - r.pos
		if toRead <= 0 {
			r.errSent = true
			return 0, r.err
		}
	}

	copy(p, r.data[r.pos:r.pos+toRead])
	r.pos += toRead
	return toRead, nil
}

// ErrMockRead is a test error for reader failures.
var ErrMockRead = errors.New("mock read error")
