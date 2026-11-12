# Strand — common local / Docker targets
#   make help

APP        := strand
VERSION    ?= 0.1.0
GO         ?= go
PNPM       ?= pnpm
WEB_DIR    := web
BIN        := ./$(APP)

.PHONY: help
help: ## Show targets
	@grep -E '^[a-zA-Z_-]+:.*?## ' $(MAKEFILE_LIST) | awk -F':.*?## ' '{printf "  %-18s %s\n", $$1, $$2}'

.PHONY: deps
deps: ## Download Go modules + install web deps
	$(GO) mod download
	cd $(WEB_DIR) && $(PNPM) install

.PHONY: config
config: ## Initialize ~/.strand config + .env
	$(GO) run ./cmd/strand config init

.PHONY: build
build: ## Build API binary to ./strand
	CGO_ENABLED=1 $(GO) build -ldflags "-s -w -X main.version=$(VERSION)" -o $(BIN) ./cmd/strand

.PHONY: web-build
web-build: ## Build web console to web/dist
	cd $(WEB_DIR) && $(PNPM) run build

.PHONY: test
test: ## Run Go + web unit tests
	$(GO) test ./internal/market/ ./internal/... -count=1
	cd $(WEB_DIR) && $(PNPM) test

.PHONY: lint-web
lint-web: ## Lint web
	cd $(WEB_DIR) && $(PNPM) run lint

.PHONY: api
api: ## Start API (foreground)
	$(GO) run ./cmd/strand start

.PHONY: web
web: ## Start Vite dev server (proxies /api → :8080)
	cd $(WEB_DIR) && $(PNPM) run dev

.PHONY: docker-up
docker-up: ## Build & start API + web via compose
	docker compose up --build -d

.PHONY: docker-down
docker-down: ## Stop compose stack
	docker compose down

.PHONY: docker-logs
docker-logs: ## Tail compose logs
	docker compose logs -f --tail=200

.PHONY: clean
clean: ## Remove local binary and web dist
	rm -f $(BIN) $(BIN)-server
	rm -rf $(WEB_DIR)/dist
