.DEFAULT_GOAL := help
GO           ?= go
CGO_ENABLED  ?= 0
GOMAINMODULE ?= $(shell $(GO) list -m)
GOFMT        ?= $(GO)fmt # need this plain fail
GOLINT 		 ?= $(GO)langci-lint run
GOTEST 		 ?= $(GO) test -v ./...
ifneq ($(shell which gotestsum),)
	GOTEST := gotestsum --format pkgname
endif
ifneq ($(shell which gofumpt),)
	GOFMT := gofumpt -d .
endif

# custom
GOBUILD_FLAGS ?= -a
EXECS	      ?= smol

.PHONY: spin # Get the code into shape
spin: unused lint format test

.PHONY: bench # Run all benchmarks
bench:
	go test -bench=. -benchmem -cpu=2,4 -count=2 ./...

.PHONY: test # Run tests
test:
	@echo ">> running tests"
	@$(GOTEST)

.PHONY: test-watch # Run tests in watch mode
test-watch:
	@echo ">> running tests (watch mode)"
	@$(GOTEST) --watch

.PHONY: lint # Run linter
lint:
	@echo ">> running linter"
	@$(GOLINT)

.PHONY: format # Run formatter but don't edit
format:
	@echo ">> running formatter"
	@$(GOFMT)

.PHONY: unused # Run check for unused packages
unused:
	@echo ">> running check for unused/missing packages"
	$(GO) mod tidy
	@git diff --exit-code -- go.sum go.mod

.PHONY: vendor # Run go mod vendor, adds/removes vendor packages
vendor:
	@echo ">> adds/removes vendored packages based on go.mod"
	$(GO) mod vendor

.PHONY: build # build for amd64
build: build_amd64

.PHONY: build_amd64
build_amd64:
	GOARCH=amd64 GOOS=linux $(GO) build $(GOBUILD_FLAGS) -o $(foreach E,$(EXECS), ./cmd/${E}/builds/${E}-amd64-linux $(GOMAINMODULE)/cmd/${E})

# Random Info Dumps
.PHONY: list-all-go-files # Lists go files by package
list-all-go-files:
	@eval "go list -f={{.GoFiles}} ./..."

.PHONY: list-all-test-files # Lists test files by package
list-all-test-files:
	@eval "go list -f={{.TestGoFiles}} ./..."

.PHONY: list-platforms # Lists supported arch-platorm by go
list-platforms:
	go tool dist list


# nix
.PHONY: nix-flake-show # Show flake structure
nix-flake-show:
	nix flake show

.PHONY: nix-run-dev # Run "dev" environment (see flakes.nix)
nix-run-dev:
	nix run .#dev

# database
.PHONY: pgcli # pop pgcli shell
pgcli:
	pgcli -U $(PGUSER) -p $(PGPORT) -h $(PGSOCK)

.PHONY: migration-status # goose migration status
migration-status:
	goose --dir db/migrations status

.PHONY: sqlc-generate # sqlc generate
sqlc-generate:
	sqlc generate -f db/sqlc-files/sqlc.yaml

.PHONY: help # Generate list of targets with descriptions
help:
	@echo "Target descriptions"
	@echo "NOTE: Targets with no description are not listed"
	@echo
	@grep '^.PHONY: .* #' Makefile | sed 's/\.PHONY: \(.*\) # \(.*\)/\1;;;\2/' | column -t -s ";;;"
