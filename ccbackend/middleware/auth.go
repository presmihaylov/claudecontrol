package middleware

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/clerk/clerk-sdk-go/v2"
	"github.com/clerk/clerk-sdk-go/v2/jwks"
	"github.com/clerk/clerk-sdk-go/v2/jwt"

	"ccbackend/appctx"
	"ccbackend/core"
	"ccbackend/models"
	"ccbackend/services"
)

// ClerkAuthMiddleware handles JWT authentication using Clerk SDK
type ClerkAuthMiddleware struct {
	usersService         services.UsersService
	organizationsService services.OrganizationsService
	clerkJWKS            *jwks.Client
}

// NewClerkAuthMiddleware creates a new authentication middleware instance
func NewClerkAuthMiddleware(
	usersService services.UsersService,
	organizationsService services.OrganizationsService,
	clerkSecretKey string,
) *ClerkAuthMiddleware {
	config := &clerk.ClientConfig{
		BackendConfig: clerk.BackendConfig{
			Key: clerk.String(clerkSecretKey),
		},
	}
	jwksClient := jwks.NewClient(config)

	return &ClerkAuthMiddleware{
		usersService:         usersService,
		organizationsService: organizationsService,
		clerkJWKS:            jwksClient,
	}
}

// WithAuth wraps an HTTP handler with JWT authentication
func (m *ClerkAuthMiddleware) WithAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Printf("🔐 Authentication middleware processing request from %s", r.RemoteAddr)

		// Check if we're in testing environment
		if os.Getenv("ENVIRONMENT") == "test" {
			log.Printf("🧪 Test environment detected - skipping Clerk validation")
			testOrg := &models.Organization{
				ID:        core.NewID("org"),
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			}
			testUser := &models.User{
				ID:             core.NewID("u"),
				AuthProvider:   "test",
				AuthProviderID: core.NewID("testuser"),
				OrganizationID: models.OrganizationID(testOrg.ID),
				CreatedAt:      time.Now(),
				UpdatedAt:      time.Now(),
			}

			log.Printf("✅ Test user created: %s", testUser.ID)
			ctx := appctx.SetUser(r.Context(), testUser)
			ctx = appctx.SetOrganization(ctx, testOrg)
			r = r.WithContext(ctx)

			next(w, r)
			return
		}

		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			log.Printf("❌ Missing Authorization header")
			m.writeErrorResponse(w, "missing authorization header", http.StatusUnauthorized)
			return
		}
		if !strings.HasPrefix(authHeader, "Bearer ") {
			log.Printf("❌ Invalid Authorization header format")
			m.writeErrorResponse(w, "invalid authorization header format", http.StatusUnauthorized)
			return
		}
		token := strings.TrimPrefix(authHeader, "Bearer ")
		if token == "" {
			log.Printf("❌ Empty bearer token")
			m.writeErrorResponse(w, "empty bearer token", http.StatusUnauthorized)
			return
		}

		// Verify JWT token using Clerk SDK
		claims, err := jwt.Verify(r.Context(), &jwt.VerifyParams{
			Token:      token,
			JWKSClient: m.clerkJWKS,
		})
		if err != nil {
			log.Printf("❌ JWT verification failed: %v", err)
			m.writeErrorResponse(w, "invalid token", http.StatusUnauthorized)
			return
		}

		log.Printf("✅ JWT token verified successfully for user: %s", claims.Subject)
		user, err := m.usersService.GetOrCreateUser(r.Context(), "clerk", claims.Subject)
		if err != nil {
			log.Printf("❌ Failed to get or create user: %v", err)
			m.writeErrorResponse(w, "internal server error", http.StatusInternalServerError)
			return
		}

		// Get the user's organization
		maybeOrg, err := m.organizationsService.GetOrganizationByID(r.Context(), string(user.OrganizationID))
		if err != nil {
			log.Printf("❌ Failed to get organization: %v", err)
			m.writeErrorResponse(w, "internal server error", http.StatusInternalServerError)
			return
		}
		if !maybeOrg.IsPresent() {
			log.Printf("❌ Organization not found: %s", user.OrganizationID)
			m.writeErrorResponse(w, "organization not found", http.StatusInternalServerError)
			return
		}
		org := maybeOrg.MustGet()

		log.Printf("✅ User authenticated successfully: %s (org: %s)", user.ID, org.ID)
		ctx := appctx.SetUser(r.Context(), user)
		ctx = appctx.SetOrganization(ctx, org)
		r = r.WithContext(ctx)

		next(w, r)
	}
}

// writeErrorResponse writes a standardized error response
func (m *ClerkAuthMiddleware) writeErrorResponse(w http.ResponseWriter, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	errorResponse := map[string]string{"error": message}
	if err := json.NewEncoder(w).Encode(errorResponse); err != nil {
		log.Printf("❌ Failed to encode error response: %v", err)
	}
}
