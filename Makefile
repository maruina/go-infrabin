
.PHONY: lint
lint: ## Lint code and proto
		buf lint

.PHONY: generate
generate: ## Generate code and proto
		rm -rf gen
		buf generate
