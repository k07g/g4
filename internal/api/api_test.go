package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/k07g/g4/internal/auth"
	"github.com/k07g/g4/internal/db"
)

const (
	testEmail    = "user@example.com"
	testPassword = "Passw0rd!123"
	// MemoryProvider always issues this confirmation code locally.
	testConfirmationCode = "000000"
)

func newTestServer(t *testing.T) (http.Handler, sqlmock.Sqlmock) {
	t.Helper()

	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	t.Cleanup(func() { mockDB.Close() })

	provider := auth.NewMemoryProvider()
	userRepo := db.NewUserRepository(mockDB)
	handler := NewHandler(provider, userRepo)
	router := NewRouter(handler, AuthMiddleware(provider))

	return router, mock
}

func doRequest(t *testing.T, router http.Handler, method, path string, body any, token string) *httptest.ResponseRecorder {
	t.Helper()

	var reqBody *bytes.Buffer
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("failed to marshal request body: %v", err)
		}
		reqBody = bytes.NewBuffer(b)
	} else {
		reqBody = bytes.NewBuffer(nil)
	}

	req := httptest.NewRequest(method, path, reqBody)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func expectUserInsert(mock sqlmock.Sqlmock) {
	rows := sqlmock.NewRows([]string{"id", "cognito_sub", "email", "created_at", "updated_at"}).
		AddRow("11111111-1111-1111-1111-111111111111", "test-sub", testEmail, time.Now(), time.Now())
	mock.ExpectQuery("INSERT INTO users").WillReturnRows(rows)
}

func signUpConfirmAndSignIn(t *testing.T, router http.Handler, mock sqlmock.Sqlmock) string {
	t.Helper()

	expectUserInsert(mock)
	if rec := doRequest(t, router, http.MethodPost, "/auth/signup", map[string]string{
		"email": testEmail, "password": testPassword,
	}, ""); rec.Code != http.StatusCreated {
		t.Fatalf("signup: status = %d, body = %s", rec.Code, rec.Body.String())
	}

	if rec := doRequest(t, router, http.MethodPost, "/auth/confirm", map[string]string{
		"email": testEmail, "code": testConfirmationCode,
	}, ""); rec.Code != http.StatusNoContent {
		t.Fatalf("confirm: status = %d, body = %s", rec.Code, rec.Body.String())
	}

	rec := doRequest(t, router, http.MethodPost, "/auth/signin", map[string]string{
		"email": testEmail, "password": testPassword,
	}, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("signin: status = %d, body = %s", rec.Code, rec.Body.String())
	}

	var resp signInResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode signin response: %v", err)
	}
	if resp.AccessToken == "" {
		t.Fatal("expected a non-empty access token")
	}
	return resp.AccessToken
}

func TestHealthz(t *testing.T) {
	router, _ := newTestServer(t)

	rec := doRequest(t, router, http.MethodGet, "/healthz", nil, "")
	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestSignUp(t *testing.T) {
	t.Run("success persists the user profile", func(t *testing.T) {
		router, mock := newTestServer(t)
		expectUserInsert(mock)

		rec := doRequest(t, router, http.MethodPost, "/auth/signup", map[string]string{
			"email": testEmail, "password": testPassword,
		}, "")

		if rec.Code != http.StatusCreated {
			t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
		}
		var resp signUpResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		if resp.UserSub == "" {
			t.Error("expected a non-empty user_sub")
		}
		if resp.UserConfirmed {
			t.Error("expected user_confirmed to be false")
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("unmet db expectations: %v", err)
		}
	})

	t.Run("missing password is rejected before hitting the provider or db", func(t *testing.T) {
		router, mock := newTestServer(t)

		rec := doRequest(t, router, http.MethodPost, "/auth/signup", map[string]string{
			"email": testEmail,
		}, "")

		if rec.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("unmet db expectations: %v", err)
		}
	})

	t.Run("duplicate signup is rejected", func(t *testing.T) {
		router, mock := newTestServer(t)
		expectUserInsert(mock)

		doRequest(t, router, http.MethodPost, "/auth/signup", map[string]string{
			"email": testEmail, "password": testPassword,
		}, "")

		rec := doRequest(t, router, http.MethodPost, "/auth/signup", map[string]string{
			"email": testEmail, "password": testPassword,
		}, "")

		if rec.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
		}
	})
}

