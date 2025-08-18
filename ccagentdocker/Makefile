# Docker image configuration
IMAGE_NAME = preslavmihaylov/ccagent-docker
TAG ?= latest

.PHONY: help build push build-deploy clean

help: ## Show this help message
	@echo "Available targets:"
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-20s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

build: ## Build the Docker image
	@echo "Building Docker image $(IMAGE_NAME):$(TAG)..."
	docker build --platform linux/amd64 -t $(IMAGE_NAME):$(TAG) .
	@echo "Build completed successfully!"

push: ## Push the Docker image to registry
	@echo "Pushing Docker image $(IMAGE_NAME):$(TAG)..."
	docker push $(IMAGE_NAME):$(TAG)
	@echo "Push completed successfully!"

build-deploy: build push ## Build and deploy the Docker image
	@echo "Build and deploy completed for $(IMAGE_NAME):$(TAG)"

clean: ## Remove local Docker image
	@echo "Removing local Docker image $(IMAGE_NAME):$(TAG)..."
	docker rmi $(IMAGE_NAME):$(TAG) 2>/dev/null || true
	@echo "Cleanup completed!"

# Development targets
dev-build: ## Build with development tag
	$(MAKE) build TAG=dev

dev-deploy: ## Build and deploy with development tag
	$(MAKE) build-deploy TAG=dev