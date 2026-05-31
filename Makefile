# shutup — dev tasks. The Go module lives in cli/.
# Monorepo-ready: backend/dashboard targets can be added alongside these later.

CLI := cli
SHUTUP_BIN ?= $(HOME)/.local/bin/shutup

.PHONY: build install test cover vet fmt clean help

help: ## show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN{FS=":.*?## "}{printf "  \033[36m%-10s\033[0m %s\n", $$1, $$2}'

build: ## build the CLI to ./cli/shutup
	cd $(CLI) && go build -o shutup .

install: ## build + install to $(SHUTUP_BIN)
	cd $(CLI) && go build -o "$(SHUTUP_BIN)" .
	@echo "installed shutup -> $(SHUTUP_BIN)"

test: ## run all tests
	cd $(CLI) && go test ./...

cover: ## run tests with a coverage summary
	cd $(CLI) && go test -cover ./...

vet: ## run go vet
	cd $(CLI) && go vet ./...

fmt: ## format the code
	cd $(CLI) && gofmt -w .

clean: ## remove the locally-built binary
	rm -f $(CLI)/shutup
