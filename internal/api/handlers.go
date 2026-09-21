package api

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/k07g/g4/internal/auth"
	"github.com/k07g/g4/internal/db"
)

type Handler struct {
	provider auth.Provider
	users    *db.UserRepository
}

func NewHandler(provider auth.Provider, users *db.UserRepository) *Handler {
	return &Handler{provider: provider, users: users}
}

type errorResponse struct {
	Error string `json:"error"`
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, errorResponse{Error: message})
}

// --- Sign up ---

type signUpRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type signUpResponse struct {
	UserSub       string `json:"user_sub"`
	UserConfirmed bool   `json:"user_confirmed"`
}

func (h *Handler) SignUp(w http.ResponseWriter, r *http.Request) {
	var req signUpRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Email == "" || req.Password == "" {
		writeError(w, http.StatusBadRequest, "email and password are required")
		return
	}

	result, err := h.provider.SignUp(r.Context(), req.Email, req.Password)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if _, err := h.users.Create(r.Context(), result.UserSub, req.Email); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to persist user profile")
		return
	}

	writeJSON(w, http.StatusCreated, signUpResponse{
		UserSub:       result.UserSub,
		UserConfirmed: result.UserConfirmed,
	})
}

// --- Confirm sign up (required by Cognito before the user can sign in) ---

type confirmSignUpRequest struct {
	Email string `json:"email"`
	Code  string `json:"code"`
}

func (h *Handler) ConfirmSignUp(w http.ResponseWriter, r *http.Request) {
	var req confirmSignUpRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Email == "" || req.Code == "" {
		writeError(w, http.StatusBadRequest, "email and code are required")
		return
	}

	if err := h.provider.ConfirmSignUp(r.Context(), req.Email, req.Code); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// --- Sign in ---

type signInRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type signInResponse struct {
	AccessToken  string `json:"access_token"`
	IDToken      string `json:"id_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int32  `json:"expires_in"`
}

func (h *Handler) SignIn(w http.ResponseWriter, r *http.Request) {
	var req signInRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Email == "" || req.Password == "" {
		writeError(w, http.StatusBadRequest, "email and password are required")
		return
	}

	result, err := h.provider.SignIn(r.Context(), req.Email, req.Password)
	if err != nil {
		if errors.Is(err, auth.ErrChallengeRequired) {
			writeError(w, http.StatusUnprocessableEntity, "additional authentication challenge required")
			return
		}
		writeError(w, http.StatusUnauthorized, "invalid email or password")
		return
	}

	writeJSON(w, http.StatusOK, signInResponse{
		AccessToken:  result.AccessToken,
		IDToken:      result.IDToken,
		RefreshToken: result.RefreshToken,
		ExpiresIn:    result.ExpiresIn,
	})
}

// --- Forgot password ---

type forgotPasswordRequest struct {
	Email string `json:"email"`
}

// ForgotPassword always responds 204 regardless of whether the email is
// registered or the provider call succeeds. This intentionally avoids
// leaking account existence through this endpoint, a well-known email
// enumeration vector for password-reset flows.
func (h *Handler) ForgotPassword(w http.ResponseWriter, r *http.Request) {
	var req forgotPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Email == "" {
		writeError(w, http.StatusBadRequest, "email is required")
		return
	}

	_ = h.provider.ForgotPassword(r.Context(), req.Email)
	w.WriteHeader(http.StatusNoContent)
}

// --- Confirm forgot password (reset) ---

type resetPasswordRequest struct {
	Email       string `json:"email"`
	Code        string `json:"code"`
	NewPassword string `json:"new_password"`
}

func (h *Handler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	var req resetPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Email == "" || req.Code == "" || req.NewPassword == "" {
		writeError(w, http.StatusBadRequest, "email, code and new_password are required")
		return
	}

	if err := h.provider.ConfirmForgotPassword(r.Context(), req.Email, req.Code, req.NewPassword); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// --- Sign out ---

func (h *Handler) SignOut(w http.ResponseWriter, r *http.Request) {
	token, ok := r.Context().Value(contextKeyAccessToken).(string)
	if !ok || token == "" {
		writeError(w, http.StatusUnauthorized, "missing access token")
		return
	}

	if err := h.provider.SignOut(r.Context(), token); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to sign out")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// --- Delete account ---

func (h *Handler) DeleteMe(w http.ResponseWriter, r *http.Request) {
	token, ok := r.Context().Value(contextKeyAccessToken).(string)
	if !ok || token == "" {
		writeError(w, http.StatusUnauthorized, "missing access token")
		return
	}
	sub, _ := r.Context().Value(contextKeyCognitoSub).(string)

	if err := h.provider.DeleteUser(r.Context(), token); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete user")
		return
	}

	if err := h.users.DeleteByCognitoSub(r.Context(), sub); err != nil && !errors.Is(err, db.ErrUserNotFound) {
		writeError(w, http.StatusInternalServerError, "user deleted from cognito but failed to remove profile data")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
