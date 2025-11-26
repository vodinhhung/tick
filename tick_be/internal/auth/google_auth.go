package auth

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

// Config holds the Google OAuth2 configuration
var googleOauthConfig *oauth2.Config

// Random string to prevent CSRF attacks
var oauthStateString = "random-string-should-be-generated-per-request"

func init() {
	// ideally, load these from .env files
	googleOauthConfig = &oauth2.Config{
		ClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
		ClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
		RedirectURL:  "http://localhost:8080/auth/google/callback",
		Scopes: []string{
			"https://www.googleapis.com/auth/userinfo.email",
			"https://www.googleapis.com/auth/userinfo.profile",
		},
		Endpoint: google.Endpoint,
	}
}

// 1. Home Page
func HandleHome(w http.ResponseWriter, r *http.Request) {
	var html = `<html><body><a href="/auth/google/login">Google Login</a></body></html>`
	fmt.Fprint(w, html)
}

// 2. Redirect to Google
func HandleGoogleLogin(w http.ResponseWriter, r *http.Request) {
	// In production, generate a random state string, save it in a cookie,
	// and verify it in the callback to prevent CSRF.
	// For simplicity, we are using a static variable here.
	url := googleOauthConfig.AuthCodeURL(oauthStateString)
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

// 3. Handle Callback from Google
func HandleGoogleCallback(w http.ResponseWriter, r *http.Request) {
	// A. Validate State (CSRF protection)
	state := r.FormValue("state")
	if state != oauthStateString {
		fmt.Println("invalid oauth state")
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		return
	}

	// B. Exchange code for token
	code := r.FormValue("code")
	token, err := googleOauthConfig.Exchange(context.Background(), code)
	if err != nil {
		fmt.Printf("code exchange failed: %s\n", err.Error())
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		return
	}

	// C. Fetch User Data
	resp, err := http.Get("https://www.googleapis.com/oauth2/v2/userinfo?access_token=" + token.AccessToken)
	if err != nil {
		fmt.Printf("failed getting user info: %s\n", err.Error())
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		return
	}
	defer resp.Body.Close()

	contents, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("failed reading response body: %s\n", err.Error())
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		return
	}

	// D. Print User Info (or create session/JWT)
	fmt.Fprintf(w, "UserInfo: %s\n", contents)
}
