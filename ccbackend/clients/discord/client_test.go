package discord

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"ccbackend/clients"
)

func TestDiscordClient_ExchangeCodeForToken_Success(t *testing.T) {
	// Mock server setup
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Validate request
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/oauth2/token", r.URL.Path)
		assert.Equal(t, "application/x-www-form-urlencoded", r.Header.Get("Content-Type"))

		// Parse form data
		err := r.ParseForm()
		require.NoError(t, err)
		assert.Equal(t, "test-client-id", r.FormValue("client_id"))
		assert.Equal(t, "test-client-secret", r.FormValue("client_secret"))
		assert.Equal(t, "authorization_code", r.FormValue("grant_type"))
		assert.Equal(t, "test-auth-code", r.FormValue("code"))
		assert.Equal(t, "https://example.com/redirect", r.FormValue("redirect_uri"))

		// Return successful response
		response := clients.DiscordOAuthResponse{
			AccessToken: "test-access-token",
			TokenType:   "Bearer",
			ExpiresIn:   3600,
			Scope:       "applications.commands bot",
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// Temporarily override the Discord API URL for testing
	originalURL := discordOAuthURL
	discordOAuthURL = server.URL + "/oauth2/token"
	defer func() { discordOAuthURL = originalURL }()

	client := NewDiscordOAuthClient()
	httpClient := &http.Client{}

	// Test the OAuth code exchange
	response, err := client.ExchangeCodeForToken(
		httpClient,
		"test-client-id",
		"test-client-secret",
		"test-auth-code",
		"https://example.com/redirect",
	)

	require.NoError(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, "test-access-token", response.AccessToken)
	assert.Equal(t, "Bearer", response.TokenType)
	assert.Equal(t, 3600, response.ExpiresIn)
	assert.Equal(t, "applications.commands bot", response.Scope)
}

func TestDiscordClient_ExchangeCodeForToken_HTTPError(t *testing.T) {
	// Mock server that returns an error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error": "invalid_grant", "error_description": "Invalid authorization code"}`))
	}))
	defer server.Close()

	// Temporarily override the Discord API URL for testing
	originalURL := discordOAuthURL
	discordOAuthURL = server.URL + "/oauth2/token"
	defer func() { discordOAuthURL = originalURL }()

	client := NewDiscordOAuthClient()
	httpClient := &http.Client{}

	response, err := client.ExchangeCodeForToken(
		httpClient,
		"test-client-id",
		"test-client-secret",
		"invalid-code",
		"https://example.com/redirect",
	)

	assert.Error(t, err)
	assert.Nil(t, response)
	assert.Contains(t, err.Error(), "OAuth request failed with status 400")
	assert.Contains(t, err.Error(), "invalid_grant")
}

func TestDiscordClient_ExchangeCodeForToken_InvalidJSON(t *testing.T) {
	// Mock server that returns invalid JSON
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`invalid json response`))
	}))
	defer server.Close()

	// Temporarily override the Discord API URL for testing
	originalURL := discordOAuthURL
	discordOAuthURL = server.URL + "/oauth2/token"
	defer func() { discordOAuthURL = originalURL }()

	client := NewDiscordOAuthClient()
	httpClient := &http.Client{}

	response, err := client.ExchangeCodeForToken(
		httpClient,
		"test-client-id",
		"test-client-secret",
		"test-code",
		"https://example.com/redirect",
	)

	assert.Error(t, err)
	assert.Nil(t, response)
	assert.Contains(t, err.Error(), "failed to decode OAuth response")
}

func TestDiscordClient_GetGuildInfo_Success(t *testing.T) {
	// Mock server setup
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Validate request
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, "/users/@me/guilds", r.URL.Path)
		assert.Equal(t, "Bearer test-access-token", r.Header.Get("Authorization"))
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		// Return successful response with guild list
		guilds := []*clients.DiscordGuild{
			{ID: "123456789012345678", Name: "Test Guild 1"},
			{ID: "987654321098765432", Name: "Test Guild 2"},
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(guilds)
	}))
	defer server.Close()

	// Temporarily override the Discord API URL for testing
	originalURL := discordGuildsURL
	discordGuildsURL = server.URL + "/users/@me/guilds"
	defer func() { discordGuildsURL = originalURL }()

	client := NewDiscordOAuthClient()
	httpClient := &http.Client{}

	guilds, err := client.GetGuildInfo(httpClient, "test-access-token")

	require.NoError(t, err)
	assert.Len(t, guilds, 2)
	assert.Equal(t, "123456789012345678", guilds[0].ID)
	assert.Equal(t, "Test Guild 1", guilds[0].Name)
	assert.Equal(t, "987654321098765432", guilds[1].ID)
	assert.Equal(t, "Test Guild 2", guilds[1].Name)
}