func TestConfirmSignUp(t *testing.T) {
	router, mock := newTestServer(t)
	expectUserInsert(mock)
	doRequest(t, router, http.MethodPost, "/auth/signup", map[string]string{
		"email": testEmail, "password": testPassword,
	}, "")

	t.Run("wrong code is rejected", func(t *testing.T) {
		rec := doRequest(t, router, http.MethodPost, "/auth/confirm", map[string]string{
			"email": testEmail, "code": "999999",
		}, "")
		if rec.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
		}
	})

	t.Run("correct code succeeds", func(t *testing.T) {
		rec := doRequest(t, router, http.MethodPost, "/auth/confirm", map[string]string{
			"email": testEmail, "code": testConfirmationCode,
		}, "")
		if rec.Code != http.StatusNoContent {
			t.Errorf("status = %d, want %d", rec.Code, http.StatusNoContent)
		}
	})
}

func TestSignIn(t *testing.T) {
	router, mock := newTestServer(t)
	expectUserInsert(mock)
	doRequest(t, router, http.MethodPost, "/auth/signup", map[string]string{
		"email": testEmail, "password": testPassword,
	}, "")

	t.Run("before confirmation is rejected", func(t *testing.T) {
		rec := doRequest(t, router, http.MethodPost, "/auth/signin", map[string]string{
			"email": testEmail, "password": testPassword,
		}, "")
		if rec.Code != http.StatusUnauthorized {
			t.Errorf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
		}
	})

	doRequest(t, router, http.MethodPost, "/auth/confirm", map[string]string{
		"email": testEmail, "code": testConfirmationCode,
	}, "")

	t.Run("wrong password is rejected", func(t *testing.T) {
		rec := doRequest(t, router, http.MethodPost, "/auth/signin", map[string]string{
			"email": testEmail, "password": "wrong-password",
		}, "")
		if rec.Code != http.StatusUnauthorized {
			t.Errorf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
		}
	})

	t.Run("correct credentials succeed", func(t *testing.T) {
		rec := doRequest(t, router, http.MethodPost, "/auth/signin", map[string]string{
			"email": testEmail, "password": testPassword,
		}, "")
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
		}
		var resp signInResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		if resp.AccessToken == "" {
			t.Error("expected a non-empty access_token")
		}
	})
}

func TestForgotPassword(t *testing.T) {
	t.Run("missing email is rejected", func(t *testing.T) {
		router, _ := newTestServer(t)
		rec := doRequest(t, router, http.MethodPost, "/auth/forgot-password", map[string]string{}, "")
		if rec.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
		}
	})

	t.Run("unknown email still responds 204, to avoid leaking account existence", func(t *testing.T) {
		router, _ := newTestServer(t)
		rec := doRequest(t, router, http.MethodPost, "/auth/forgot-password", map[string]string{
			"email": "unknown@example.com",
		}, "")
		if rec.Code != http.StatusNoContent {
			t.Errorf("status = %d, want %d", rec.Code, http.StatusNoContent)
		}
	})

	t.Run("registered, confirmed email responds 204", func(t *testing.T) {
		router, mock := newTestServer(t)
		expectUserInsert(mock)
		doRequest(t, router, http.MethodPost, "/auth/signup", map[string]string{
			"email": testEmail, "password": testPassword,
		}, "")
		doRequest(t, router, http.MethodPost, "/auth/confirm", map[string]string{
			"email": testEmail, "code": testConfirmationCode,
		}, "")

		rec := doRequest(t, router, http.MethodPost, "/auth/forgot-password", map[string]string{
			"email": testEmail,
		}, "")
		if rec.Code != http.StatusNoContent {
			t.Errorf("status = %d, want %d", rec.Code, http.StatusNoContent)
		}
	})
}

