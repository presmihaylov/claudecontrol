package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"github.com/gorilla/mux"

	"ccbackend/appctx"
	"ccbackend/core"
	"ccbackend/middleware"
	"ccbackend/models"
	"ccbackend/models/api"
)

type DashboardHTTPHandler struct {
	handler *DashboardAPIHandler
}

func NewDashboardHTTPHandler(handler *DashboardAPIHandler) *DashboardHTTPHandler {
	return &DashboardHTTPHandler{
		handler: handler,
	}
}

type SlackIntegrationRequest struct {
	SlackAuthToken string `json:"slackAuthToken"`
	RedirectURL    string `json:"redirectUrl"`
}

type DiscordIntegrationRequest struct {
	DiscordAuthCode string `json:"code"`
	GuildID         string `json:"guild_id"`
	RedirectURL     string `json:"redirect_url"`
}

type GitHubIntegrationRequest struct {
	Code           string `json:"code"`
	InstallationID string `json:"installation_id"`
}

type CCAgentSecretKeyResponse struct {
	SecretKey   string `json:"secret_key"`
	GeneratedAt string `json:"generated_at"`
}

type CCAgentContainerIntegrationCreateRequest struct {
	InstancesCount int    `json:"instances_count"`
	RepoURL        string `json:"repo_url"`
}

func (h *DashboardHTTPHandler) HandleUserAuthenticate(w http.ResponseWriter, r *http.Request) {
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
	h.writeJSONResponse(w, http.StatusOK, apiUser)
}

func (h *DashboardHTTPHandler) HandleGetUserProfile(w http.ResponseWriter, r *http.Request) {
	log.Printf("👤 Get user profile request received from %s", r.RemoteAddr)

	// Get user entity from context (set by authentication middleware)
	user, ok := appctx.GetUser(r.Context())
	if !ok {
		log.Printf("❌ User not found in context")
		http.Error(w, "authentication required", http.StatusUnauthorized)
		return
	}

	log.Printf("✅ User profile retrieved from context: %s (email: %s)", user.ID, user.Email)

	// Convert domain user to API model
	apiUser := api.DomainUserToAPIUser(user)

	// Return user profile data
	h.writeJSONResponse(w, http.StatusOK, apiUser)
}

func (h *DashboardHTTPHandler) HandleListSlackIntegrations(w http.ResponseWriter, r *http.Request) {
	log.Printf("📋 List Slack integrations request received from %s", r.RemoteAddr)

	// Get user entity from context (set by authentication middleware)
	user, ok := appctx.GetUser(r.Context())
	if !ok {
		log.Printf("❌ User not found in context")
		http.Error(w, "authentication required", http.StatusUnauthorized)
		return
	}

	integrations, err := h.handler.ListSlackIntegrations(r.Context(), user)
	if err != nil {
		log.Printf("❌ Failed to get Slack integrations: %v", err)
		http.Error(w, "failed to get slack integrations", http.StatusInternalServerError)
		return
	}

	// Convert domain integrations to API models
	apiIntegrations := api.DomainSlackIntegrationsToAPISlackIntegrations(integrations)

	h.writeJSONResponse(w, http.StatusOK, apiIntegrations)
}

func (h *DashboardHTTPHandler) HandleCreateSlackIntegration(w http.ResponseWriter, r *http.Request) {
	log.Printf("➕ Create Slack integration request received from %s", r.RemoteAddr)

	// Get user entity from context (set by authentication middleware)
	user, ok := appctx.GetUser(r.Context())
	if !ok {
		log.Printf("❌ User not found in context")
		http.Error(w, "authentication required", http.StatusUnauthorized)
		return
	}

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

	integration, err := h.handler.CreateSlackIntegration(r.Context(), req.SlackAuthToken, req.RedirectURL, user)
	if err != nil {
		log.Printf("❌ Failed to create Slack integration: %v", err)
		http.Error(w, "failed to create slack integration", http.StatusInternalServerError)
		return
	}

	// Convert domain integration to API model
	apiIntegration := api.DomainSlackIntegrationToAPISlackIntegration(integration)

	h.writeJSONResponse(w, http.StatusOK, apiIntegration)
}

