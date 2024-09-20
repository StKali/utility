#!/bin/bash

set -e

# Add go bin to PATH and install dependencies ..."
export PATH=$PATH:$(go env GOPATH)/bin
echo "Successfully set $(go env GOPATH)/bin to PATH"

# Install toolchains
go install go.uber.org/mock/mockgen@latest
echo "Successfully installed mockgen $(mockgen --version)"

go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
echo "Successfully installed golangci-lint $(golangci-lint --version)"

# Tidy go packages
go mod tidy
echo "Successfully tidied go packages"