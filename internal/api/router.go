package api

import "net/http"

func NewRouter(h *Handler, authMiddleware func(http.Handler) http.Handler) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	mux.HandleFunc("POST /auth/signup", h.SignUp)
	mux.HandleFunc("POST /auth/confirm", h.ConfirmSignUp)
	mux.HandleFunc("POST /auth/signin", h.SignIn)
	mux.HandleFunc("POST /auth/forgot-password", h.ForgotPassword)
	mux.HandleFunc("POST /auth/reset-password", h.ResetPassword)
	mux.Handle("POST /auth/signout", authMiddleware(http.HandlerFunc(h.SignOut)))
	mux.Handle("DELETE /auth/me", authMiddleware(http.HandlerFunc(h.DeleteMe)))

	return mux
}
