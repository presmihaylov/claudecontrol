package usecases

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"ccagent/core/log"
)

type HealthcheckUseCase struct {
	healthcheckURL string
	httpClient     *http.Client
}

func NewHealthcheckUseCase(baseURL string) *HealthcheckUseCase {
	return &HealthcheckUseCase{
		healthcheckURL: fmt.Sprintf("%s/health", baseURL),
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

func (h *HealthcheckUseCase) PerformHealthcheck(ctx context.Context) error {
	log.Info("📋 Starting to perform healthcheck to %s", h.healthcheckURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, h.healthcheckURL, nil)
	if err != nil {
		log.Info("❌ Failed to create healthcheck request: %v", err)
		return fmt.Errorf("failed to create healthcheck request: %w", err)
	}

	resp, err := h.httpClient.Do(req)
	if err != nil {
		log.Info("❌ Healthcheck request failed: %v", err)
		return fmt.Errorf("healthcheck request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Info("❌ Healthcheck returned non-OK status: %d", resp.StatusCode)
		return fmt.Errorf("healthcheck returned status %d", resp.StatusCode)
	}

	log.Info("📋 Completed successfully - healthcheck passed")
	return nil
}

func (h *HealthcheckUseCase) StartPeriodicHealthcheck(ctx context.Context, interval time.Duration, onFailure func()) {
	log.Info("📋 Starting periodic healthcheck with interval %v", interval)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Info("📋 Stopping periodic healthcheck due to context cancellation")
			return
		case <-ticker.C:
			if err := h.PerformHealthcheck(ctx); err != nil {
				log.Info("⚠️ Periodic healthcheck failed: %v", err)
				onFailure()
			}
		}
	}
}