func (h *DashboardHTTPHandler) HandleDeleteSlackIntegration(w http.ResponseWriter, r *http.Request) {
	log.Printf("🗑️ Delete Slack integration request received from %s", r.RemoteAddr)

	// Extract integration ID from URL path parameters
	vars := mux.Vars(r)
	integrationIDStr, ok := vars["id"]
	if !ok || !core.IsValidULID(integrationIDStr) {
		log.Printf("❌ Missing or invalid integration ID in URL path")
		http.Error(w, "integration ID must be a valid ULID", http.StatusBadRequest)
		return
	}

	err := h.handler.DeleteSlackIntegration(r.Context(), integrationIDStr)
	if err != nil {
		log.Printf("❌ Failed to delete Slack integration: %v", err)
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, "integration not found", http.StatusNotFound)
		} else {
			http.Error(w, "failed to delete slack integration", http.StatusInternalServerError)
		}
		return
	}

	log.Printf("✅ Slack integration deleted successfully: %s", integrationIDStr)

	// Return success response
	w.WriteHeader(http.StatusNoContent)
}

func (h *DashboardHTTPHandler) HandleListDiscordIntegrations(w http.ResponseWriter, r *http.Request) {
	log.Printf("📋 List Discord integrations request received from %s", r.RemoteAddr)

	user, ok := appctx.GetUser(r.Context())
	if !ok {
		log.Printf("❌ User not found in context")
		http.Error(w, "authentication required", http.StatusUnauthorized)
		return
	}

	integrations, err := h.handler.ListDiscordIntegrations(r.Context(), user)
	if err != nil {
		log.Printf("❌ Failed to get Discord integrations: %v", err)
		http.Error(w, "failed to get discord integrations", http.StatusInternalServerError)
		return
	}

	apiIntegrations := api.DomainDiscordIntegrationsToAPIDiscordIntegrations(integrations)

	h.writeJSONResponse(w, http.StatusOK, apiIntegrations)
}

func (h *DashboardHTTPHandler) HandleCreateDiscordIntegration(w http.ResponseWriter, r *http.Request) {
	log.Printf("➕ Create Discord integration request received from %s", r.RemoteAddr)

	user, ok := appctx.GetUser(r.Context())
	if !ok {
		log.Printf("❌ User not found in context")
		http.Error(w, "authentication required", http.StatusUnauthorized)
		return
	}

	var req DiscordIntegrationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("❌ Failed to parse request body: %v", err)
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.DiscordAuthCode == "" {
		log.Printf("❌ Missing code in request")
		http.Error(w, "code is required", http.StatusBadRequest)
		return
	}

	if req.GuildID == "" {
		log.Printf("❌ Missing guild_id in request")
		http.Error(w, "guild_id is required", http.StatusBadRequest)
		return
	}

	integration, err := h.handler.CreateDiscordIntegration(
		r.Context(),
		req.DiscordAuthCode,
		req.GuildID,
		req.RedirectURL,
		user,
	)
	if err != nil {
		log.Printf("❌ Failed to create Discord integration: %v", err)
		http.Error(w, "failed to create discord integration", http.StatusInternalServerError)
		return
	}

	apiIntegration := api.DomainDiscordIntegrationToAPIDiscordIntegration(integration)

	h.writeJSONResponse(w, http.StatusOK, apiIntegration)
}

