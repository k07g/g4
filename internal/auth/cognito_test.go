package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"testing"
)

func TestCognitoClient_SecretHash(t *testing.T) {
	t.Run("no client secret configured", func(t *testing.T) {
		c := &CognitoClient{clientID: "client-id"}
		if got := c.secretHash("user@example.com"); got != nil {
			t.Errorf("secretHash() = %v, want nil", got)
		}
	})

	t.Run("computes HMAC-SHA256 of username+clientID", func(t *testing.T) {
		c := &CognitoClient{clientID: "client-id", clientSecret: "shh"}

		username := "user@example.com"
		mac := hmac.New(sha256.New, []byte("shh"))
		mac.Write([]byte(username + "client-id"))
		want := base64.StdEncoding.EncodeToString(mac.Sum(nil))

		got := c.secretHash(username)
		if got == nil {
			t.Fatal("secretHash() = nil, want a value")
		}
		if *got != want {
			t.Errorf("secretHash() = %q, want %q", *got, want)
		}
	})

	t.Run("differs per username", func(t *testing.T) {
		c := &CognitoClient{clientID: "client-id", clientSecret: "shh"}

		a := c.secretHash("alice@example.com")
		b := c.secretHash("bob@example.com")
		if *a == *b {
			t.Error("expected secretHash to differ between usernames")
		}
	})
}
