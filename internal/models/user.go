package models

import "time"

// User is the application-level profile record, keyed by the Cognito
// "sub" claim rather than owning credentials itself.
type User struct {
	ID         string    `json:"id"`
	CognitoSub string    `json:"-"`
	Email      string    `json:"email"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}
