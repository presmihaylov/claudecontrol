# Docker image configuration
IMAGE_NAME = preslavmihaylov/ccagent-docker
ROTATOR_IMAGE_NAME = preslavmihaylov/ccagent-token-rotator
TAG ?= latest

.PHONY: help build build-rotator build-all push push-rotator push-all build-deploy build-deploy-rotator build-deploy-all clean clean-rotator clean-all

help: ## Show this help message
	@echo "Available targets:"
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-20s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

# Main ccagent image targets
build: ## Build the main ccagent Docker image
	@echo "Building Docker image $(IMAGE_NAME):$(TAG)..."
	docker build --platform linux/amd64 -t $(IMAGE_NAME):$(TAG) .
	@echo "Build completed successfully!"

push: ## Push the main ccagent Docker image to registry
	@echo "Pushing Docker image $(IMAGE_NAME):$(TAG)..."
	docker push $(IMAGE_NAME):$(TAG)
	@echo "Push completed successfully!"

build-deploy: build push ## Build and deploy the main ccagent Docker image
	@echo "Build and deploy completed for $(IMAGE_NAME):$(TAG)"

# Token rotator image targets
build-rotator: ## Build the token rotator Docker image
	@echo "Building Docker image $(ROTATOR_IMAGE_NAME):$(TAG)..."
	docker build --platform linux/amd64 -t $(ROTATOR_IMAGE_NAME):$(TAG) -f Dockerfile.token-rotator .
	@echo "Token rotator build completed successfully!"

push-rotator: ## Push the token rotator Docker image to registry
	@echo "Pushing Docker image $(ROTATOR_IMAGE_NAME):$(TAG)..."
	docker push $(ROTATOR_IMAGE_NAME):$(TAG)
	@echo "Token rotator push completed successfully!"

build-deploy-rotator: build-rotator push-rotator ## Build and deploy the token rotator Docker image
	@echo "Build and deploy completed for $(ROTATOR_IMAGE_NAME):$(TAG)"

# Combined targets for both images
build-all: build build-rotator ## Build both ccagent and token rotator images

push-all: push push-rotator ## Push both ccagent and token rotator images

build-deploy-all: build-all push-all ## Build and deploy both images
	@echo "Build and deploy completed for all images with tag $(TAG)"

# Cleanup targets
clean: ## Remove local ccagent Docker image
	@echo "Removing local Docker image $(IMAGE_NAME):$(TAG)..."
	docker rmi $(IMAGE_NAME):$(TAG) 2>/dev/null || true
	@echo "Cleanup completed!"

clean-rotator: ## Remove local token rotator Docker image
	@echo "Removing local Docker image $(ROTATOR_IMAGE_NAME):$(TAG)..."
	docker rmi $(ROTATOR_IMAGE_NAME):$(TAG) 2>/dev/null || true
	@echo "Token rotator cleanup completed!"

clean-all: clean clean-rotator ## Remove all local Docker images
	@echo "All images cleaned up!"

# Development targets
dev-build: ## Build main image with development tag
	$(MAKE) build TAG=dev

dev-build-rotator: ## Build token rotator with development tag
	$(MAKE) build-rotator TAG=dev

dev-build-all: ## Build all images with development tag
	$(MAKE) build-all TAG=dev

dev-deploy: ## Build and deploy main image with development tag
	$(MAKE) build-deploy TAG=dev

dev-deploy-rotator: ## Build and deploy token rotator with development tag
	$(MAKE) build-deploy-rotator TAG=dev

dev-deploy-all: ## Build and deploy all images with development tag
	$(MAKE) build-deploy-all TAG=dev