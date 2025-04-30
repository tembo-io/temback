GO       ?= go
GOOS     ?= $(word 1,$(subst /, ,$(word 4, $(shell $(GO) version))))
GOARCH   ?= $(word 2,$(subst /, ,$(word 4, $(shell $(GO) version))))
PLATFORM := $(GOOS)-$(GOARCH)
VERSION  := v0.2.0
REVISION := $(shell git rev-parse --short HEAD)
REGISTRY ?= localhost:5001
ldflags = -ldflags="-s -w -X 'main.version=$(VERSION)' -X 'main.build=$(REVISION)'"

############################################################################
# Artifacts
.PHONY: temback # Build temback.
temback: _build/$(PLATFORM)/temback

_build/%/temback: main.go go.* template.md
	GOOS=$(word 1,$(subst -, ,$*)) GOARCH=$(word 2,$(subst -, ,$*)) $(GO) build $(ldflags) -o $@ ./$<

run: _build/$(PLATFORM)/temback
	@./_build/$(PLATFORM)/temback --version

show-build: _build/$(PLATFORM)/temback
	@echo ./_build/$(PLATFORM)/temback

.PHONY: version-env # Echo setting an environment variable with the release version.
version-env:
	@echo VERSION=$(VERSION)

############################################################################
# Release artifacts.
.PHONY: release # Build a release zip file or .tar.gz & tar.bz2 files.
ifeq ($(GOOS),windows)
release: _build/artifacts/temback-$(VERSION)-windows-$(GOARCH).zip
else
release: _build/artifacts/temback-$(VERSION)-$(PLATFORM).tar.gz
endif

# Build a release zip file for Windows.
_build/artifacts/temback-$(VERSION)-windows-$(GOARCH).zip: README.md LICENSE.md CHANGELOG.md _build/windows-$(GOARCH)/temback
	@mkdir -p "_build/artifacts/temback-$(VERSION)-windows-$(GOARCH)"
	cp $^ "_build/artifacts/temback-$(VERSION)-windows-$(GOARCH)"
	cd _build/artifacts && 7z a "temback-$(VERSION)-windows-$(GOARCH).zip" "temback-$(VERSION)-windows-$(GOARCH)"
	rm -R "_build/artifacts/temback-$(VERSION)-windows-$(GOARCH)"

# Build a .tar.gz file for the specified platform.
_build/artifacts/temback-$(VERSION)-$(PLATFORM).tar.gz: README.md LICENSE.md CHANGELOG.md _build/$(PLATFORM)/temback
	@mkdir -p "_build/artifacts/temback-$(VERSION)-$(PLATFORM)"
	cp $^ "_build/artifacts/temback-$(VERSION)-$(PLATFORM)"
	cd _build/artifacts && tar zcvf "temback-$(VERSION)-$(PLATFORM).tar.gz" "temback-$(VERSION)-$(PLATFORM)"
	rm -R "_build/artifacts/temback-$(VERSION)-$(PLATFORM)"

############################################################################
# OCI images.
.PHONY: image # Build the linux/amd64 OCI image.
image: _build/linux-amd64/temback _build/linux-arm64/temback
	registry=$(REGISTRY) version=$(VERSION) revision=$(REVISION) docker buildx bake $(if $(filter true,$(PUSH)),--push,)

.PHONY: clean # Remove generated files
clean:
	$(GO) clean
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
