package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"

	"github.com/go2engle/gantry/internal/auth"
	"github.com/go2engle/gantry/internal/db"
	ghplugin "github.com/go2engle/gantry/internal/plugins/github"
)

// GetGitHubSSOConfig returns whether GitHub SSO is enabled (public endpoint,
// used by the login page to show/hide the SSO button).
func (h *Handlers) GetGitHubSSOConfig(w http.ResponseWriter, r *http.Request) {
	p, err := h.DB.GetPlugin(r.Context(), "github")
	if err != nil || p == nil || !p.Enabled {
		writeJSON(w, http.StatusOK, map[string]any{"ssoEnabled": false})
		return
	}
	ssoEnabled, _ := p.Config["ssoEnabled"].(bool)
	clientID, _ := p.Config["oauthClientId"].(string)
	writeJSON(w, http.StatusOK, map[string]any{
		"ssoEnabled": ssoEnabled && clientID != "",
	})
}

// GitHubOAuthBegin redirects the browser to GitHub's OAuth authorization page.
// This is a public endpoint — the user is not yet authenticated.
func (h *Handlers) GitHubOAuthBegin(w http.ResponseWriter, r *http.Request) {
	p, err := h.DB.GetPlugin(r.Context(), "github")
	if err != nil || p == nil || !p.Enabled {
		writeError(w, http.StatusNotFound, "GitHub plugin not installed or not enabled")
		return
	}
	ssoEnabled, _ := p.Config["ssoEnabled"].(bool)
	clientID, _ := p.Config["oauthClientId"].(string)
	if !ssoEnabled || clientID == "" {
		writeError(w, http.StatusBadRequest, "GitHub SSO is not configured")
		return
	}

	state, err := randomHex16()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to generate state")
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     "gh_oauth_state",
		Value:    state,
		Path:     "/",
		MaxAge:   600,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	authURL := fmt.Sprintf(
		"https://github.com/login/oauth/authorize?client_id=%s&state=%s&scope=read:user+user:email",
		clientID, state,
	)
	http.Redirect(w, r, authURL, http.StatusTemporaryRedirect)
}

// GitHubOAuthCallback handles the GitHub OAuth redirect back to Gantry.
// It validates the state, exchanges the code for a GitHub access token,
// fetches the GitHub user, finds or creates a Gantry account, issues a JWT,
// then redirects to the frontend with the token in the query string.
func (h *Handlers) GitHubOAuthCallback(w http.ResponseWriter, r *http.Request) {
	// Validate OAuth state to prevent CSRF.
	stateCookie, err := r.Cookie("gh_oauth_state")
	if err != nil || r.URL.Query().Get("state") != stateCookie.Value {
		writeError(w, http.StatusBadRequest, "invalid or missing oauth state")
		return
	}
	// Clear state cookie.
	http.SetCookie(w, &http.Cookie{
		Name:   "gh_oauth_state",
		Value:  "",
		Path:   "/",
		MaxAge: -1,
	})

	code := r.URL.Query().Get("code")
	if code == "" {
		writeError(w, http.StatusBadRequest, "missing oauth code")
		return
	}

	p, err := h.DB.GetPlugin(r.Context(), "github")
	if err != nil || p == nil || !p.Enabled {
		writeError(w, http.StatusInternalServerError, "GitHub plugin unavailable")
		return
	}
	clientID, _ := p.Config["oauthClientId"].(string)
	clientSecret, _ := p.Config["oauthClientSecret"].(string)
	defaultRole, _ := p.Config["defaultRole"].(string)
	if defaultRole == "" {
		defaultRole = "viewer"
	}

	// Exchange code → GitHub access token.
	accessToken, err := ghplugin.ExchangeOAuthCode(code, clientID, clientSecret)
	if err != nil {
		writeError(w, http.StatusBadGateway, "failed to exchange oauth code: "+err.Error())
		return
	}

	// Fetch GitHub user profile.
	ghUser, err := ghplugin.FetchUserWithToken(accessToken)
	if err != nil {
		writeError(w, http.StatusBadGateway, "failed to fetch GitHub user: "+err.Error())
		return
	}

	// Find existing Gantry user by GitHub login convention (username = "github:<login>").
	username := "github:" + ghUser.Login
	gantryUser, _ := h.DB.GetUserByUsername(r.Context(), username)

	if gantryUser == nil {
		// Create a new Gantry user for this GitHub identity.
		displayName := ghUser.Name
		if displayName == "" {
			displayName = ghUser.Login
		}
		newUser := &db.User{
			Username:     username,
			PasswordHash: "", // SSO users have no local password
			DisplayName:  displayName,
			Email:        ghUser.Email,
			Role:         defaultRole,
		}
		if err := h.DB.CreateUser(r.Context(), newUser); err != nil {
			// If username already exists (race), try to fetch again.
			gantryUser, _ = h.DB.GetUserByUsername(r.Context(), username)
			if gantryUser == nil {
				writeError(w, http.StatusInternalServerError, "failed to create user: "+err.Error())
				return
			}
		} else {
			gantryUser = newUser
		}
	}

	// Issue a Gantry JWT.
	token, err := h.Auth.GenerateToken(&auth.User{
		ID:       gantryUser.ID,
		Username: gantryUser.Username,
		Role:     gantryUser.Role,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to generate token")
		return
	}

	// Redirect to the SPA with the token as a query param.
	// The frontend reads this param on load, stores it, and cleans the URL.
	http.Redirect(w, r, "/?github_token="+token, http.StatusTemporaryRedirect)
}

// GetGitHubRepo fetches live repository info from GitHub for a given URL.
// Query param: ?url=https://github.com/owner/repo
func (h *Handlers) GetGitHubRepo(w http.ResponseWriter, r *http.Request) {
	repoURL := r.URL.Query().Get("url")
	if repoURL == "" {
		writeError(w, http.StatusBadRequest, "url query param is required")
		return
	}
	if !strings.Contains(repoURL, "github.com") {
		writeError(w, http.StatusBadRequest, "only github.com URLs are supported")
		return
	}

	owner, repo, err := ghplugin.ParseGitHubURL(repoURL)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid GitHub URL: "+err.Error())
		return
	}

	p, err := h.DB.GetPlugin(r.Context(), "github")
	if err != nil || p == nil {
		writeError(w, http.StatusNotFound, "GitHub plugin not installed")
		return
	}
	if !p.Enabled {
		writeError(w, http.StatusBadRequest, "GitHub plugin is not enabled")
		return
	}

	client, err := ghplugin.NewClient(p.Config)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "GitHub client error: "+err.Error())
		return
	}

	repoInfo, err := client.GetRepo(owner, repo)
	if err != nil {
		writeError(w, http.StatusBadGateway, "failed to fetch repo: "+err.Error())
		return
	}

	commits, _ := client.GetCommits(owner, repo, 5)
	prs, _ := client.GetPullRequests(owner, repo, 10)
	readme, _ := client.GetReadme(owner, repo)
	latestRelease, _ := client.GetLatestRelease(owner, repo)

	// Ensure slices are never null in JSON.
	if commits == nil {
		commits = []ghplugin.Commit{}
	}
	if prs == nil {
		prs = []ghplugin.PullRequest{}
	}

	writeJSON(w, http.StatusOK, ghplugin.RepoInfo{
		Repo:          repoInfo,
		Commits:       commits,
		PullRequests:  prs,
		Readme:        readme,
		LatestRelease: latestRelease,
	})
}

// randomHex16 generates 16 random bytes as a hex string.
func randomHex16() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