func (h *DashboardHTTPHandler) HandleDeleteDiscordIntegration(w http.ResponseWriter, r *http.Request) {
	log.Printf("🗑️ Delete Discord integration request received from %s", r.RemoteAddr)

	vars := mux.Vars(r)
	integrationIDStr, ok := vars["id"]
	if !ok || !core.IsValidULID(integrationIDStr) {
		log.Printf("❌ Missing or invalid integration ID in URL path")
		http.Error(w, "integration ID must be a valid ULID", http.StatusBadRequest)
		return
	}

	err := h.handler.DeleteDiscordIntegration(r.Context(), integrationIDStr)
	if err != nil {
		log.Printf("❌ Failed to delete Discord integration: %v", err)
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, "integration not found", http.StatusNotFound)
		} else {
			http.Error(w, "failed to delete discord integration", http.StatusInternalServerError)
		}
		return
	}

	log.Printf("✅ Discord integration deleted successfully: %s", integrationIDStr)

	w.WriteHeader(http.StatusNoContent)
}

func (h *DashboardHTTPHandler) HandleGetOrganization(w http.ResponseWriter, r *http.Request) {
	log.Printf("📋 Get organization request received from %s", r.RemoteAddr)

	organization, err := h.handler.GetOrganization(r.Context())
	if err != nil {
		log.Printf("❌ Failed to get organization: %v", err)
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, "organization not found", http.StatusNotFound)
		} else {
			http.Error(w, "failed to get organization", http.StatusInternalServerError)
		}
		return
	}

	log.Printf("✅ Organization retrieved successfully: %s", organization.ID)

	h.writeJSONResponse(w, http.StatusOK, organization)
}

func (h *DashboardHTTPHandler) HandleGenerateCCAgentSecretKey(w http.ResponseWriter, r *http.Request) {
	log.Printf("🔑 Generate CCAgent secret key request received from %s", r.RemoteAddr)

	secretKey, err := h.handler.GenerateCCAgentSecretKey(r.Context())
	if err != nil {
		log.Printf("❌ Failed to generate CCAgent secret key: %v", err)
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, "organization not found", http.StatusNotFound)
		} else {
			http.Error(w, "failed to generate secret key", http.StatusInternalServerError)
		}
		return
	}

	// Get organization from context to get the timestamp
	org, ok := appctx.GetOrganization(r.Context())
	if !ok {
		log.Printf("❌ Organization not found in context")
		http.Error(w, "organization not found", http.StatusNotFound)
		return
	}

	log.Printf("✅ CCAgent secret key generated successfully")

	// Return the secret key response with timestamp
	generatedAt := ""
	if org.CCAgentSecretKeyGeneratedAt != nil {
		generatedAt = org.CCAgentSecretKeyGeneratedAt.Format("2006-01-02T15:04:05Z07:00")
	}

	response := CCAgentSecretKeyResponse{
		SecretKey:   secretKey,
		GeneratedAt: generatedAt,
	}

	h.writeJSONResponse(w, http.StatusOK, response)
}

func (h *DashboardHTTPHandler) HandleListGitHubIntegrations(w http.ResponseWriter, r *http.Request) {
	log.Printf("📋 List GitHub integrations request received from %s", r.RemoteAddr)

	// Get user entity from context (set by authentication middleware)
	user, ok := appctx.GetUser(r.Context())
	if !ok {
		log.Printf("❌ User not found in context")
		http.Error(w, "authentication required", http.StatusUnauthorized)
		return
	}

	integrations, err := h.handler.ListGitHubIntegrations(r.Context(), user)
	if err != nil {
		log.Printf("❌ Failed to list GitHub integrations: %v", err)
		http.Error(w, "failed to list GitHub integrations", http.StatusInternalServerError)
		return
	}

	// Convert domain integrations to API models
	apiIntegrations := api.DomainGitHubIntegrationsToAPIGitHubIntegrations(integrations)

	log.Printf("✅ Successfully retrieved %d GitHub integrations", len(integrations))
	h.writeJSONResponse(w, http.StatusOK, apiIntegrations)
}

