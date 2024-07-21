#
# SPDX-License-Identifier: Apache-2.0
#

base_dir := $(patsubst %/,%,$(dir $(realpath $(lastword $(MAKEFILE_LIST)))))

go_bin_dir := $(shell go env GOPATH)/bin

.PHONY: test
test: lint unit-test

.PHONY: unit-test
unit-test:
	cd '$(base_dir)' && \
		go test -timeout 30s -race -coverprofile=cover.out ./...

.PHONY: generate
generate:
	go install github.com/maxbrunsfeld/counterfeiter/v6@latest
	cd '$(base_dir)' && \
		go generate ./...

.PHONY: lint
lint: staticcheck golangci-lint

.PHONY: staticcheck
staticcheck:
	go install honnef.co/go/tools/cmd/staticcheck@latest
	cd '$(base_dir)' && \
		staticcheck -f stylish ./...

.PHONY: install-golangci-lint
install-golangci-lint:
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b '$(go_bin_dir)'

$(go_bin_dir)/golangci-lint:
	$(MAKE) install-golangci-lint

.PHONY: golangci-lint
golangci-lint: $(go_bin_dir)/golangci-lint
	cd '$(base_dir)' && \
		golangci-lint run

.PHONY: scan
scan:
	go install golang.org/x/vuln/cmd/govulncheck@latest
	cd '$(base_dir)' && \
		govulncheck ./...
