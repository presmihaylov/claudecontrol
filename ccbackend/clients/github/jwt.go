package github

import (
	"fmt"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type githubJWTClient struct {
	appID      string
	privateKey []byte

	mu        sync.RWMutex
	token     string
	expiresAt time.Time
}

func newGitHubJWTClient(appID string, privateKey []byte) (*githubJWTClient, error) {
	// Validate private key can be parsed
	if _, err := jwt.ParseRSAPrivateKeyFromPEM(privateKey); err != nil {
		return nil, fmt.Errorf("invalid private key: %w", err)
	}

	return &githubJWTClient{
		appID:      appID,
		privateKey: privateKey,
	}, nil
}

func (c *githubJWTClient) getToken() (string, error) {
	c.mu.RLock()
	// Check if cached token is still valid with 10 minute buffer
	if c.token != "" && time.Now().Add(10*time.Minute).Before(c.expiresAt) {
		defer c.mu.RUnlock()
		return c.token, nil
	}
	c.mu.RUnlock()

	// Need to generate new token
	c.mu.Lock()
	defer c.mu.Unlock()

	// Double-check after acquiring write lock
	if c.token != "" && time.Now().Add(10*time.Minute).Before(c.expiresAt) {
		return c.token, nil
	}

	// Generate new JWT
	token, expiresAt, err := c.generateJWT()
	if err != nil {
		return "", err
	}

	// Cache the token
	c.token = token
	c.expiresAt = expiresAt

	return token, nil
}

func (c *githubJWTClient) generateJWT() (string, time.Time, error) {
	now := time.Now()
	expiresAt := now.Add(10 * time.Minute) // GitHub max is 10 minutes

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{
		"iat": jwt.NewNumericDate(now.Add(-60 * time.Second)), // 60 seconds in past for clock drift
		"exp": jwt.NewNumericDate(expiresAt),
		"iss": c.appID,
	})

	privateKey, err := jwt.ParseRSAPrivateKeyFromPEM(c.privateKey)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("failed to parse private key: %w", err)
	}

	tokenString, err := token.SignedString(privateKey)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("failed to sign token: %w", err)
	}

	return tokenString, expiresAt, nil
}
