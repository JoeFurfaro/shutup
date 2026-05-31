# shutup CLI — dev tasks. Run from this directory (cli/).

SHUTUP_BIN ?= $(HOME)/.local/bin/shutup

.PHONY: build install test cover vet fmt clean help

help: ## show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN{FS=":.*?## "}{printf "  \033[36m%-10s\033[0m %s\n", $$1, $$2}'

build: ## build the CLI to ./shutup
	go build -o shutup .

install: ## build + install to $(SHUTUP_BIN)
	go build -o "$(SHUTUP_BIN)" .
	@echo "installed shutup -> $(SHUTUP_BIN)"

test: ## run all tests
	go test ./...

cover: ## run tests with a coverage summary
	go test -cover ./...

vet: ## run go vet
	go vet ./...

fmt: ## format the code
	gofmt -w .

clean: ## remove the locally-built binary
	rm -f shutup