func (h *DashboardHTTPHandler) HandleCreateGitHubIntegration(w http.ResponseWriter, r *http.Request) {
	log.Printf("➕ Create GitHub integration request received from %s", r.RemoteAddr)

	// Get user entity from context (set by authentication middleware)
	user, ok := appctx.GetUser(r.Context())
	if !ok {
		log.Printf("❌ User not found in context")
		http.Error(w, "authentication required", http.StatusUnauthorized)
		return
	}

	var req GitHubIntegrationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("❌ Failed to decode request: %v", err)
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.Code == "" {
		log.Printf("❌ Missing code in request")
		http.Error(w, "code is required", http.StatusBadRequest)
		return
	}

	if req.InstallationID == "" {
		log.Printf("❌ Missing installation_id in request")
		http.Error(w, "installation_id is required", http.StatusBadRequest)
		return
	}

	integration, err := h.handler.CreateGitHubIntegration(r.Context(), req.Code, req.InstallationID, user)
	if err != nil {
		log.Printf("❌ Failed to create GitHub integration: %v", err)
		if strings.Contains(err.Error(), "verify") || strings.Contains(err.Error(), "OAuth") {
			http.Error(w, "failed to verify GitHub installation", http.StatusUnauthorized)
		} else {
			http.Error(w, "failed to create GitHub integration", http.StatusInternalServerError)
		}
		return
	}

	// Convert domain integration to API model
	apiIntegration := api.DomainGitHubIntegrationToAPIGitHubIntegration(integration)

	log.Printf("✅ GitHub integration created successfully: %s", integration.ID)
	h.writeJSONResponse(w, http.StatusCreated, apiIntegration)
}

func (h *DashboardHTTPHandler) HandleGetGitHubIntegrationByID(w http.ResponseWriter, r *http.Request) {
	log.Printf("📋 Get GitHub integration request received from %s", r.RemoteAddr)

	vars := mux.Vars(r)
	integrationIDStr, ok := vars["id"]
	if !ok || !core.IsValidULID(integrationIDStr) {
		log.Printf("❌ Missing or invalid integration ID in URL path")
		http.Error(w, "integration ID must be a valid ULID", http.StatusBadRequest)
		return
	}

	integration, err := h.handler.GetGitHubIntegrationByID(r.Context(), integrationIDStr)
	if err != nil {
		log.Printf("❌ Failed to get GitHub integration: %v", err)
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, "integration not found", http.StatusNotFound)
		} else {
			http.Error(w, "failed to get GitHub integration", http.StatusInternalServerError)
		}
		return
	}

	// Convert domain integration to API model
	apiIntegration := api.DomainGitHubIntegrationToAPIGitHubIntegration(integration)

	log.Printf("✅ GitHub integration retrieved successfully: %s", integrationIDStr)
	h.writeJSONResponse(w, http.StatusOK, apiIntegration)
}

func (h *DashboardHTTPHandler) HandleDeleteGitHubIntegration(w http.ResponseWriter, r *http.Request) {
	log.Printf("🗑️ Delete GitHub integration request received from %s", r.RemoteAddr)

	vars := mux.Vars(r)
	integrationIDStr, ok := vars["id"]
	if !ok || !core.IsValidULID(integrationIDStr) {
		log.Printf("❌ Missing or invalid integration ID in URL path")
		http.Error(w, "integration ID must be a valid ULID", http.StatusBadRequest)
		return
	}

	err := h.handler.DeleteGitHubIntegration(r.Context(), integrationIDStr)
	if err != nil {
		log.Printf("❌ Failed to delete GitHub integration: %v", err)
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, "integration not found", http.StatusNotFound)
		} else {
			http.Error(w, "failed to delete GitHub integration", http.StatusInternalServerError)
		}
		return
	}

	log.Printf("✅ GitHub integration deleted successfully: %s", integrationIDStr)
	w.WriteHeader(http.StatusNoContent)
}

// AnthropicIntegrationRequest represents the request body for creating an Anthropic integration
type AnthropicIntegrationRequest struct {
	APIKey       *string `json:"api_key,omitempty"`
	OAuthToken   *string `json:"oauth_token,omitempty"`
	CodeVerifier *string `json:"code_verifier,omitempty"`
}

