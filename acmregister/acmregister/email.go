package acmregister

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"
)

// Email describes an email type.
type Email string

// Split splits up the email into 2 parts.
func (e Email) Split() (username, hostname string, ok bool) {
	return strings.Cut(string(e), "@")
}

// Username returns the username part of the email.
func (e Email) Username() string {
	name, _, _ := e.Split()
	return name
}

// EmailHostsVerifier whitelists hosts allowed for the email.
type EmailHostsVerifier []string

func (h EmailHostsVerifier) Verify(email Email) error {
	_, host, ok := email.Split()
	if !ok {
		return errors.New("email missing @hostname.com")
	}

	for _, allow := range h {
		if allow == host {
			return nil
		}
	}

	return fmt.Errorf("unknown email host %q, must be within %s", host, h)
}

// String returns a label string.
func (h EmailHostsVerifier) String() string {
	switch len(h) {
	case 0:
		return ""
	case 1:
		return h[0]
	case 2:
		return h[0] + " or " + h[1]
	default:
		return strings.Join(
			h[:len(h)-1], ", ") +
			", or " +
			h[len(h)-1]
	}
}
