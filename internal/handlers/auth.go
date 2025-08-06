package handlers

import (
	"encoding/json"
	"katkam/internal/auth"
	"net/http"
)

type AuthHandler struct {
	authorizer *auth.Authorizer
}

func NewAuthHandler(authorizer *auth.Authorizer) *AuthHandler {
	return &AuthHandler{
		authorizer: authorizer,
	}
}

func (ac *AuthHandler) ValidateToken(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != "POST" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]string{"error": "Method not allowed"})
		return
	}

	if token := r.Header["Authorization"]; token != nil {
		ok, err := ac.authorizer.VerifyJWT(token[0])

		if err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]bool{"tokenValid": false})
			return
		}

		if !ok {
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]bool{"tokenValid": false})
			return
		}

		json.NewEncoder(w).Encode(map[string]bool{"tokenValid": true})
	} else {
		json.NewEncoder(w).Encode(map[string]bool{"tokenValid": false})
	}
}

func (ac *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var loginReq LoginRequest
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != "POST" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]string{"error": "Method not allowed"})
		return
	}

	if err := json.NewDecoder(r.Body).Decode(&loginReq); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	username := loginReq.Username
	password := loginReq.Password
	if username == "" || password == "" {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "Empty credentials"})
		return
	}

	ok, err := ac.authorizer.AuthorizeUser(username, password)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	if !ok {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Server error"})
		return
	}

	jwt, err := ac.authorizer.GetJwtToken(username)

	// Set JWT token as HTTP-only cookie
	cookie := &http.Cookie{
		Name:     "jwt",
		Value:    string(jwt),
		Path:     "/",
		MaxAge:   3600 * 24, // 24 hour
		HttpOnly: true,
		Secure:   false, // Set to true in production with HTTPS
		SameSite: http.SameSiteLaxMode,
	}
	http.SetCookie(w, cookie)

	response := map[string]string{
		"token":   string(jwt),
		"message": "Authentication successful",
	}

	json.NewEncoder(w).Encode(response)
}

func (ac *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != "POST" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]string{"error": "Method not allowed"})
		return
	}

	// Clear the auth token cookie
	cookie := &http.Cookie{
		Name:     "auth_token",
		Value:    "",
		Path:     "/",
		MaxAge:   -1, // Delete cookie immediately
		HttpOnly: true,
		Secure:   false, // Set to true in production with HTTPS
		SameSite: http.SameSiteLaxMode,
	}
	http.SetCookie(w, cookie)

	response := map[string]string{
		"message": "Logout successful",
	}
	json.NewEncoder(w).Encode(response)
}

func (ac *AuthHandler) ProtectedEndpoint(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	// This endpoint would typically be wrapped with JWT verification middleware
	response := map[string]string{
		"message": "Access granted to protected resource",
		"data":    "This is protected content",
	}
	json.NewEncoder(w).Encode(response)
}
