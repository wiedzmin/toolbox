default:
    @just --choose

# Build project
build:
    go build -v ./...

# Build project's docker infra
build-docker:
    docker-compose build

# Run generation commands
generate:
    go generate -v ./...

# Run unit tests
test:
    go test -v ./...

# Run functional tests
test-func:
    echo "add actual functional testing commands here"

# Lint cource code
lint:
    golangci-lint run ./...

# Fix linter complaints automatically
lint-fix:
    golangci-lint run --fix ./...

# Download dependencies
deps-download:
    go mod download

# Update dependencies versions
deps-update:
    go get -u ./...

# Garbage-collect dependencies
deps-gc:
    go mod tidy

# Publish source code updates
publish: generate build lint
    git push

# Enforce source code updates publishing
publish-force: generate build lint
    git push --force-with-lease

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
