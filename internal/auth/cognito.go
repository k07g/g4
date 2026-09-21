package auth

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider/types"
)

// CognitoClient wraps the Cognito Identity Provider operations needed for
// sign-up, sign-in, sign-out and account deletion.
type CognitoClient struct {
	client       *cognitoidentityprovider.Client
	userPoolID   string
	clientID     string
	clientSecret string
}

func NewCognitoClient(ctx context.Context, region, userPoolID, clientID, clientSecret string) (*CognitoClient, error) {
	cfg, err := awsconfig.LoadDefaultConfig(ctx, awsconfig.WithRegion(region))
	if err != nil {
		return nil, err
	}

	return &CognitoClient{
		client:       cognitoidentityprovider.NewFromConfig(cfg),
		userPoolID:   userPoolID,
		clientID:     clientID,
		clientSecret: clientSecret,
	}, nil
}

// secretHash computes the Cognito SECRET_HASH parameter required when the
// app client is configured with a client secret. Returns nil when no
// secret is configured.
func (c *CognitoClient) secretHash(username string) *string {
	if c.clientSecret == "" {
		return nil
	}
	mac := hmac.New(sha256.New, []byte(c.clientSecret))
	mac.Write([]byte(username + c.clientID))
	h := base64.StdEncoding.EncodeToString(mac.Sum(nil))
	return aws.String(h)
}

var _ Provider = (*CognitoClient)(nil)

func (c *CognitoClient) SignUp(ctx context.Context, email, password string) (*SignUpResult, error) {
	out, err := c.client.SignUp(ctx, &cognitoidentityprovider.SignUpInput{
		ClientId: aws.String(c.clientID),
		Username: aws.String(email),
		Password: aws.String(password),
		UserAttributes: []types.AttributeType{
			{Name: aws.String("email"), Value: aws.String(email)},
		},
		SecretHash: c.secretHash(email),
	})
	if err != nil {
		return nil, err
	}
	return &SignUpResult{
		UserSub:       aws.ToString(out.UserSub),
		UserConfirmed: out.UserConfirmed,
	}, nil
}

func (c *CognitoClient) ConfirmSignUp(ctx context.Context, email, code string) error {
	_, err := c.client.ConfirmSignUp(ctx, &cognitoidentityprovider.ConfirmSignUpInput{
		ClientId:         aws.String(c.clientID),
		Username:         aws.String(email),
		ConfirmationCode: aws.String(code),
		SecretHash:       c.secretHash(email),
	})
	return err
}

func (c *CognitoClient) SignIn(ctx context.Context, email, password string) (*SignInResult, error) {
	authParams := map[string]string{
		"USERNAME": email,
		"PASSWORD": password,
	}
	if h := c.secretHash(email); h != nil {
		authParams["SECRET_HASH"] = *h
	}

	out, err := c.client.InitiateAuth(ctx, &cognitoidentityprovider.InitiateAuthInput{
		AuthFlow:       types.AuthFlowTypeUserPasswordAuth,
		ClientId:       aws.String(c.clientID),
		AuthParameters: authParams,
	})
	if err != nil {
		return nil, err
	}
	if out.AuthenticationResult == nil {
		return nil, ErrChallengeRequired
	}

	r := out.AuthenticationResult
	return &SignInResult{
		AccessToken:  aws.ToString(r.AccessToken),
		IDToken:      aws.ToString(r.IdToken),
		RefreshToken: aws.ToString(r.RefreshToken),
		ExpiresIn:    r.ExpiresIn,
	}, nil
}

// SignOut invalidates all tokens issued to the user identified by the given
// access token.
func (c *CognitoClient) SignOut(ctx context.Context, accessToken string) error {
	_, err := c.client.GlobalSignOut(ctx, &cognitoidentityprovider.GlobalSignOutInput{
		AccessToken: aws.String(accessToken),
	})
	return err
}

// GetUser resolves the caller identity for a given access token, used by
// the auth middleware to authenticate incoming requests.
func (c *CognitoClient) GetUser(ctx context.Context, accessToken string) (*Identity, error) {
	out, err := c.client.GetUser(ctx, &cognitoidentityprovider.GetUserInput{
		AccessToken: aws.String(accessToken),
	})
	if err != nil {
		return nil, err
	}

	identity := &Identity{}
	for _, attr := range out.UserAttributes {
		switch aws.ToString(attr.Name) {
		case "sub":
			identity.Sub = aws.ToString(attr.Value)
		case "email":
			identity.Email = aws.ToString(attr.Value)
		}
	}
	return identity, nil
}

// DeleteUser permanently deletes the user identified by the given access
// token from the Cognito user pool.
func (c *CognitoClient) DeleteUser(ctx context.Context, accessToken string) error {
	_, err := c.client.DeleteUser(ctx, &cognitoidentityprovider.DeleteUserInput{
		AccessToken: aws.String(accessToken),
	})
	return err
}

// ForgotPassword requests that Cognito send a password-reset confirmation
// code to the user's verified email address.
func (c *CognitoClient) ForgotPassword(ctx context.Context, email string) error {
	_, err := c.client.ForgotPassword(ctx, &cognitoidentityprovider.ForgotPasswordInput{
		ClientId:   aws.String(c.clientID),
		Username:   aws.String(email),
		SecretHash: c.secretHash(email),
	})
	return err
}

// ConfirmForgotPassword sets a new password using the code sent by
// ForgotPassword.
func (c *CognitoClient) ConfirmForgotPassword(ctx context.Context, email, code, newPassword string) error {
	_, err := c.client.ConfirmForgotPassword(ctx, &cognitoidentityprovider.ConfirmForgotPasswordInput{
		ClientId:         aws.String(c.clientID),
		Username:         aws.String(email),
		ConfirmationCode: aws.String(code),
		Password:         aws.String(newPassword),
		SecretHash:       c.secretHash(email),
	})
	return err
}