func (h *DashboardHTTPHandler) HandleListAnthropicIntegrations(w http.ResponseWriter, r *http.Request) {
	log.Printf("📋 List Anthropic integrations request received from %s", r.RemoteAddr)

	// Get user entity from context (set by authentication middleware)
	user, ok := appctx.GetUser(r.Context())
	if !ok {
		log.Printf("❌ User not found in context")
		http.Error(w, "authentication required", http.StatusUnauthorized)
		return
	}

	integrations, err := h.handler.ListAnthropicIntegrations(r.Context(), user)
	if err != nil {
		log.Printf("❌ Failed to list Anthropic integrations: %v", err)
		http.Error(w, "failed to list Anthropic integrations", http.StatusInternalServerError)
		return
	}

	// Convert domain integrations to API models
	apiIntegrations := api.DomainAnthropicIntegrationsToAPIAnthropicIntegrations(integrations)

	log.Printf("✅ Successfully retrieved %d Anthropic integrations", len(integrations))
	h.writeJSONResponse(w, http.StatusOK, apiIntegrations)
}

func (h *DashboardHTTPHandler) HandleCreateAnthropicIntegration(w http.ResponseWriter, r *http.Request) {
	log.Printf("➕ Create Anthropic integration request received from %s", r.RemoteAddr)

	// Get user entity from context (set by authentication middleware)
	user, ok := appctx.GetUser(r.Context())
	if !ok {
		log.Printf("❌ User not found in context")
		http.Error(w, "authentication required", http.StatusUnauthorized)
		return
	}

	var req AnthropicIntegrationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("❌ Failed to decode request: %v", err)
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	// Validate request
	if req.APIKey == nil && req.OAuthToken == nil {
		log.Printf("❌ Neither API key nor OAuth token provided")
		http.Error(w, "either api_key or oauth_token must be provided", http.StatusBadRequest)
		return
	}
	if req.APIKey != nil && req.OAuthToken != nil {
		log.Printf("❌ Both API key and OAuth token provided")
		http.Error(w, "only one of api_key or oauth_token can be provided", http.StatusBadRequest)
		return
	}

	integration, err := h.handler.CreateAnthropicIntegration(
		r.Context(),
		req.APIKey,
		req.OAuthToken,
		req.CodeVerifier,
		user,
	)
	if err != nil {
		log.Printf("❌ Failed to create Anthropic integration: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Convert domain integration to API model
	apiIntegration := api.DomainAnthropicIntegrationToAPIAnthropicIntegration(integration)

	log.Printf("✅ Anthropic integration created successfully: %s", integration.ID)
	h.writeJSONResponse(w, http.StatusCreated, apiIntegration)
}

func (h *DashboardHTTPHandler) HandleGetAnthropicIntegrationByID(w http.ResponseWriter, r *http.Request) {
	log.Printf("📋 Get Anthropic integration request received from %s", r.RemoteAddr)

	vars := mux.Vars(r)
	integrationIDStr, ok := vars["id"]
	if !ok || !core.IsValidULID(integrationIDStr) {
		log.Printf("❌ Missing or invalid integration ID in URL path")
		http.Error(w, "integration ID must be a valid ULID", http.StatusBadRequest)
		return
	}

	integration, err := h.handler.GetAnthropicIntegrationByID(r.Context(), integrationIDStr)
	if err != nil {
		log.Printf("❌ Failed to get Anthropic integration: %v", err)
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, "integration not found", http.StatusNotFound)
		} else {
			http.Error(w, "failed to get Anthropic integration", http.StatusInternalServerError)
		}
		return
	}

	// Convert domain integration to API model
	apiIntegration := api.DomainAnthropicIntegrationToAPIAnthropicIntegration(integration)

	log.Printf("✅ Successfully retrieved Anthropic integration: %s", integrationIDStr)
	h.writeJSONResponse(w, http.StatusOK, apiIntegration)
}

