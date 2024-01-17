GOLANGCI_VERSION = v1.55.2

help: ## show help, shown by default if no target is specified
	@grep -E '^[0-9a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

lint: ## run code linters
	CGO_ENABLED=0 golangci-lint run

build: ## build code
	go build ./...

test: ## run tests
	go test -race ./...

test-coverage: ## run unit tests and create test coverage
	CGO_ENABLED=0 go test ./... -coverprofile .testCoverage -covermode=atomic -coverpkg=./...
	go tool cover -func .testCoverage | grep total | awk '{print "Total coverage: "$$3}'

test-coverage-web: test-coverage ## run unit tests and show test coverage in browser
	go tool cover -html=.testCoverage

install-linters: ## install the linter
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@${GOLANGCI_VERSION}

release: ## build release binaries for current git tag and publish on github
	goreleaser release

release-snapshot: ## build release binaries from current git state as snapshot
	goreleaser release --snapshot --clean
