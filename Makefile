##@ General

.PHONY: help
help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Checks

.PHONY: checks
checks: go-checks ## Runs all checks.

.PHONY: go-checks
go-checks: ## Runs go checks.
	@make -f "$$PRIMUS_HOME/src/make/main.mk" go-fmt
	@make -f "$$PRIMUS_HOME/src/make/main.mk" go-deps
	@make -f "$$PRIMUS_HOME/src/make/main.mk" go-lint
	@make -f "$$PRIMUS_HOME/src/make/main.mk" go-test