func (h *DashboardHTTPHandler) HandleDeleteAnthropicIntegration(w http.ResponseWriter, r *http.Request) {
	log.Printf("🗑️ Delete Anthropic integration request received from %s", r.RemoteAddr)

	vars := mux.Vars(r)
	integrationIDStr, ok := vars["id"]
	if !ok || !core.IsValidULID(integrationIDStr) {
		log.Printf("❌ Missing or invalid integration ID in URL path")
		http.Error(w, "integration ID must be a valid ULID", http.StatusBadRequest)
		return
	}

	err := h.handler.DeleteAnthropicIntegration(r.Context(), integrationIDStr)
	if err != nil {
		log.Printf("❌ Failed to delete Anthropic integration: %v", err)
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, "integration not found", http.StatusNotFound)
		} else {
			http.Error(w, "failed to delete Anthropic integration", http.StatusInternalServerError)
		}
		return
	}

	log.Printf("✅ Anthropic integration deleted successfully: %s", integrationIDStr)
	w.WriteHeader(http.StatusNoContent)
}

// CCAgent Container Integration Handlers

func (h *DashboardHTTPHandler) HandleListCCAgentContainerIntegrations(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	org, ok := appctx.GetOrganization(ctx)
	if !ok || org == nil {
		http.Error(w, "organization not found in context", http.StatusInternalServerError)
		return
	}

	integrations, err := h.handler.ccAgentContainerService.ListCCAgentContainerIntegrations(ctx, models.OrgID(org.ID))
	if err != nil {
		log.Printf("❌ Failed to list CCAgent container integrations: %v", err)
		http.Error(w, "failed to list CCAgent container integrations", http.StatusInternalServerError)
		return
	}

	h.writeJSONResponse(w, http.StatusOK, integrations)
}

func (h *DashboardHTTPHandler) HandleGetCCAgentContainerIntegrationByID(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	org, ok := appctx.GetOrganization(ctx)
	if !ok || org == nil {
		http.Error(w, "organization not found in context", http.StatusInternalServerError)
		return
	}

	vars := mux.Vars(r)
	integrationID := vars["id"]

	integration, err := h.handler.ccAgentContainerService.GetCCAgentContainerIntegrationByID(
		ctx,
		models.OrgID(org.ID),
		integrationID,
	)
	if err != nil {
		log.Printf("❌ Failed to get CCAgent container integration: %v", err)
		http.Error(w, "failed to get CCAgent container integration", http.StatusInternalServerError)
		return
	}

	if !integration.IsPresent() {
		http.Error(w, "CCAgent container integration not found", http.StatusNotFound)
		return
	}

	h.writeJSONResponse(w, http.StatusOK, integration.MustGet())
}

