VERSION  := v0.0.1-dev
ldflags = -ldflags="-s -w -X 'main.version=$(VERSION)' -X 'main.build=$(shell git rev-parse --short HEAD)'"

temback: main.go go.*
	go build $(ldflags) -o $@ .

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
	sudo curl -Lo /bin/hadolint https://github.com/hadolint/hadolint/releases/download/v2.12.0/hadolint-Linux-x86_64
	sudo chmod +x /bin/hadolint
