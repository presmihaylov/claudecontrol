package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"ccbackend/models/api"
	"ccbackend/services"

	"github.com/clerk/clerk-sdk-go/v2"
	"github.com/clerk/clerk-sdk-go/v2/jwt"
	"github.com/clerk/clerk-sdk-go/v2/jwks"
)

type DashboardAPIHandler struct {
	usersService             *services.UsersService
	slackIntegrationsService *services.SlackIntegrationsService
	clerkJWKS                *jwks.Client
}

func NewDashboardAPIHandler(usersService *services.UsersService, slackIntegrationsService *services.SlackIntegrationsService, clerkSecretKey string) *DashboardAPIHandler {
	// Create JWKS client for JWT verification
	config := &clerk.ClientConfig{
		BackendConfig: clerk.BackendConfig{
			Key: clerk.String(clerkSecretKey),
		},
	}
	jwksClient := jwks.NewClient(config)
	
	return &DashboardAPIHandler{
		usersService:             usersService,
		slackIntegrationsService: slackIntegrationsService,
		clerkJWKS:                jwksClient,
	}
}

func (h *DashboardAPIHandler) HandleUserAuthenticate(w http.ResponseWriter, r *http.Request) {
	log.Printf("🔐 User authentication request received from %s", r.RemoteAddr)

	if r.Method != http.MethodPost {
		log.Printf("❌ Invalid method: %s", r.Method)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract bearer token from Authorization header
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		log.Printf("❌ Missing Authorization header")
		http.Error(w, "missing authorization header", http.StatusUnauthorized)
		return
	}

	if !strings.HasPrefix(authHeader, "Bearer ") {
		log.Printf("❌ Invalid Authorization header format")
		http.Error(w, "invalid authorization header format", http.StatusUnauthorized)
		return
	}

	token := strings.TrimPrefix(authHeader, "Bearer ")
	if token == "" {
		log.Printf("❌ Empty bearer token")
		http.Error(w, "empty bearer token", http.StatusUnauthorized)
		return
	}

	// Verify JWT token using Clerk SDK
	claims, err := jwt.Verify(r.Context(), &jwt.VerifyParams{
		Token:      token,
		JWKSClient: h.clerkJWKS,
	})
	if err != nil {
		log.Printf("❌ JWT verification failed: %v", err)
		http.Error(w, "invalid token", http.StatusUnauthorized)
		return
	}

	log.Printf("✅ JWT token verified successfully for user: %s", claims.Subject)

	// Get or create user in database
	user, err := h.usersService.GetOrCreateUser("clerk", claims.Subject)
	if err != nil {
		log.Printf("❌ Failed to get or create user: %v", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	log.Printf("✅ User authenticated successfully: %s", user.ID)

	// Convert domain user to API model
	apiUser := api.DomainUserToAPIUser(user)

	// Return user data
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(apiUser); err != nil {
		log.Printf("❌ Failed to encode user response: %v", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
}

type SlackIntegrationRequest struct {
	SlackAuthToken string `json:"slackAuthToken"`
	RedirectURL    string `json:"redirectUrl"`
}

func (h *DashboardAPIHandler) HandleSlackIntegration(w http.ResponseWriter, r *http.Request) {
	log.Printf("🔗 Slack integration request received from %s", r.RemoteAddr)

	if r.Method != http.MethodPost {
		log.Printf("❌ Invalid method: %s", r.Method)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract bearer token from Authorization header
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		log.Printf("❌ Missing Authorization header")
		http.Error(w, "missing authorization header", http.StatusUnauthorized)
		return
	}

	if !strings.HasPrefix(authHeader, "Bearer ") {
		log.Printf("❌ Invalid Authorization header format")
		http.Error(w, "invalid authorization header format", http.StatusUnauthorized)
		return
	}

	token := strings.TrimPrefix(authHeader, "Bearer ")
	if token == "" {
		log.Printf("❌ Empty bearer token")
		http.Error(w, "empty bearer token", http.StatusUnauthorized)
		return
	}

	// Verify JWT token using Clerk SDK
	claims, err := jwt.Verify(r.Context(), &jwt.VerifyParams{
		Token:      token,
		JWKSClient: h.clerkJWKS,
	})
	if err != nil {
		log.Printf("❌ JWT verification failed: %v", err)
		http.Error(w, "invalid token", http.StatusUnauthorized)
		return
	}

	log.Printf("✅ JWT token verified successfully for user: %s", claims.Subject)

	// Parse request body
	var req SlackIntegrationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("❌ Failed to parse request body: %v", err)
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.SlackAuthToken == "" {
		log.Printf("❌ Missing slackAuthToken in request")
		http.Error(w, "slackAuthToken is required", http.StatusBadRequest)
		return
	}

	// Get or create user to ensure they exist in the database
	user, err := h.usersService.GetOrCreateUser("clerk", claims.Subject)
	if err != nil {
		log.Printf("❌ Failed to get or create user: %v", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	// Create Slack integration
	integration, err := h.slackIntegrationsService.CreateSlackIntegration(req.SlackAuthToken, req.RedirectURL, user.ID)
	if err != nil {
		log.Printf("❌ Failed to create Slack integration: %v", err)
		http.Error(w, "failed to create slack integration", http.StatusInternalServerError)
		return
	}

	log.Printf("✅ Slack integration created successfully: %s", integration.ID)

	// Convert domain integration to API model
	apiIntegration := api.DomainSlackIntegrationToAPISlackIntegration(integration)

	// Return integration data
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(apiIntegration); err != nil {
		log.Printf("❌ Failed to encode integration response: %v", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
}

func (h *DashboardAPIHandler) SetupEndpoints() {
	log.Printf("🚀 Registering dashboard API endpoint on /users/authenticate")
	http.HandleFunc("/users/authenticate", h.HandleUserAuthenticate)
	log.Printf("✅ Dashboard API endpoint registered successfully")

	log.Printf("🚀 Registering Slack integration API endpoint on /slack/integrations")
	http.HandleFunc("/slack/integrations", h.HandleSlackIntegration)
	log.Printf("✅ Slack integration API endpoint registered successfully")
}