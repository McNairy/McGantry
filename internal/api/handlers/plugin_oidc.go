package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go2engle/gantry/internal/auth"
	"github.com/go2engle/gantry/internal/db"
)

type oidcDiscovery struct {
	AuthorizationEndpoint string `json:"authorization_endpoint"`
	TokenEndpoint         string `json:"token_endpoint"`
	UserinfoEndpoint      string `json:"userinfo_endpoint"`
}

type oidcTokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
}

type oidcUserInfo struct {
	Sub               string   `json:"sub"`
	Email             string   `json:"email"`
	Name              string   `json:"name"`
	PreferredUsername string   `json:"preferred_username"`
	Groups            []string `json:"groups"`
}

// PluginOIDCBegin starts the OIDC authorization flow for an external auth-provider plugin.
// GET /api/v1/auth/plugin/{name}
func (h *Handlers) PluginOIDCBegin(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	p, err := h.DB.GetPlugin(r.Context(), name)
	if err != nil || p == nil || !p.Enabled || p.Manifest == nil || p.Manifest.Category != "auth-provider" {
		writeError(w, http.StatusNotFound, "auth provider not found or not enabled")
		return
	}

	issuerURL, clientID, _, redirectBase := oidcPluginConfig(p.Config, r)
	if issuerURL == "" || clientID == "" {
		writeError(w, http.StatusBadRequest, "auth provider is not fully configured (requires issuer URL and client ID)")
		return
	}

	discovery, err := fetchOIDCDiscovery(issuerURL)
	if err != nil {
		log.Printf("[plugin-oidc] %s: discovery failed for %s: %v", name, issuerURL, err)
		writeError(w, http.StatusBadGateway, fmt.Sprintf("failed to reach OIDC provider at %s/.well-known/openid-configuration: %v", issuerURL, err))
		return
	}

	state, err := randomHex16()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to generate state")
		return
	}

	cookieName := "oidc_state_" + name
	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    state,
		Path:     "/",
		MaxAge:   600,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   isHTTPSRequest(r),
	})

	if returnTo := normalizeReturnTo(r, r.URL.Query().Get("return_to")); returnTo != "" {
		http.SetCookie(w, &http.Cookie{
			Name:     "oidc_return_to_" + name,
			Value:    returnTo,
			Path:     "/",
			MaxAge:   600,
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
			Secure:   isHTTPSRequest(r),
		})
	}

	callbackURL := redirectBase + "/api/v1/auth/plugin/" + name + "/callback"

	params := url.Values{}
	params.Set("response_type", "code")
	params.Set("client_id", clientID)
	params.Set("redirect_uri", callbackURL)
	params.Set("scope", "openid email profile groups")
	params.Set("state", state)
	http.Redirect(w, r, discovery.AuthorizationEndpoint+"?"+params.Encode(), http.StatusTemporaryRedirect)
}

