package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"sync"
)

// memoryFixedConfirmationCode is the confirmation code accepted by
// MemoryProvider for every sign-up and password reset. Since there is no
// real email delivery in local development, a fixed, documented code lets
// ConfirmSignUp/ConfirmForgotPassword be exercised deterministically (e.g.
// from curl or a smoke-test script).
const memoryFixedConfirmationCode = "000000"

var (
	ErrUserExists         = errors.New("user already exists")
	ErrUserNotFound       = errors.New("user not found")
	ErrInvalidCode        = errors.New("invalid confirmation code")
	ErrNotConfirmed       = errors.New("user is not confirmed")
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrInvalidToken       = errors.New("invalid or expired access token")
)

// MemoryProvider is an in-memory Provider implementation for local
// development and automated tests, so the sign-up/confirm/sign-in/sign-out/
// delete flow can be exercised end-to-end without an AWS account. It mimics
// the Cognito behaviour this service depends on: sign-up requires
// confirmation before sign-in succeeds, and access tokens are invalidated
// on sign-out or account deletion. It is not safe for production use — data
// is lost on restart and there is no password hashing.
type MemoryProvider struct {
	mu            sync.Mutex
	usersByEmail  map[string]*memoryUser
	tokensToEmail map[string]string
}

type memoryUser struct {
	sub       string
	email     string
	password  string
	confirmed bool
	// resetCode is set by ForgotPassword and cleared once consumed by a
	// successful ConfirmForgotPassword. Empty means no reset is pending.
	resetCode string
}

func NewMemoryProvider() *MemoryProvider {
	return &MemoryProvider{
		usersByEmail:  make(map[string]*memoryUser),
		tokensToEmail: make(map[string]string),
	}
}

func (m *MemoryProvider) SignUp(_ context.Context, email, password string) (*SignUpResult, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.usersByEmail[email]; exists {
		return nil, ErrUserExists
	}

	sub := newMemoryToken()
	m.usersByEmail[email] = &memoryUser{
		sub:      sub,
		email:    email,
		password: password,
	}

	return &SignUpResult{UserSub: sub, UserConfirmed: false}, nil
}

func (m *MemoryProvider) ConfirmSignUp(_ context.Context, email, code string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	u, ok := m.usersByEmail[email]
	if !ok {
		return ErrUserNotFound
	}
	if code != memoryFixedConfirmationCode {
		return ErrInvalidCode
	}
	u.confirmed = true
	return nil
}

func (m *MemoryProvider) SignIn(_ context.Context, email, password string) (*SignInResult, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	u, ok := m.usersByEmail[email]
	if !ok || u.password != password {
		return nil, ErrInvalidCredentials
	}
	if !u.confirmed {
		return nil, ErrNotConfirmed
	}

	token := newMemoryToken()
	m.tokensToEmail[token] = email

	return &SignInResult{
		AccessToken:  token,
		IDToken:      token,
		RefreshToken: newMemoryToken(),
		ExpiresIn:    3600,
	}, nil
}

func (m *MemoryProvider) SignOut(_ context.Context, accessToken string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.tokensToEmail[accessToken]; !ok {
		return ErrInvalidToken
	}
	delete(m.tokensToEmail, accessToken)
	return nil
}

func (m *MemoryProvider) GetUser(_ context.Context, accessToken string) (*Identity, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	email, ok := m.tokensToEmail[accessToken]
	if !ok {
		return nil, ErrInvalidToken
	}
	u := m.usersByEmail[email]
	return &Identity{Sub: u.sub, Email: u.email}, nil
}

func (m *MemoryProvider) DeleteUser(_ context.Context, accessToken string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	email, ok := m.tokensToEmail[accessToken]
	if !ok {
		return ErrInvalidToken
	}
	delete(m.usersByEmail, email)
	for tok, em := range m.tokensToEmail {
		if em == email {
			delete(m.tokensToEmail, tok)
		}
	}
	return nil
}

// ForgotPassword issues a (fixed, local-only) reset code for the user. Like
// Cognito, a user must be confirmed before they can reset their password.
func (m *MemoryProvider) ForgotPassword(_ context.Context, email string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	u, ok := m.usersByEmail[email]
	if !ok {
		return ErrUserNotFound
	}
	if !u.confirmed {
		return ErrNotConfirmed
	}
	u.resetCode = memoryFixedConfirmationCode
	return nil
}

// ConfirmForgotPassword sets a new password if code matches the pending
// reset issued by ForgotPassword. An unknown email is reported as the same
// ErrInvalidCode as a wrong code, so this endpoint can't be used to probe
// which emails are registered.
func (m *MemoryProvider) ConfirmForgotPassword(_ context.Context, email, code, newPassword string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	u, ok := m.usersByEmail[email]
	if !ok || u.resetCode == "" || u.resetCode != code {
		return ErrInvalidCode
	}
	u.password = newPassword
	u.resetCode = ""
	return nil
}

func newMemoryToken() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

var _ Provider = (*MemoryProvider)(nil)
