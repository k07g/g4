package auth

import "context"

// Identity is the caller identity resolved from an access token.
type Identity struct {
	Sub   string
	Email string
}

type SignUpResult struct {
	UserSub       string
	UserConfirmed bool
}

type SignInResult struct {
	AccessToken  string
	IDToken      string
	RefreshToken string
	ExpiresIn    int32
}

// Provider is the authentication backend used by the API layer. CognitoClient
// is the production implementation; MemoryProvider is a local/test
// implementation that requires no AWS account.
type Provider interface {
	SignUp(ctx context.Context, email, password string) (*SignUpResult, error)
	ConfirmSignUp(ctx context.Context, email, code string) error
	SignIn(ctx context.Context, email, password string) (*SignInResult, error)
	SignOut(ctx context.Context, accessToken string) error
	GetUser(ctx context.Context, accessToken string) (*Identity, error)
	DeleteUser(ctx context.Context, accessToken string) error
}
