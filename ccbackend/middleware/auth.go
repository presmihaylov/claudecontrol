package middleware

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"ccbackend/appctx"
	"ccbackend/services"

	"github.com/clerk/clerk-sdk-go/v2"
	"github.com/clerk/clerk-sdk-go/v2/jwt"
	"github.com/clerk/clerk-sdk-go/v2/jwks"
)

// ClerkAuthMiddleware handles JWT authentication using Clerk SDK
type ClerkAuthMiddleware struct {
	usersService *services.UsersService
	clerkJWKS    *jwks.Client
}

// NewClerkAuthMiddleware creates a new authentication middleware instance
func NewClerkAuthMiddleware(usersService *services.UsersService, clerkSecretKey string) *ClerkAuthMiddleware {
	// Create JWKS client for JWT verification
	config := &clerk.ClientConfig{
		BackendConfig: clerk.BackendConfig{
			Key: clerk.String(clerkSecretKey),
		},
	}
	jwksClient := jwks.NewClient(config)

	return &ClerkAuthMiddleware{
		usersService: usersService,
		clerkJWKS:    jwksClient,
	}
}

// WithAuth wraps an HTTP handler with JWT authentication
func (m *ClerkAuthMiddleware) WithAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Printf("🔐 Authentication middleware processing request from %s", r.RemoteAddr)

		// Extract bearer token from Authorization header
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

		// Get or create user in database
		user, err := m.usersService.GetOrCreateUser("clerk", claims.Subject)
		if err != nil {
			log.Printf("❌ Failed to get or create user: %v", err)
			m.writeErrorResponse(w, "internal server error", http.StatusInternalServerError)
			return
		}

		log.Printf("✅ User authenticated successfully: %s", user.ID)

		// Add full user entity to request context
		ctx := appctx.SetUser(r.Context(), user)
		r = r.WithContext(ctx)

		// Call the next handler
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