func TestResetPassword(t *testing.T) {
	const newPassword = "NewPassw0rd!123"

	setUpConfirmedUserWithResetRequested := func(t *testing.T) http.Handler {
		t.Helper()
		router, mock := newTestServer(t)
		expectUserInsert(mock)
		doRequest(t, router, http.MethodPost, "/auth/signup", map[string]string{
			"email": testEmail, "password": testPassword,
		}, "")
		doRequest(t, router, http.MethodPost, "/auth/confirm", map[string]string{
			"email": testEmail, "code": testConfirmationCode,
		}, "")
		doRequest(t, router, http.MethodPost, "/auth/forgot-password", map[string]string{
			"email": testEmail,
		}, "")
		return router
	}

	t.Run("missing fields are rejected", func(t *testing.T) {
		router, _ := newTestServer(t)
		rec := doRequest(t, router, http.MethodPost, "/auth/reset-password", map[string]string{
			"email": testEmail,
		}, "")
		if rec.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
		}
	})

	t.Run("wrong code is rejected", func(t *testing.T) {
		router := setUpConfirmedUserWithResetRequested(t)
		rec := doRequest(t, router, http.MethodPost, "/auth/reset-password", map[string]string{
			"email": testEmail, "code": "999999", "new_password": newPassword,
		}, "")
		if rec.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
		}
	})

	t.Run("correct code resets the password", func(t *testing.T) {
		router := setUpConfirmedUserWithResetRequested(t)

		rec := doRequest(t, router, http.MethodPost, "/auth/reset-password", map[string]string{
			"email": testEmail, "code": testConfirmationCode, "new_password": newPassword,
		}, "")
		if rec.Code != http.StatusNoContent {
			t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
		}

		rec = doRequest(t, router, http.MethodPost, "/auth/signin", map[string]string{
			"email": testEmail, "password": newPassword,
		}, "")
		if rec.Code != http.StatusOK {
			t.Fatalf("signin with new password: status = %d, body = %s", rec.Code, rec.Body.String())
		}
	})
}

func TestSignOut(t *testing.T) {
	t.Run("missing authorization header is rejected", func(t *testing.T) {
		router, _ := newTestServer(t)
		rec := doRequest(t, router, http.MethodPost, "/auth/signout", nil, "")
		if rec.Code != http.StatusUnauthorized {
			t.Errorf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
		}
	})

	t.Run("valid token succeeds and is then invalidated", func(t *testing.T) {
		router, mock := newTestServer(t)
		token := signUpConfirmAndSignIn(t, router, mock)

		rec := doRequest(t, router, http.MethodPost, "/auth/signout", nil, token)
		if rec.Code != http.StatusNoContent {
			t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
		}

		// The token must no longer authenticate any protected route.
		rec = doRequest(t, router, http.MethodPost, "/auth/signout", nil, token)
		if rec.Code != http.StatusUnauthorized {
			t.Errorf("reused token: status = %d, want %d", rec.Code, http.StatusUnauthorized)
		}
	})
}

func TestDeleteMe(t *testing.T) {
	t.Run("missing authorization header is rejected", func(t *testing.T) {
		router, _ := newTestServer(t)
		rec := doRequest(t, router, http.MethodDelete, "/auth/me", nil, "")
		if rec.Code != http.StatusUnauthorized {
			t.Errorf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
		}
	})

	t.Run("valid token deletes the profile and invalidates the token", func(t *testing.T) {
		router, mock := newTestServer(t)
		token := signUpConfirmAndSignIn(t, router, mock)

		mock.ExpectExec("DELETE FROM users").WillReturnResult(sqlmock.NewResult(0, 1))

		rec := doRequest(t, router, http.MethodDelete, "/auth/me", nil, token)
		if rec.Code != http.StatusNoContent {
			t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("unmet db expectations: %v", err)
		}

		// The access token must no longer be valid after account deletion.
		rec = doRequest(t, router, http.MethodDelete, "/auth/me", nil, token)
		if rec.Code != http.StatusUnauthorized {
			t.Errorf("reused token: status = %d, want %d", rec.Code, http.StatusUnauthorized)
		}
	})
}
