# Build and Release Automation for Vigil

# Architectures to build
PLATFORMS=linux/amd64,linux/arm64
IMAGE=leshicodes/vigil

# Get current branch and short SHA
BRANCH=$(shell git rev-parse --abbrev-ref HEAD)
SHA=$(shell git rev-parse --short HEAD)

# Determine tag based on branch
ifeq ($(BRANCH),main)
	TAG=latest
else ifeq ($(BRANCH),develop)
	TAG=edge
else
	TAG=$(BRANCH)
endif

.PHONY: all help build-multi release

all: help

help:
	@echo "Vigil Build System"
	@echo ""
	@echo "Commands:"
	@echo "  make build-multi    Build for amd64 and arm64 (local only)"
	@echo "  make release        Build, tag, and push to registry"
	@echo ""
	@echo "Current Context:"
	@echo "  Branch: $(BRANCH)"
	@echo "  SHA:    $(SHA)"
	@echo "  Tag:    $(TAG)"

build-multi:
	docker buildx build --platform $(PLATFORMS) -t $(IMAGE):$(TAG) .

release:
	docker buildx build --platform $(PLATFORMS) \
		-t $(IMAGE):$(TAG) \
		-t $(IMAGE):$(SHA) \
		--push .
