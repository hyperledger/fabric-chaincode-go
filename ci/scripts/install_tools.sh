# Copyright the Hyperledger Fabric contributors. All rights reserved.
# SPDX-License-Identifier: Apache-2.0

set -euo pipefail

export GOBIN=/go/bin

pushd ci/tools
go install golang.org/x/lint/golint
go install golang.org/x/tools/cmd/goimports
popd
