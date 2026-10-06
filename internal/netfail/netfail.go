// Package netfail words the errors of a request that never got an answer,
// which net/http reports as a method, a URL and a chain of syscalls.
package netfail

import (
	"context"
	"errors"
	"net"
	"net/url"
	"syscall"
)

// ErrUnreachable is what every error from Explain unwraps to.
var ErrUnreachable = errors.New("unreachable")

type unreachable struct{ host, reason string }

func (u unreachable) Error() string { return "could not reach " + u.host + ": " + u.reason }

func (u unreachable) Is(target error) bool { return target == ErrUnreachable }

// Explain says in a few words why host could not be reached.
func Explain(host string, err error) error {
	var dns *net.DNSError
	var urlErr *url.Error
	var opErr *net.OpError
	switch {
	case errors.Is(err, context.Canceled):
		return unreachable{host, "cancelled"}
	case errors.As(err, &dns), errors.Is(err, syscall.ENETUNREACH), errors.Is(err, syscall.EHOSTUNREACH):
		return unreachable{host, "check your internet connection"}
	case errors.Is(err, context.DeadlineExceeded), errors.As(err, &urlErr) && urlErr.Timeout():
		return unreachable{host, "it took too long to answer"}
	case errors.Is(err, syscall.ECONNREFUSED):
		return unreachable{host, "connection refused"}
	}

	// Anything else: the innermost reason, without the request around it.
	if errors.As(err, &urlErr) {
		err = urlErr.Err
	}
	if errors.As(err, &opErr) {
		err = opErr.Err
	}
	return unreachable{host, err.Error()}
}
