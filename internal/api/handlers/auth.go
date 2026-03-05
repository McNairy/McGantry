package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/gantrydev/gantry/internal/api/middleware"
	"github.com/gantrydev/gantry/internal/auth"
	"github.com/gantrydev/gantry/internal/db"
	"github.com/gantrydev/gantry/internal/events"
)

// loginRequest represents the JSON body of a login request.
type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// loginResponse represents the JSON response returned on successful login.
type loginResponse struct {
	Token string   `json:"token"`
	User  *db.User `json:"user"`
}

// registerRequest represents the JSON body for user registration.
type registerRequest struct {
	Username    string `json:"username"`
	Password    string `json:"password"`
	DisplayName string `json:"displayName,omitempty"`
	Email       string `json:"email,omitempty"`
	Role        string `json:"role,omitempty"`
}

// Login handles POST /auth/login. It verifies the username and password
// against the database and returns a signed JWT on success.
func (h *Handlers) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Username == "" || req.Password == "" {
		writeError(w, http.StatusBadRequest, "username and password are required")
		return
	}

	// Look up the user in the database.
	user, err := h.DB.GetUserByUsername(r.Context(), req.Username)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}

	// Verify the password against the stored hash.
	if err := h.Auth.CheckPassword(user.PasswordHash, req.Password); err != nil {
		writeError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}

	// Build the auth.User needed by GenerateToken.
	authUser := &auth.User{
		ID:       user.ID,
		Username: user.Username,
		Role:     user.Role,
	}

	token, err := h.Auth.GenerateToken(authUser)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to generate token")
		return
	}

	// Publish login event.
	h.Events.Publish(events.Event{
		Type: events.UserLogin,
		Data: map[string]any{
			"userId":   user.ID,
			"username": user.Username,
		},
	})

	writeJSON(w, http.StatusOK, loginResponse{
		Token: token,
		User:  user,
	})
}

// GetMe handles GET /auth/me. It returns the currently authenticated user's
// information from the JWT claims stored in the request context.
func (h *Handlers) GetMe(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetClaims(r.Context())
	if claims == nil {
		writeError(w, http.StatusUnauthorized, "not authenticated")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"userId":   claims.UserID,
		"username": claims.Username,
		"role":     claims.Role,
	})
}

// Register handles POST /auth/register. It creates a new user account.
// This endpoint requires admin privileges (enforced by RequireRole middleware).
func (h *Handlers) Register(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Username == "" || req.Password == "" {
		writeError(w, http.StatusBadRequest, "username and password are required")
		return
	}

	hash, err := h.Auth.HashPassword(req.Password)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to hash password")
		return
	}

	role := req.Role
	if role == "" {
		role = "viewer"
	}

	user := &db.User{
		Username:     req.Username,
		PasswordHash: hash,
		DisplayName:  req.DisplayName,
		Email:        req.Email,
		Role:         role,
	}

	if err := h.DB.CreateUser(r.Context(), user); err != nil {
		writeError(w, http.StatusConflict, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, user)
}
