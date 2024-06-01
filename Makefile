#
# SPDX-License-Identifier: Apache-2.0
#

base_dir := $(patsubst %/,%,$(dir $(realpath $(lastword $(MAKEFILE_LIST)))))

v2_dir := $(base_dir)/v2

go_bin_dir := $(shell go env GOPATH)/bin

.PHONY: unit-test
unit-test:
	cd '$(v2_dir)' && \
		go test -timeout 10s -race -coverprofile=cover.out ./...

.PHONY: generate
generate:
	go install github.com/maxbrunsfeld/counterfeiter/v6@latest
	cd '$(v2_dir)' && \
		go generate ./...

.PHONY: lint
lint: staticcheck golangci-lint

.PHONY: staticcheck
staticcheck:
	go install honnef.co/go/tools/cmd/staticcheck@latest
	cd '$(v2_dir)' && \
		staticcheck -f stylish ./...

.PHONY: install-golangci-lint
install-golangci-lint:
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b '$(go_bin_dir)'

$(go_bin_dir)/golangci-lint:
	$(MAKE) install-golangci-lint

.PHONY: golangci-lint
golangci-lint: $(go_bin_dir)/golangci-lint
	cd '$(v2_dir)' && \
		golangci-lint run

.PHONY: scan-v2
scan-v2:
	go install golang.org/x/vuln/cmd/govulncheck@latest
	cd '$(v2_dir)' && \
		govulncheck ./...

.PHONY: scan-v1
scan-v1:
	go install golang.org/x/vuln/cmd/govulncheck@latest
	cd '$(base_dir)' && \
		govulncheck ./...
