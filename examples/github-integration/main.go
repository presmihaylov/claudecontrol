package main

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

const (
	githubInstallURL = "https://github.com/apps/claude-control/installations/select_target"
	localPort        = "8080"
	callbackPath     = "/callback"
)

var (
	targetRepo string
)

func main() {
	fmt.Println("🔧 GitHub Integration Example")
	fmt.Println("==============================")

	err := godotenv.Load()
	if err != nil {
		log.Printf("Warning: Error loading .env file: %v", err)
		log.Printf("Continuing with system environment variables...")
	}

	targetRepo = os.Getenv("GITHUB_TARGET_REPO")

	fmt.Println("✅ GitHub App installation flow ready")
	if targetRepo != "" {
		fmt.Printf("🎯 Target Repository: %s\n", targetRepo)
	} else {
		fmt.Println("🌐 Will install on selected repositories")
	}
	fmt.Println("")

	mux := http.NewServeMux()
	mux.HandleFunc("/", handleHome)
	mux.HandleFunc("/install", handleInstallFlow)
	mux.HandleFunc(callbackPath, handleCallback)

	server := &http.Server{
		Addr:         ":" + localPort,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	fmt.Printf("🚀 Starting server on http://localhost:%s\n", localPort)
	fmt.Println("")
	fmt.Println("🔗 Open this link in your browser to begin GitHub OAuth flow:")
	fmt.Printf("   http://localhost:%s\n", localPort)
	fmt.Println("")
	fmt.Println("⏹️  Press Ctrl+C to stop the server")
	fmt.Println("")

	log.Fatal(server.ListenAndServe())
}

func handleHome(w http.ResponseWriter, r *http.Request) {
	installURL := buildInstallURL()
	
	var repoStep string
	if targetRepo != "" {
		repoStep = fmt.Sprintf(`            <div class="step">2. Install app on repository: <strong>%s</strong></div>`, targetRepo)
	} else {
		repoStep = `            <div class="step">2. Choose which repositories to install the app on</div>`
	}

	html := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>GitHub Integration Example</title>
    <style>
        body { font-family: Arial, sans-serif; max-width: 600px; margin: 50px auto; padding: 20px; }
        .container { text-align: center; }
        .button { 
            background-color: #24292e; 
            color: white; 
            padding: 12px 24px; 
            text-decoration: none; 
            border-radius: 6px; 
            display: inline-block; 
            margin: 20px 0;
            font-size: 16px;
        }
        .button:hover { background-color: #0366d6; }
        .info { background-color: #f6f8fa; padding: 15px; border-radius: 6px; margin: 20px 0; }
        .step { margin: 10px 0; text-align: left; }
    </style>
</head>
<body>
    <div class="container">
        <h1>🔧 GitHub App Installation Example</h1>
        <p>This example demonstrates how to install a GitHub App on repositories.</p>
        
        <div class="info">
            <h3>What this will do:</h3>
            <div class="step">1. Redirect you to GitHub App installation page</div>
            %s
            <div class="step">3. Redirect back with installation details</div>
            <div class="step">4. Display installation ID and repository information</div>
        </div>

        <a href="%s" class="button">🚀 Install GitHub App</a>
        
        <p><small>You'll be redirected to GitHub to install the Claude Control app on your repositories.</small></p>
    </div>
</body>
</html>`, repoStep, installURL)

	w.Header().Set("Content-Type", "text/html")
	fmt.Fprint(w, html)
}

func handleInstallFlow(w http.ResponseWriter, r *http.Request) {
	html := `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>GitHub App Installation</title>
    <style>
        body { font-family: Arial, sans-serif; max-width: 600px; margin: 50px auto; padding: 20px; }
        .container { text-align: center; }
        .info { background-color: #fff3cd; padding: 15px; border-radius: 6px; margin: 20px 0; }
        .step { margin: 10px 0; text-align: left; }
        .warning { background-color: #f8d7da; color: #721c24; padding: 15px; border-radius: 6px; margin: 20px 0; }
    </style>
</head>
<body>
    <div class="container">
        <h1>📁 Repository-Specific Access</h1>
        
        <div class="warning">
            <h3>⚠️ GitHub App Required</h3>
            <p>To select specific repositories during authorization, you need to create a <strong>GitHub App</strong> instead of an OAuth App.</p>
        </div>

        <div class="info">
            <h3>To enable repository selection:</h3>
            <div class="step">1. Go to <a href="https://github.com/settings/apps" target="_blank">GitHub Apps settings</a></div>
            <div class="step">2. Click "New GitHub App"</div>
            <div class="step">3. Configure your app with these settings:</div>
            <div class="step" style="margin-left: 20px;">- Homepage URL: http://localhost:8080</div>
            <div class="step" style="margin-left: 20px;">- Authorization callback URL: http://localhost:8080/callback</div>
            <div class="step" style="margin-left: 20px;">- Repository permissions: Contents (Read), Metadata (Read)</div>
            <div class="step">4. Install the app on your account/organization</div>
            <div class="step">5. During installation, you can select specific repositories</div>
        </div>

        <p><a href="/">← Back to OAuth Flow</a></p>
        
        <div class="info">
            <h3>💡 Alternative: Use Environment Variable</h3>
            <p>Set <code>GITHUB_TARGET_REPO=owner/repository</code> to focus on a specific repository. The OAuth flow will still grant access to all your repositories, but the example will highlight your target repository.</p>
        </div>
    </div>
</body>
</html>`

	w.Header().Set("Content-Type", "text/html")
	fmt.Fprint(w, html)
}

func handleCallback(w http.ResponseWriter, r *http.Request) {
	installationID := r.URL.Query().Get("installation_id")
	setupAction := r.URL.Query().Get("setup_action")
	
	if installationID == "" {
		http.Error(w, "No installation ID received", http.StatusBadRequest)
		return
	}

	fmt.Printf("\n🎉 GitHub App Installation Completed!\n")
	fmt.Printf("Installation ID: %s\n", installationID)
	fmt.Printf("Setup Action: %s\n", setupAction)
	
	// Get additional installation details
	repos := getInstallationRepos(installationID)
	account := getInstallationAccount(installationID)

	html := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>GitHub App Installation - Success!</title>
    <style>
        body { font-family: Arial, sans-serif; max-width: 800px; margin: 50px auto; padding: 20px; }
        .success { background-color: #d4edda; color: #155724; padding: 15px; border-radius: 6px; margin: 20px 0; }
        .installation-box { 
            background-color: #f8f9fa; 
            border: 1px solid #dee2e6; 
            padding: 15px; 
            border-radius: 6px; 
            font-family: monospace; 
            word-break: break-all;
            margin: 20px 0;
        }
        .instructions { background-color: #e7f3ff; padding: 15px; border-radius: 6px; margin: 20px 0; }
        .repo-list { text-align: left; margin: 10px 0; }
        .repo-item { margin: 5px 0; padding: 5px; background-color: #f6f8fa; border-radius: 3px; }
        .info-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 20px; margin: 20px 0; }
        .info-card { background-color: #f8f9fa; padding: 15px; border-radius: 6px; }
        pre { background-color: #f6f8fa; padding: 10px; border-radius: 6px; overflow-x: auto; }
    </style>
</head>
<body>
    <div class="success">
        <h2>🎉 GitHub App Successfully Installed!</h2>
        <p><strong>Installed on:</strong> %s</p>
    </div>

    <div class="info-grid">
        <div class="info-card">
            <h3>📋 Installation Details</h3>
            <p><strong>Installation ID:</strong> %s</p>
            <p><strong>Setup Action:</strong> %s</p>
        </div>
        <div class="info-card">
            <h3>📁 Repositories</h3>
            <div class="repo-list">%s</div>
        </div>
    </div>

    <div class="instructions">
        <h3>📖 What's Next:</h3>
        
        <h4>1. Save the Installation ID:</h4>
        <div class="installation-box">Installation ID: %s</div>
        
        <h4>2. Use Installation ID for API calls:</h4>
        <pre># Get installation repositories
curl -H "Accept: application/vnd.github.v3+json" \
     https://api.github.com/app/installations/%s/repositories

# Get installation details  
curl -H "Accept: application/vnd.github.v3+json" \
     https://api.github.com/app/installations/%s</pre>
        
        <h4>3. Generate Installation Access Token:</h4>
        <pre># Use your GitHub App's private key to generate installation tokens
# See: https://docs.github.com/en/developers/apps/building-github-apps/authenticating-with-github-apps</pre>
    </div>

    <div class="instructions">
        <h3>⚠️ Important Notes:</h3>
        <ul>
            <li>Save the Installation ID (%s) - you'll need it for API calls</li>
            <li>The app is now installed on the selected repositories</li>
            <li>You can manage installations at: https://github.com/settings/installations</li>
            <li>Use the Installation ID to generate access tokens for API calls</li>
        </ul>
    </div>

    <p><small>You can now close this window. The GitHub App is successfully installed!</small></p>
</body>
</html>`, account, installationID, setupAction, formatRepoList(repos), installationID, installationID, installationID, installationID)

	w.Header().Set("Content-Type", "text/html")
	fmt.Fprint(w, html)

	fmt.Printf("📋 Installed Repositories: %v\n", repos)
	fmt.Printf("🏢 Account: %s\n", account)
	fmt.Println("\n📖 Next Steps:")
	fmt.Println("  1. Save Installation ID: " + installationID)
	fmt.Println("  2. Use Installation ID for GitHub App API calls")
	fmt.Println("  3. Generate installation access tokens using your app's private key")
	fmt.Println("\n📚 See: https://docs.github.com/en/developers/apps/building-github-apps/authenticating-with-github-apps")
}

func buildInstallURL() string {
	params := url.Values{}
	params.Set("state", fmt.Sprintf("redirect_uri=%s", fmt.Sprintf("http://localhost:%s%s", localPort, callbackPath)))
	
	return githubInstallURL + "?" + params.Encode()
}

func getInstallationRepos(installationID string) []string {
	// In a real implementation, you would use the GitHub API with an installation access token
	// For this example, we'll return placeholder data
	return []string{
		"📁 Selected repositories from installation",
		"💡 Use GitHub API to fetch actual repository list",
		"🔗 GET /app/installations/" + installationID + "/repositories",
	}
}

func getInstallationAccount(installationID string) string {
	// In a real implementation, you would fetch this from GitHub API
	// For this example, we'll return placeholder data
	return "Account/Organization (use API to fetch actual name)"
}

func formatRepoList(repos []string) string {
	if len(repos) == 0 {
		return "<div class=\"repo-item\">No repositories found</div>"
	}

	var formatted strings.Builder
	for _, repo := range repos {
		if strings.HasPrefix(repo, "🎯") || strings.HasPrefix(repo, "❌") {
			// Already has emoji
			formatted.WriteString(fmt.Sprintf("<div class=\"repo-item\">%s</div>", repo))
		} else {
			// Add folder emoji
			formatted.WriteString(fmt.Sprintf("<div class=\"repo-item\">📁 %s</div>", repo))
		}
	}
	return formatted.String()
}

