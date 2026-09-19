package auth

import "errors"

// ErrChallengeRequired is returned when Cognito responds to sign-in with an
// authentication challenge (e.g. NEW_PASSWORD_REQUIRED, MFA) instead of
// issuing tokens directly.
var ErrChallengeRequired = errors.New("authentication challenge required")