func (h *DashboardHTTPHandler) HandleCreateCCAgentContainerIntegration(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	org, ok := appctx.GetOrganization(ctx)
	if !ok || org == nil {
		http.Error(w, "organization not found in context", http.StatusInternalServerError)
		return
	}

	var req CCAgentContainerIntegrationCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("❌ Failed to decode request body: %v", err)
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	integration, err := h.handler.ccAgentContainerService.CreateCCAgentContainerIntegration(
		ctx,
		models.OrgID(org.ID),
		req.InstancesCount,
		req.RepoURL,
	)
	if err != nil {
		log.Printf("❌ Failed to create CCAgent container integration: %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	h.writeJSONResponse(w, http.StatusCreated, integration)
}

func (h *DashboardHTTPHandler) HandleDeleteCCAgentContainerIntegration(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	org, ok := appctx.GetOrganization(ctx)
	if !ok || org == nil {
		http.Error(w, "organization not found in context", http.StatusInternalServerError)
		return
	}

	vars := mux.Vars(r)
	integrationID := vars["id"]

	if err := h.handler.ccAgentContainerService.DeleteCCAgentContainerIntegration(ctx, models.OrgID(org.ID), integrationID); err != nil {
		log.Printf("❌ Failed to delete CCAgent container integration: %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	log.Printf("✅ CCAgent container integration deleted successfully: %s", integrationID)
	w.WriteHeader(http.StatusNoContent)
}

func (h *DashboardHTTPHandler) HandleRedeployCCAgentContainer(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	org, ok := appctx.GetOrganization(ctx)
	if !ok || org == nil {
		http.Error(w, "organization not found in context", http.StatusInternalServerError)
		return
	}

	vars := mux.Vars(r)
	integrationID := vars["id"]
	
	log.Printf("🚀 Redeploy CCAgent container request received for integration: %s, org: %s", integrationID, org.ID)

	if err := h.handler.ccAgentContainerService.RedeployCCAgentContainer(ctx, models.OrgID(org.ID), integrationID); err != nil {
		log.Printf("❌ Failed to redeploy CCAgent container: %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	log.Printf("✅ CCAgent container redeployed successfully: %s", integrationID)
	h.writeJSONResponse(w, http.StatusOK, map[string]string{
		"message": "CCAgent container redeployed successfully",
	})
}

func (h *DashboardHTTPHandler) HandleListGitHubRepositories(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	org, ok := appctx.GetOrganization(ctx)
	if !ok || org == nil {
		http.Error(w, "organization not found in context", http.StatusInternalServerError)
		return
	}

	repositories, err := h.handler.githubService.ListAvailableRepositories(ctx, models.OrgID(org.ID))
	if err != nil {
		log.Printf("❌ Failed to list GitHub repositories: %v", err)
		http.Error(w, "failed to list repositories", http.StatusInternalServerError)
		return
	}

	h.writeJSONResponse(w, http.StatusOK, repositories)
}

func (h *DashboardHTTPHandler) SetupEndpoints(router *mux.Router, authMiddleware *middleware.ClerkAuthMiddleware) {
	log.Printf("🚀 Registering dashboard API endpoints")

	// User authentication endpoint
	router.HandleFunc("/users/authenticate", authMiddleware.WithAuth(h.HandleUserAuthenticate)).Methods("POST")
	log.Printf("✅ POST /users/authenticate endpoint registered")

	// User profile endpoint
	router.HandleFunc("/users/profile", authMiddleware.WithAuth(h.HandleGetUserProfile)).Methods("GET")
	log.Printf("✅ GET /users/profile endpoint registered")

	// Slack integrations endpoints
	router.HandleFunc("/slack/integrations", authMiddleware.WithAuth(h.HandleListSlackIntegrations)).Methods("GET")
	log.Printf("✅ GET /slack/integrations endpoint registered")

	router.HandleFunc("/slack/integrations", authMiddleware.WithAuth(h.HandleCreateSlackIntegration)).Methods("POST")
	log.Printf("✅ POST /slack/integrations endpoint registered")

	router.HandleFunc("/slack/integrations/{id}", authMiddleware.WithAuth(h.HandleDeleteSlackIntegration)).
		Methods("DELETE")
	log.Printf("✅ DELETE /slack/integrations/{id} endpoint registered")

	// Discord integrations endpoints
	router.HandleFunc("/discord/integrations", authMiddleware.WithAuth(h.HandleListDiscordIntegrations)).Methods("GET")
	log.Printf("✅ GET /discord/integrations endpoint registered")

	router.HandleFunc("/discord/integrations", authMiddleware.WithAuth(h.HandleCreateDiscordIntegration)).
		Methods("POST")
	log.Printf("✅ POST /discord/integrations endpoint registered")

	router.HandleFunc("/discord/integrations/{id}", authMiddleware.WithAuth(h.HandleDeleteDiscordIntegration)).
		Methods("DELETE")
	log.Printf("✅ DELETE /discord/integrations/{id} endpoint registered")

	// GitHub integrations endpoints
	router.HandleFunc("/github/integrations", authMiddleware.WithAuth(h.HandleListGitHubIntegrations)).Methods("GET")
	log.Printf("✅ GET /github/integrations endpoint registered")

	router.HandleFunc("/github/integrations", authMiddleware.WithAuth(h.HandleCreateGitHubIntegration)).Methods("POST")
	log.Printf("✅ POST /github/integrations endpoint registered")

	router.HandleFunc("/github/integrations/{id}", authMiddleware.WithAuth(h.HandleGetGitHubIntegrationByID)).
		Methods("GET")
	log.Printf("✅ GET /github/integrations/{id} endpoint registered")

	router.HandleFunc("/github/integrations/{id}", authMiddleware.WithAuth(h.HandleDeleteGitHubIntegration)).
		Methods("DELETE")
	log.Printf("✅ DELETE /github/integrations/{id} endpoint registered")

	// Anthropic integrations endpoints
	router.HandleFunc("/anthropic/integrations", authMiddleware.WithAuth(h.HandleListAnthropicIntegrations)).
		Methods("GET")
	log.Printf("✅ GET /anthropic/integrations endpoint registered")
	router.HandleFunc("/anthropic/integrations", authMiddleware.WithAuth(h.HandleCreateAnthropicIntegration)).
		Methods("POST")
	log.Printf("✅ POST /anthropic/integrations endpoint registered")
	router.HandleFunc("/anthropic/integrations/{id}", authMiddleware.WithAuth(h.HandleGetAnthropicIntegrationByID)).
		Methods("GET")
	log.Printf("✅ GET /anthropic/integrations/{id} endpoint registered")
	router.HandleFunc("/anthropic/integrations/{id}", authMiddleware.WithAuth(h.HandleDeleteAnthropicIntegration)).
		Methods("DELETE")
	log.Printf("✅ DELETE /anthropic/integrations/{id} endpoint registered")

	// GitHub repository listing endpoint
	router.HandleFunc("/github/repositories", authMiddleware.WithAuth(h.HandleListGitHubRepositories)).Methods("GET")
	log.Printf("✅ GET /github/repositories endpoint registered")

	// CCAgent Container integrations endpoints
	router.HandleFunc("/ccagent-container/integrations", authMiddleware.WithAuth(h.HandleListCCAgentContainerIntegrations)).
		Methods("GET")
	log.Printf("✅ GET /ccagent-container/integrations endpoint registered")
	router.HandleFunc("/ccagent-container/integrations", authMiddleware.WithAuth(h.HandleCreateCCAgentContainerIntegration)).
		Methods("POST")
	log.Printf("✅ POST /ccagent-container/integrations endpoint registered")
	router.HandleFunc("/ccagent-container/integrations/{id}", authMiddleware.WithAuth(h.HandleGetCCAgentContainerIntegrationByID)).
		Methods("GET")
	log.Printf("✅ GET /ccagent-container/integrations/{id} endpoint registered")
	router.HandleFunc("/ccagent-container/integrations/{id}", authMiddleware.WithAuth(h.HandleDeleteCCAgentContainerIntegration)).
		Methods("DELETE")
	log.Printf("✅ DELETE /ccagent-container/integrations/{id} endpoint registered")
	router.HandleFunc("/ccagents/{id}/redeploy", authMiddleware.WithAuth(h.HandleRedeployCCAgentContainer)).
		Methods("POST")
	log.Printf("✅ POST /ccagents/{id}/redeploy endpoint registered")

	// Organization endpoints
	router.HandleFunc("/organizations", authMiddleware.WithAuth(h.HandleGetOrganization)).Methods("GET")
	log.Printf("✅ GET /organizations endpoint registered")

	router.HandleFunc("/organizations/ccagent_secret_key", authMiddleware.WithAuth(h.HandleGenerateCCAgentSecretKey)).
		Methods("POST")
	log.Printf("✅ POST /organizations/ccagent_secret_key endpoint registered")

	log.Printf("✅ All dashboard API endpoints registered successfully")
}

func (h *DashboardHTTPHandler) writeJSONResponse(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("❌ Failed to encode JSON response: %v", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
	}
}
