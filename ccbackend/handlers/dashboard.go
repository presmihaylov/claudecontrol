package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/gorilla/mux"

	"ccbackend/appctx"
	"ccbackend/middleware"
	"ccbackend/models"
	"ccbackend/models/api"
	"ccbackend/services"
)

type DashboardAPIHandler struct {
	usersService             *services.UsersService
	slackIntegrationsService *services.SlackIntegrationsService
}

func NewDashboardAPIHandler(usersService *services.UsersService, slackIntegrationsService *services.SlackIntegrationsService) *DashboardAPIHandler {
	return &DashboardAPIHandler{
		usersService:             usersService,
		slackIntegrationsService: slackIntegrationsService,
	}
}

func (h *DashboardAPIHandler) HandleUserAuthenticate(w http.ResponseWriter, r *http.Request) {
	log.Printf("🔐 User authentication request received from %s", r.RemoteAddr)

	if r.Method != http.MethodPost {
		log.Printf("❌ Invalid method: %s", r.Method)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get user entity from context (set by authentication middleware)
	user, ok := appctx.GetUser(r.Context())
	if !ok {
		log.Printf("❌ User not found in context")
		http.Error(w, "authentication required", http.StatusUnauthorized)
		return
	}

	log.Printf("✅ User data retrieved from context: %s", user.ID)

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

type CCAgentSecretKeyResponse struct {
	SecretKey string `json:"secret_key"`
}

func (h *DashboardAPIHandler) HandleListSlackIntegrations(w http.ResponseWriter, r *http.Request) {
	log.Printf("📋 List Slack integrations request received from %s", r.RemoteAddr)

	// Get user entity from context (set by authentication middleware)
	user, ok := appctx.GetUser(r.Context())
	if !ok {
		log.Printf("❌ User not found in context")
		http.Error(w, "authentication required", http.StatusUnauthorized)
		return
	}

	h.handleListSlackIntegrations(w, r, user)
}

func (h *DashboardAPIHandler) HandleCreateSlackIntegration(w http.ResponseWriter, r *http.Request) {
	log.Printf("➕ Create Slack integration request received from %s", r.RemoteAddr)

	// Get user entity from context (set by authentication middleware)
	user, ok := appctx.GetUser(r.Context())
	if !ok {
		log.Printf("❌ User not found in context")
		http.Error(w, "authentication required", http.StatusUnauthorized)
		return
	}

	h.handleCreateSlackIntegration(w, r, user)
}

func (h *DashboardAPIHandler) HandleDeleteSlackIntegration(w http.ResponseWriter, r *http.Request) {
	log.Printf("🗑️ Delete Slack integration request received from %s", r.RemoteAddr)

	h.handleDeleteSlackIntegration(w, r)
}

func (h *DashboardAPIHandler) HandleGenerateCCAgentSecretKey(w http.ResponseWriter, r *http.Request) {
	log.Printf("🔑 Generate CCAgent secret key request received from %s", r.RemoteAddr)

	h.handleGenerateCCAgentSecretKey(w, r)
}

func (h *DashboardAPIHandler) handleListSlackIntegrations(w http.ResponseWriter, r *http.Request, user *models.User) {
	log.Printf("📋 Listing Slack integrations for user: %s", user.ID)

	integrations, err := h.slackIntegrationsService.GetSlackIntegrationsByUserID(user.ID)
	if err != nil {
		log.Printf("❌ Failed to get Slack integrations: %v", err)
		http.Error(w, "failed to get slack integrations", http.StatusInternalServerError)
		return
	}

	log.Printf("✅ Retrieved %d Slack integrations for user: %s", len(integrations), user.ID)

	// Convert domain integrations to API models
	apiIntegrations := api.DomainSlackIntegrationsToAPISlackIntegrations(integrations)

	// Return integrations data
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(apiIntegrations); err != nil {
		log.Printf("❌ Failed to encode integrations response: %v", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
}

func (h *DashboardAPIHandler) handleCreateSlackIntegration(w http.ResponseWriter, r *http.Request, user *models.User) {
	log.Printf("➕ Creating Slack integration for user: %s", user.ID)

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

	// Create Slack integration using the authenticated user ID
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

func (h *DashboardAPIHandler) handleDeleteSlackIntegration(w http.ResponseWriter, r *http.Request) {
	log.Printf("🗑️ Deleting Slack integration")

	// Extract integration ID from URL path parameters
	vars := mux.Vars(r)
	integrationIDStr, ok := vars["id"]
	if !ok || integrationIDStr == "" {
		log.Printf("❌ Missing integration ID in URL path")
		http.Error(w, "integration ID is required", http.StatusBadRequest)
		return
	}

	integrationID, err := uuid.Parse(integrationIDStr)
	if err != nil {
		log.Printf("❌ Invalid integration ID format: %v", err)
		http.Error(w, "invalid integration ID format", http.StatusBadRequest)
		return
	}

	// Delete the integration (service will get user from context)
	if err := h.slackIntegrationsService.DeleteSlackIntegration(r.Context(), integrationID); err != nil {
		log.Printf("❌ Failed to delete Slack integration: %v", err)
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, "integration not found", http.StatusNotFound)
		} else {
			http.Error(w, "failed to delete slack integration", http.StatusInternalServerError)
		}
		return
	}

	log.Printf("✅ Slack integration deleted successfully: %s", integrationID)

	// Return success response
	w.WriteHeader(http.StatusNoContent)
}

func (h *DashboardAPIHandler) handleGenerateCCAgentSecretKey(w http.ResponseWriter, r *http.Request) {
	log.Printf("🔑 Generating CCAgent secret key")

	// Extract integration ID from URL path parameters
	vars := mux.Vars(r)
	integrationIDStr, ok := vars["id"]
	if !ok || integrationIDStr == "" {
		log.Printf("❌ Missing integration ID in URL path")
		http.Error(w, "integration ID is required", http.StatusBadRequest)
		return
	}

	integrationID, err := uuid.Parse(integrationIDStr)
	if err != nil {
		log.Printf("❌ Invalid integration ID format: %v", err)
		http.Error(w, "invalid integration ID format", http.StatusBadRequest)
		return
	}

	// Generate the secret key (service will get user from context)
	secretKey, err := h.slackIntegrationsService.GenerateCCAgentSecretKey(r.Context(), integrationID)
	if err != nil {
		log.Printf("❌ Failed to generate CCAgent secret key: %v", err)
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, "integration not found", http.StatusNotFound)
		} else {
			http.Error(w, "failed to generate secret key", http.StatusInternalServerError)
		}
		return
	}

	log.Printf("✅ CCAgent secret key generated successfully for integration: %s", integrationID)

	// Return the secret key response
	response := CCAgentSecretKeyResponse{
		SecretKey: secretKey,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("❌ Failed to encode secret key response: %v", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
}

func (h *DashboardAPIHandler) SetupEndpoints(router *mux.Router, authMiddleware *middleware.ClerkAuthMiddleware) {
	log.Printf("🚀 Registering dashboard API endpoints")

	// User authentication endpoint
	router.HandleFunc("/users/authenticate", authMiddleware.WithAuth(h.HandleUserAuthenticate)).Methods("POST")
	log.Printf("✅ POST /users/authenticate endpoint registered")

	// Slack integrations endpoints
	router.HandleFunc("/slack/integrations", authMiddleware.WithAuth(h.HandleListSlackIntegrations)).Methods("GET")
	log.Printf("✅ GET /slack/integrations endpoint registered")

	router.HandleFunc("/slack/integrations", authMiddleware.WithAuth(h.HandleCreateSlackIntegration)).Methods("POST")
	log.Printf("✅ POST /slack/integrations endpoint registered")

	router.HandleFunc("/slack/integrations/{id}", authMiddleware.WithAuth(h.HandleDeleteSlackIntegration)).Methods("DELETE")
	log.Printf("✅ DELETE /slack/integrations/{id} endpoint registered")

	router.HandleFunc("/slack/integrations/{id}/ccagent_secret_key", authMiddleware.WithAuth(h.HandleGenerateCCAgentSecretKey)).Methods("POST")
	log.Printf("✅ POST /slack/integrations/{id}/ccagent_secret_key endpoint registered")

	log.Printf("✅ All dashboard API endpoints registered successfully")
}