// PluginOIDCCallback handles the OIDC authorization code callback for an external auth-provider plugin.
// GET /api/v1/auth/plugin/{name}/callback
func (h *Handlers) PluginOIDCCallback(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	p, err := h.DB.GetPlugin(r.Context(), name)
	if err != nil || p == nil || !p.Enabled || p.Manifest == nil || p.Manifest.Category != "auth-provider" {
		writeError(w, http.StatusNotFound, "auth provider not found or not enabled")
		return
	}

	cookieName := "oidc_state_" + name
	stateCookie, err := r.Cookie(cookieName)
	if err != nil || r.URL.Query().Get("state") != stateCookie.Value {
		writeError(w, http.StatusBadRequest, "invalid or missing OIDC state")
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name: cookieName, Value: "", Path: "/", MaxAge: -1,
		HttpOnly: true, SameSite: http.SameSiteLaxMode, Secure: isHTTPSRequest(r),
	})

	code := r.URL.Query().Get("code")
	if code == "" {
		writeError(w, http.StatusBadRequest, "missing authorization code")
		return
	}

	issuerURL, clientID, clientSecret, redirectBase := oidcPluginConfig(p.Config, r)
	if issuerURL == "" || clientID == "" {
		writeError(w, http.StatusInternalServerError, "auth provider config changed during flow")
		return
	}

	discovery, err := fetchOIDCDiscovery(issuerURL)
	if err != nil {
		log.Printf("[plugin-oidc] %s: discovery failed for %s: %v", name, issuerURL, err)
		writeError(w, http.StatusBadGateway, fmt.Sprintf("failed to reach OIDC provider at %s/.well-known/openid-configuration: %v", issuerURL, err))
		return
	}

	callbackURL := redirectBase + "/api/v1/auth/plugin/" + name + "/callback"
	tokens, err := exchangeOIDCCode(discovery.TokenEndpoint, code, clientID, clientSecret, callbackURL)
	if err != nil {
		log.Printf("[plugin-oidc] %s: token exchange failed: %v", name, err)
		writeSSOProviderError(w, p.Manifest.Title, "exchange authorization code", err)
		return
	}

	userInfo, err := fetchOIDCUserInfo(discovery.UserinfoEndpoint, tokens.AccessToken)
	if err != nil {
		log.Printf("[plugin-oidc] %s: userinfo failed: %v", name, err)
		writeSSOProviderError(w, p.Manifest.Title, "fetch user info", err)
		return
	}

	returnTo := ""
	if c, err := r.Cookie("oidc_return_to_" + name); err == nil && c.Value != "" {
		returnTo = normalizeReturnTo(r, c.Value)
		http.SetCookie(w, &http.Cookie{
			Name: "oidc_return_to_" + name, Value: "", Path: "/", MaxAge: -1,
			HttpOnly: true, SameSite: http.SameSiteLaxMode, Secure: isHTTPSRequest(r),
		})
	}

	ctx := r.Context()
	// Find user by plugin-namespaced subject, then fall back to email.
	username := name + ":" + userInfo.Sub
	gantryUser, _ := h.DB.GetUserByUsername(ctx, username)

	if gantryUser == nil && userInfo.Email != "" {
		usersByEmail, err := h.DB.GetUsersByEmail(ctx, userInfo.Email)
		if err == nil && len(usersByEmail) == 1 {
			gantryUser = usersByEmail[0]
		}
	}

	if gantryUser == nil {
		autoProvision, _ := p.Config["autoProvision"].(bool)
		if !autoProvision {
			errorURL := "/login?error=sso_not_authorized"
			if returnTo != "" {
				errorURL = returnTo + "/login?error=sso_not_authorized"
			}
			http.Redirect(w, r, errorURL, http.StatusTemporaryRedirect)
			return
		}

		defaultRole, _ := p.Config["defaultRole"].(string)
		if defaultRole == "" {
			defaultRole = "viewer"
		}
		displayName := userInfo.Name
		if displayName == "" {
			displayName = userInfo.PreferredUsername
		}
		if displayName == "" {
			displayName = userInfo.Sub
		}
		newUser := &db.User{
			Username:     username,
			PasswordHash: "",
			DisplayName:  displayName,
			Email:        userInfo.Email,
			Role:         defaultRole,
			SSOOnly:      true,
		}
		if err := h.DB.CreateUser(ctx, newUser); err != nil {
			gantryUser, _ = h.DB.GetUserByUsername(ctx, username)
			if gantryUser == nil {
				writeError(w, http.StatusInternalServerError, "failed to create user: "+err.Error())
				return
			}
		} else {
			gantryUser = newUser
		}
	}

	// Sync groups from OIDC claims if enabled.
	if syncGroups, _ := p.Config["syncGroups"].(bool); syncGroups && len(userInfo.Groups) > 0 {
		h.syncOIDCGroups(r.Context(), name, userInfo.Groups, gantryUser)
	}

	token, err := h.Auth.GenerateToken(&auth.User{
		ID:       gantryUser.ID,
		Username: gantryUser.Username,
		Role:     gantryUser.Role,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to generate token")
		return
	}
	http.SetCookie(w, sessionCookie(r, token))

	redirectURL := "/"
	if returnTo != "" {
		redirectURL = returnTo + "/"
	}
	http.Redirect(w, r, redirectURL, http.StatusTemporaryRedirect)
}

