package auth

import (
	"context"
	"errors"
	"testing"
)

const testEmail = "user@example.com"
const testPassword = "Passw0rd!123"

func confirmedUser(t *testing.T, p *MemoryProvider) *SignUpResult {
	t.Helper()

	result, err := p.SignUp(context.Background(), testEmail, testPassword)
	if err != nil {
		t.Fatalf("SignUp returned error: %v", err)
	}
	if err := p.ConfirmSignUp(context.Background(), testEmail, memoryFixedConfirmationCode); err != nil {
		t.Fatalf("ConfirmSignUp returned error: %v", err)
	}
	return result
}

func TestMemoryProvider_SignUp(t *testing.T) {
	p := NewMemoryProvider()

	result, err := p.SignUp(context.Background(), testEmail, testPassword)
	if err != nil {
		t.Fatalf("SignUp returned error: %v", err)
	}
	if result.UserSub == "" {
		t.Error("expected a non-empty UserSub")
	}
	if result.UserConfirmed {
		t.Error("expected UserConfirmed to be false immediately after sign-up")
	}

	if _, err := p.SignUp(context.Background(), testEmail, testPassword); !errors.Is(err, ErrUserExists) {
		t.Errorf("SignUp with duplicate email: got %v, want %v", err, ErrUserExists)
	}
}

func TestMemoryProvider_ConfirmSignUp(t *testing.T) {
	p := NewMemoryProvider()

	if err := p.ConfirmSignUp(context.Background(), "missing@example.com", memoryFixedConfirmationCode); !errors.Is(err, ErrUserNotFound) {
		t.Errorf("ConfirmSignUp for unknown user: got %v, want %v", err, ErrUserNotFound)
	}

	if _, err := p.SignUp(context.Background(), testEmail, testPassword); err != nil {
		t.Fatalf("SignUp returned error: %v", err)
	}

	if err := p.ConfirmSignUp(context.Background(), testEmail, "wrong-code"); !errors.Is(err, ErrInvalidCode) {
		t.Errorf("ConfirmSignUp with wrong code: got %v, want %v", err, ErrInvalidCode)
	}

	if err := p.ConfirmSignUp(context.Background(), testEmail, memoryFixedConfirmationCode); err != nil {
		t.Errorf("ConfirmSignUp with correct code returned error: %v", err)
	}
}

func TestMemoryProvider_SignIn(t *testing.T) {
	p := NewMemoryProvider()

	if _, err := p.SignIn(context.Background(), testEmail, testPassword); !errors.Is(err, ErrInvalidCredentials) {
		t.Errorf("SignIn for unknown user: got %v, want %v", err, ErrInvalidCredentials)
	}

	if _, err := p.SignUp(context.Background(), testEmail, testPassword); err != nil {
		t.Fatalf("SignUp returned error: %v", err)
	}

	if _, err := p.SignIn(context.Background(), testEmail, testPassword); !errors.Is(err, ErrNotConfirmed) {
		t.Errorf("SignIn before confirmation: got %v, want %v", err, ErrNotConfirmed)
	}

	if err := p.ConfirmSignUp(context.Background(), testEmail, memoryFixedConfirmationCode); err != nil {
		t.Fatalf("ConfirmSignUp returned error: %v", err)
	}

	if _, err := p.SignIn(context.Background(), testEmail, "wrong-password"); !errors.Is(err, ErrInvalidCredentials) {
		t.Errorf("SignIn with wrong password: got %v, want %v", err, ErrInvalidCredentials)
	}

	result, err := p.SignIn(context.Background(), testEmail, testPassword)
	if err != nil {
		t.Fatalf("SignIn returned error: %v", err)
	}
	if result.AccessToken == "" || result.RefreshToken == "" {
		t.Errorf("expected non-empty tokens, got %+v", result)
	}
	if result.ExpiresIn <= 0 {
		t.Errorf("expected a positive ExpiresIn, got %d", result.ExpiresIn)
	}
}

func TestMemoryProvider_GetUser(t *testing.T) {
	p := NewMemoryProvider()
	signUp := confirmedUser(t, p)

	if _, err := p.GetUser(context.Background(), "not-a-real-token"); !errors.Is(err, ErrInvalidToken) {
		t.Errorf("GetUser with invalid token: got %v, want %v", err, ErrInvalidToken)
	}

	signIn, err := p.SignIn(context.Background(), testEmail, testPassword)
	if err != nil {
		t.Fatalf("SignIn returned error: %v", err)
	}

	identity, err := p.GetUser(context.Background(), signIn.AccessToken)
	if err != nil {
		t.Fatalf("GetUser returned error: %v", err)
	}
	if identity.Sub != signUp.UserSub {
		t.Errorf("identity.Sub = %q, want %q", identity.Sub, signUp.UserSub)
	}
	if identity.Email != testEmail {
		t.Errorf("identity.Email = %q, want %q", identity.Email, testEmail)
	}
}

func TestMemoryProvider_SignOut(t *testing.T) {
	p := NewMemoryProvider()
	confirmedUser(t, p)

	signIn, err := p.SignIn(context.Background(), testEmail, testPassword)
	if err != nil {
		t.Fatalf("SignIn returned error: %v", err)
	}

	if err := p.SignOut(context.Background(), signIn.AccessToken); err != nil {
		t.Fatalf("SignOut returned error: %v", err)
	}

	if _, err := p.GetUser(context.Background(), signIn.AccessToken); !errors.Is(err, ErrInvalidToken) {
		t.Errorf("GetUser after sign-out: got %v, want %v", err, ErrInvalidToken)
	}

	if err := p.SignOut(context.Background(), signIn.AccessToken); !errors.Is(err, ErrInvalidToken) {
		t.Errorf("SignOut on an already-signed-out token: got %v, want %v", err, ErrInvalidToken)
	}
}

func TestMemoryProvider_DeleteUser(t *testing.T) {
	p := NewMemoryProvider()
	confirmedUser(t, p)

	// Sign in twice so the user holds two independent access tokens.
	firstSignIn, err := p.SignIn(context.Background(), testEmail, testPassword)
	if err != nil {
		t.Fatalf("SignIn returned error: %v", err)
	}
	secondSignIn, err := p.SignIn(context.Background(), testEmail, testPassword)
	if err != nil {
		t.Fatalf("SignIn returned error: %v", err)
	}

	if err := p.DeleteUser(context.Background(), firstSignIn.AccessToken); err != nil {
		t.Fatalf("DeleteUser returned error: %v", err)
	}

	// Deleting the user must invalidate every token issued to them, not
	// just the one used to authenticate the delete request.
	if _, err := p.GetUser(context.Background(), secondSignIn.AccessToken); !errors.Is(err, ErrInvalidToken) {
		t.Errorf("GetUser with a second, unused token after delete: got %v, want %v", err, ErrInvalidToken)
	}

	if err := p.DeleteUser(context.Background(), firstSignIn.AccessToken); !errors.Is(err, ErrInvalidToken) {
		t.Errorf("DeleteUser with an already-invalidated token: got %v, want %v", err, ErrInvalidToken)
	}

	// The email should be free to sign up again.
	if _, err := p.SignUp(context.Background(), testEmail, testPassword); err != nil {
		t.Errorf("SignUp after delete returned error: %v", err)
	}
}
