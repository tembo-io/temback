VERSION  := v0.0.1-dev
REVISION := $(shell git rev-parse --short HEAD)
REGISTRY ?= localhost:5001
ldflags = -ldflags="-s -w -X 'main.version=$(VERSION)' -X 'main.build=$(REVISION)'"

temback: main.go go.*
	go build $(ldflags) -o $@ .

image: temback
	registry=$(REGISTRY) version=$(VERSION) revision=$(REVISION) docker buildx bake $(if $(filter true,$(PUSH)),--push,)

.PHONY: clean # Remove generated files
clean:
	$(GO) clean
	$(RM) -rf _build vendor

.PHONY: lint # Lint the project
lint: .pre-commit-config.yaml .golangci.yaml
	@pre-commit run --show-diff-on-failure --color=always --all-files

## .git/hooks/pre-commit: Install the pre-commit hook
.git/hooks/pre-commit:
	@printf "#!/bin/sh\nmake lint\n" > $@
	@chmod +x $@

.PHONY: debian-lint-depends # Install linting tools on Debian
debian-lint-depends:
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sudo sh -s -- -b /usr/bin v2.1.2
