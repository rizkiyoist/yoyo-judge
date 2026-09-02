package server

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

type oauthStateEntry struct {
	expiry time.Time
}

var (
	oauthStateMu sync.Mutex
	oauthStates  = map[string]oauthStateEntry{}
)

func oauthConfig() *oauth2.Config {
	return &oauth2.Config{
		ClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
		ClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
		RedirectURL:  os.Getenv("GOOGLE_REDIRECT_URL"),
		Scopes: []string{
			"https://www.googleapis.com/auth/userinfo.email",
			"https://www.googleapis.com/auth/userinfo.profile",
		},
		Endpoint: google.Endpoint,
	}
}

func isGoogleConfigured() bool {
	return os.Getenv("GOOGLE_CLIENT_ID") != "" &&
		os.Getenv("GOOGLE_CLIENT_SECRET") != "" &&
		os.Getenv("GOOGLE_REDIRECT_URL") != ""
}

func newOAuthState() string {
	buf := make([]byte, 16)
	_, _ = rand.Read(buf)
	state := hex.EncodeToString(buf)
	oauthStateMu.Lock()
	oauthStates[state] = oauthStateEntry{expiry: time.Now().Add(5 * time.Minute)}
	oauthStateMu.Unlock()
	return state
}

func consumeOAuthState(state string) bool {
	oauthStateMu.Lock()
	defer oauthStateMu.Unlock()
	entry, ok := oauthStates[state]
	delete(oauthStates, state)
	return ok && time.Now().Before(entry.expiry)
}

func (s *Store) handleGoogleLogin(w http.ResponseWriter, r *http.Request) {
	if !isGoogleConfigured() {
		writeError(w, http.StatusServiceUnavailable, "Google OAuth is not configured on this server")
		return
	}
	url := oauthConfig().AuthCodeURL(newOAuthState(), oauth2.AccessTypeOnline)
	http.Redirect(w, r, url, http.StatusFound)
}

type googleUserInfo struct {
	Sub           string `json:"sub"`
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
	GivenName     string `json:"given_name"`
	FamilyName    string `json:"family_name"`
}

func (s *Store) handleGoogleCallback(w http.ResponseWriter, r *http.Request) {
	if !isGoogleConfigured() {
		writeError(w, http.StatusServiceUnavailable, "Google OAuth is not configured on this server")
		return
	}
	if !consumeOAuthState(r.URL.Query().Get("state")) {
		writeError(w, http.StatusBadRequest, "invalid or expired OAuth state")
		return
	}

	token, err := oauthConfig().Exchange(context.Background(), r.URL.Query().Get("code"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "failed to exchange OAuth code: "+err.Error())
		return
	}

	client := oauthConfig().Client(context.Background(), token)
	resp, err := client.Get("https://www.googleapis.com/oauth2/v3/userinfo")
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to fetch Google user info")
		return
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	var info googleUserInfo
	if err := json.Unmarshal(body, &info); err != nil || info.Sub == "" {
		writeError(w, http.StatusInternalServerError, "failed to parse Google user info")
		return
	}

	user, err := s.findOrCreateUserByGoogle(info.Sub, info.Email, info.GivenName, info.FamilyName)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to upsert user: "+err.Error())
		return
	}

	sessionToken := s.createSession(user.ID)
	redirectTo := frontendBaseURL(r) + "/auth/callback?token=" + sessionToken
	http.Redirect(w, r, redirectTo, http.StatusFound)
}

// frontendBaseURL returns the base URL to redirect to after OAuth.
// FRONTEND_URL overrides; otherwise it's inferred from the current request's
// scheme + host + BASE_PATH (correct for the embedded single-binary deploy).
func frontendBaseURL(r *http.Request) string {
	if u := os.Getenv("FRONTEND_URL"); u != "" {
		return strings.TrimSuffix(u, "/")
	}
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	if proto := r.Header.Get("X-Forwarded-Proto"); proto != "" {
		scheme = proto
	}
	bp := os.Getenv("BASE_PATH")
	if bp == "" {
		bp = "/yoyojudge"
	}
	bp = strings.TrimSuffix(bp, "/")
	return fmt.Sprintf("%s://%s%s", scheme, r.Host, bp)
}
