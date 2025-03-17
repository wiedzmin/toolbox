default:
    @just --choose

# Build project
build:
    go build -v ./...

# Build project
install:
    go install -v ./...

# Run generation commands
generate:
    go generate -v ./...

# Lint source code
lint:
    golangci-lint run ./...

# Update dependencies versions
deps-update:
    go get -u ./...

# Garbage-collect dependencies
deps-gc:
    go mod tidy

# Install "pre-commit" hooks
pre-commit-install:
    pre-commit install

# Update "pre-commit" hooks
pre-commit-update-hooks:
    pre-commit autoupdate

# cleanup current devenv
devenv-cleanup:
    rm -rf ${PWD}/.devenv ${PWD}/.direnv
    rm -f ${PWD}/devenv.lock ${PWD}/.devenv.flake.nix ${PWD}/.pre-commit-config.yaml
    touch .envrc

# cleanup and GC current devenv
devenv-cleanup-and-gc:
    rm -rf ${PWD}/.devenv ${PWD}/.direnv
    rm -f ${PWD}/devenv.lock ${PWD}/.devenv.flake.nix ${PWD}/.pre-commit-config.yaml
    sudo nix-collect-garbage -d
    touch .envrc
