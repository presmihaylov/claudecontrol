// main.go
package main

import (
	"bufio"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

func main() {
	// --- Hardcoded Claude Code OAuth values (from their flow) ---
	const AUTH_URL = "https://claude.ai/oauth/authorize"
	const CLIENT_ID = "9d1c250a-e61b-44d9-88ed-5944d1962f5e"
	const REDIRECT_URI = "https://console.anthropic.com/oauth/code/callback"
	const SCOPE = "org:create_api_key user:profile user:inference"
	// ------------------------------------------------------------

	// Dynamic values (PKCE + state)
	codeVerifier := randomURLSafe(64)       // must be 43-128 chars after encoding
	codeChallenge := pkceS256(codeVerifier) // BASE64URL(SHA256(verifier))
	state := randomURLSafe(24)

	// Build the authorization URL
	u, err := url.Parse(AUTH_URL)
	check(err)
	q := u.Query()
	q.Set("code", "true") // anthopic-specific flag seen in their flow
	q.Set("response_type", "code")
	q.Set("client_id", CLIENT_ID)
	q.Set("redirect_uri", REDIRECT_URI)
	q.Set("scope", SCOPE)
	q.Set("state", state)
	q.Set("code_challenge", codeChallenge)
	q.Set("code_challenge_method", "S256")
	u.RawQuery = q.Encode()

	fmt.Println("==== Claude Code OAuth (Manual Copy-Paste) ====")
	fmt.Println("Open this URL to authorize:")
	fmt.Println(u.String())

	_ = openBrowser(u.String())

	// Prompt user to paste the ?code= value from the redirect URL
	reader := bufio.NewReader(os.Stdin)
	fmt.Println("\nAfter authorizing, you'll land at:")
	fmt.Println(REDIRECT_URI)
	fmt.Println("Copy the `code` query parameter from that URL and paste it below.")
	fmt.Print("Paste code: ")
	code, _ := reader.ReadString('\n')
	code = strings.TrimSpace(code)

	fmt.Println("\n--- Received ---")
	fmt.Printf("code: %s\n", code)
	fmt.Println("\n--- Keep these for the token exchange ---")
	fmt.Printf("code_verifier: %s\n", codeVerifier)
	fmt.Printf("state (issued): %s\n", state)

	// Next (not implemented here): POST to token endpoint with:
	// grant_type=authorization_code
	// code=<pasted code>
	// client_id=<CLIENT_ID>
	// redirect_uri=<REDIRECT_URI>
	// code_verifier=<codeVerifier>
}

// ---- helpers ----

func randomURLSafe(n int) string {
	b := make([]byte, n)
	_, err := rand.Read(b)
	check(err)
	s := base64.RawURLEncoding.EncodeToString(b) // no padding
	// Clamp to 43..128 chars for PKCE verifier
	if len(s) > 128 {
		s = s[:128]
	}
	for len(s) < 43 {
		s += "A"
	}
	return s
}

func pkceS256(verifier string) string {
	sum := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(sum[:]) // no padding
}

func openBrowser(u string) error {
	switch runtime.GOOS {
	case "darwin":
		return exec.Command("open", u).Start()
	case "windows":
		return exec.Command("rundll32", "url.dll,FileProtocolHandler", u).Start()
	default:
		return exec.Command("xdg-open", u).Start()
	}
}

func check(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}