// oidcPluginConfig extracts OIDC settings from plugin config.
// Returns issuerURL, clientID, clientSecret, and the redirect base URL.
func oidcPluginConfig(config map[string]any, r *http.Request) (issuerURL, clientID, clientSecret, redirectBase string) {
	issuerURL, _ = config["oidcIssuerUrl"].(string)
	if issuerURL == "" {
		issuerURL, _ = config["authentikUrl"].(string)
	}
	clientID, _ = config["clientId"].(string)
	clientSecret, _ = config["clientSecret"].(string)

	// Use gantryExternalUrl if configured (needed when behind a reverse proxy),
	// otherwise derive from the request.
	redirectBase, _ = config["gantryExternalUrl"].(string)
	redirectBase = strings.TrimRight(redirectBase, "/")
	if redirectBase == "" {
		redirectBase = requestOrigin(r)
	}
	return
}

func fetchOIDCDiscovery(issuerURL string) (*oidcDiscovery, error) {
	issuerURL = strings.TrimRight(issuerURL, "/")
	resp, err := http.Get(issuerURL + "/.well-known/openid-configuration")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("discovery endpoint returned %d", resp.StatusCode)
	}
	var d oidcDiscovery
	if err := json.NewDecoder(resp.Body).Decode(&d); err != nil {
		return nil, err
	}
	if d.AuthorizationEndpoint == "" || d.TokenEndpoint == "" {
		return nil, fmt.Errorf("incomplete OIDC discovery document")
	}
	return &d, nil
}

func exchangeOIDCCode(tokenEndpoint, code, clientID, clientSecret, redirectURI string) (*oidcTokenResponse, error) {
	body := url.Values{}
	body.Set("grant_type", "authorization_code")
	body.Set("code", code)
	body.Set("client_id", clientID)
	body.Set("client_secret", clientSecret)
	body.Set("redirect_uri", redirectURI)

	resp, err := http.PostForm(tokenEndpoint, body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token endpoint returned %d: %s", resp.StatusCode, string(raw))
	}
	var t oidcTokenResponse
	if err := json.Unmarshal(raw, &t); err != nil {
		return nil, err
	}
	if t.AccessToken == "" {
		return nil, fmt.Errorf("token endpoint returned no access_token")
	}
	return &t, nil
}

func fetchOIDCUserInfo(userinfoEndpoint, accessToken string) (*oidcUserInfo, error) {
	req, err := http.NewRequest("GET", userinfoEndpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("userinfo endpoint returned %d", resp.StatusCode)
	}
	var u oidcUserInfo
	if err := json.NewDecoder(resp.Body).Decode(&u); err != nil {
		return nil, err
	}
	if u.Sub == "" {
		return nil, fmt.Errorf("userinfo response missing 'sub' claim")
	}
	return &u, nil
}

func (h *Handlers) syncOIDCGroups(ctx context.Context, pluginName string, groups []string, user *db.User) {
	var groupIDs []string
	for _, groupName := range groups {
		sourceID := pluginName + "/" + groupName
		g, err := h.DB.GetGroupByName(ctx, sourceID)
		if err != nil {
			newGroup := &db.Group{
				Name:        sourceID,
				DisplayName: groupName,
				Source:      pluginName,
				SourceID:    sourceID,
				Role:        "viewer",
			}
			if err := h.DB.CreateGroup(ctx, newGroup); err != nil {
				continue
			}
			g = newGroup
		}
		if g != nil {
			groupIDs = append(groupIDs, g.ID)
		}
	}
	_ = h.DB.SyncUserGroups(ctx, user.ID, groupIDs)
}
