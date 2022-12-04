
.PHONY: lint
lint: ## Lint code and proto
		buf lint

.PHONY: generate
generate: ## Generate code and proto
		rm -rf gen
		buf generate

.PHONY: test
test: ## Run tests
	go test -v ./...
