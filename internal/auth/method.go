package auth

import (
	"context"
	"crypto/subtle"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

// Method authenticates one kind of login (email/password, Nostr, ...).
// Register implementations so Service.Login can dispatch by name.
type Method interface {
	Name() string
	Authenticate(ctx context.Context, creds Credentials) (Identity, error)
}

// MethodFunc adapts a plain function to the Method interface.
type MethodFunc struct {
	name string
	fn   func(context.Context, Credentials) (Identity, error)
}

func (m MethodFunc) Name() string { return m.name }

func (m MethodFunc) Authenticate(ctx context.Context, c Credentials) (Identity, error) {
	return m.fn(ctx, c)
}

// methods is the registry of login methods by name.
var methods = map[string]Method{}

// Register adds a login method. Later registrations win.
func Register(m Method) {
	methods[m.Name()] = m
}

// constantTimeEquals reports whether a and b are equal without short-circuiting
// on the first differing byte (avoids timing side channels).
func constantTimeEquals(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

// RegisterPassword installs the email/password method. It accepts exactly the
// configured admin email and the bcrypt-hashed password. Every failure, wrong
// email or wrong password, returns the same generic error.
func RegisterPassword(email, passwordHash string) {
	Register(MethodFunc{
		name: "password",
		fn: func(_ context.Context, c Credentials) (Identity, error) {
			if !constantTimeEquals(strings.ToLower(c.Email), strings.ToLower(email)) {
				return Identity{}, ErrInvalidCredentials
			}
			if bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(c.Password)) != nil {
				return Identity{}, ErrInvalidCredentials
			}
			return Identity{UserID: "god", Method: "password"}, nil
		},
	})
}