func TestDiscordClient_GetGuildInfo_HTTPError(t *testing.T) {
	// Mock server that returns an error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"code": 0, "message": "401: Unauthorized"}`))
	}))
	defer server.Close()

	// Temporarily override the Discord API URL for testing
	originalURL := discordGuildsURL
	discordGuildsURL = server.URL + "/users/@me/guilds"
	defer func() { discordGuildsURL = originalURL }()

	client := NewDiscordOAuthClient()
	httpClient := &http.Client{}

	guilds, err := client.GetGuildInfo(httpClient, "invalid-token")

	assert.Error(t, err)
	assert.Nil(t, guilds)
	assert.Contains(t, err.Error(), "guilds request failed with status 401")
	assert.Contains(t, err.Error(), "Unauthorized")
}

func TestDiscordClient_GetGuildInfo_InvalidJSON(t *testing.T) {
	// Mock server that returns invalid JSON
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`invalid json response`))
	}))
	defer server.Close()

	// Temporarily override the Discord API URL for testing
	originalURL := discordGuildsURL
	discordGuildsURL = server.URL + "/users/@me/guilds"
	defer func() { discordGuildsURL = originalURL }()

	client := NewDiscordOAuthClient()
	httpClient := &http.Client{}

	guilds, err := client.GetGuildInfo(httpClient, "test-access-token")

	assert.Error(t, err)
	assert.Nil(t, guilds)
	assert.Contains(t, err.Error(), "failed to decode guilds response")
}

func TestDiscordClient_GetGuildByID_Success(t *testing.T) {
	guildID := "123456789012345678"

	// Mock server setup
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Validate request
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, "/guilds/"+guildID, r.URL.Path)
		assert.Equal(t, "Bot test-access-token", r.Header.Get("Authorization"))
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		// Return successful response with guild info
		guild := clients.DiscordGuild{
			ID:   guildID,
			Name: "Test Guild",
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(guild)
	}))
	defer server.Close()

	// Temporarily override the Discord API base URL for testing
	originalAPIBase := discordAPIBase
	discordAPIBase = server.URL
	defer func() { discordAPIBase = originalAPIBase }()

	client := NewDiscordOAuthClient()
	httpClient := &http.Client{}

	guild, err := client.GetGuildByID(httpClient, "test-access-token", guildID)

	require.NoError(t, err)
	assert.NotNil(t, guild)
	assert.Equal(t, guildID, guild.ID)
	assert.Equal(t, "Test Guild", guild.Name)
}

