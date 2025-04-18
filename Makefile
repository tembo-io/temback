VERSION  := v0.0.1-dev
REVISION := $(shell git rev-parse --short HEAD)
REGISTRY ?= localhost:5001
ldflags = -ldflags="-s -w -X 'main.version=$(VERSION)' -X 'main.build=$(REVISION)'"

## temback: Build temback for the local platform.
temback: main.go go.*
	go build $(ldflags) -o $@ .

## temback-linux-amd64: Build temback linux/amd64.
temback-linux-amd64: main.go go.*
	GOOS=linux GOARCH=amd64 go build $(ldflags) -o $@ .

.PHONY: image # Build the linux/amd64 OCI image.
image: temback-linux-amd64
	registry=$(REGISTRY) version=$(VERSION) revision=$(REVISION) docker buildx bake $(if $(filter true,$(PUSH)),--push,)

.PHONY: clean # Remove generated files
clean:
	go clean
	$(RM) -rf temback*

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