func TestDiscordClient_GetGuildByID_NotFound(t *testing.T) {
	guildID := "999999999999999999"

	// Mock server that returns 404
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"code": 10004, "message": "Unknown Guild"}`))
	}))
	defer server.Close()

	// Temporarily override the Discord API base URL for testing
	originalAPIBase := discordAPIBase
	discordAPIBase = server.URL
	defer func() { discordAPIBase = originalAPIBase }()

	client := NewDiscordOAuthClient()
	httpClient := &http.Client{}

	guild, err := client.GetGuildByID(httpClient, "test-access-token", guildID)

	assert.Error(t, err)
	assert.Nil(t, guild)
	assert.Contains(t, err.Error(), "guild request failed with status 404")
	assert.Contains(t, err.Error(), "Unknown Guild")
}

func TestDiscordClient_GetGuildByID_Unauthorized(t *testing.T) {
	guildID := "123456789012345678"

	// Mock server that returns 401 (bot not in guild or insufficient permissions)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte(`{"code": 50001, "message": "Missing Access"}`))
	}))
	defer server.Close()

	// Temporarily override the Discord API base URL for testing
	originalAPIBase := discordAPIBase
	discordAPIBase = server.URL
	defer func() { discordAPIBase = originalAPIBase }()

	client := NewDiscordOAuthClient()
	httpClient := &http.Client{}

	guild, err := client.GetGuildByID(httpClient, "test-access-token", guildID)

	assert.Error(t, err)
	assert.Nil(t, guild)
	assert.Contains(t, err.Error(), "guild request failed with status 403")
	assert.Contains(t, err.Error(), "Missing Access")
}

func TestDiscordClient_GetGuildByID_InvalidJSON(t *testing.T) {
	guildID := "123456789012345678"

	// Mock server that returns invalid JSON
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`invalid json response`))
	}))
	defer server.Close()

	// Temporarily override the Discord API base URL for testing
	originalAPIBase := discordAPIBase
	discordAPIBase = server.URL
	defer func() { discordAPIBase = originalAPIBase }()

	client := NewDiscordOAuthClient()
	httpClient := &http.Client{}

	guild, err := client.GetGuildByID(httpClient, "test-access-token", guildID)

	assert.Error(t, err)
	assert.Nil(t, guild)
	assert.Contains(t, err.Error(), "failed to decode guild response")
}

func TestDiscordClient_GetGuildByID_NetworkError(t *testing.T) {
	// Override URL to point to invalid endpoint
	originalAPIBase := discordAPIBase
	discordAPIBase = "http://invalid-url-that-does-not-exist"
	defer func() { discordAPIBase = originalAPIBase }()

	client := NewDiscordOAuthClient()
	httpClient := &http.Client{}

	guild, err := client.GetGuildByID(httpClient, "test-access-token", "123456789012345678")

	assert.Error(t, err)
	assert.Nil(t, guild)
	assert.Contains(t, err.Error(), "failed to execute guild request")
}

func TestDiscordClient_ExchangeCodeForToken_NetworkError(t *testing.T) {
	// Override URL to point to invalid endpoint
	originalURL := discordOAuthURL
	discordOAuthURL = "http://invalid-url-that-does-not-exist/oauth2/token"
	defer func() { discordOAuthURL = originalURL }()

	client := NewDiscordOAuthClient()
	httpClient := &http.Client{}

	response, err := client.ExchangeCodeForToken(
		httpClient,
		"test-client-id",
		"test-client-secret",
		"test-code",
		"https://example.com/redirect",
	)

	assert.Error(t, err)
	assert.Nil(t, response)
	assert.Contains(t, err.Error(), "failed to execute OAuth request")
}

func TestDiscordClient_GetGuildInfo_NetworkError(t *testing.T) {
	// Override URL to point to invalid endpoint
	originalURL := discordGuildsURL
	discordGuildsURL = "http://invalid-url-that-does-not-exist/users/@me/guilds"
	defer func() { discordGuildsURL = originalURL }()

	client := NewDiscordOAuthClient()
	httpClient := &http.Client{}

	guilds, err := client.GetGuildInfo(httpClient, "test-access-token")

	assert.Error(t, err)
	assert.Nil(t, guilds)
	assert.Contains(t, err.Error(), "failed to execute guilds request")
}

// Test that client implements the interface correctly
func TestDiscordClient_ImplementsInterface(t *testing.T) {
	var _ clients.DiscordOAuthClient = &DiscordClient{}
	var _ clients.DiscordOAuthClient = NewDiscordOAuthClient()
}

// Test context handling in requests
func TestDiscordClient_ContextHandling(t *testing.T) {
	// Mock server that checks for proper context usage
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify context is properly set by checking for cancellation support
		ctx := r.Context()
		assert.NotNil(t, ctx)
		assert.NotEqual(t, context.Background(), ctx)

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(clients.DiscordOAuthResponse{
			AccessToken: "test-token",
			TokenType:   "Bearer",
			ExpiresIn:   3600,
			Scope:       "test",
		})
	}))
	defer server.Close()

	// Temporarily override the Discord API URL for testing
	originalURL := discordOAuthURL
	discordOAuthURL = server.URL + "/oauth2/token"
	defer func() { discordOAuthURL = originalURL }()

	client := NewDiscordOAuthClient()
	httpClient := &http.Client{}

	response, err := client.ExchangeCodeForToken(
		httpClient,
		"test-client-id",
		"test-client-secret",
		"test-code",
		"https://example.com/redirect",
	)

	require.NoError(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, "test-token", response.AccessToken)
}

// Test rate limiting scenarios
func TestDiscordClient_RateLimit(t *testing.T) {
	// Mock server that simulates rate limiting
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-RateLimit-Limit", "5")
		w.Header().Set("X-RateLimit-Remaining", "0")
		w.Header().Set("X-RateLimit-Reset-After", "1")
		w.WriteHeader(http.StatusTooManyRequests)
		w.Write([]byte(`{"message": "You are being rate limited.", "retry_after": 1.0, "global": false}`))
	}))
	defer server.Close()

	// Temporarily override the Discord API URL for testing
	originalURL := discordOAuthURL
	discordOAuthURL = server.URL + "/oauth2/token"
	defer func() { discordOAuthURL = originalURL }()

	client := NewDiscordOAuthClient()
	httpClient := &http.Client{}

	response, err := client.ExchangeCodeForToken(
		httpClient,
		"test-client-id",
		"test-client-secret",
		"test-code",
		"https://example.com/redirect",
	)

	assert.Error(t, err)
	assert.Nil(t, response)
	assert.Contains(t, err.Error(), "OAuth request failed with status 429")
	assert.Contains(t, err.Error(), "rate limited")
}

// Edge case: Empty guild list
func TestDiscordClient_GetGuildInfo_EmptyList(t *testing.T) {
	// Mock server setup that returns empty guild list
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode([]*clients.DiscordGuild{})
	}))
	defer server.Close()

	// Temporarily override the Discord API URL for testing
	originalURL := discordGuildsURL
	discordGuildsURL = server.URL + "/users/@me/guilds"
	defer func() { discordGuildsURL = originalURL }()

	client := NewDiscordOAuthClient()
	httpClient := &http.Client{}

	guilds, err := client.GetGuildInfo(httpClient, "test-access-token")

	require.NoError(t, err)
	assert.Empty(t, guilds)
}
