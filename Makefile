.PHONY: build
build:
	go build -o dist/aws-parameter-store-tui ./cmd/aws-parameter-store-tui
	go build -o dist/aws-secrets-manager-tui ./cmd/aws-secrets-manager-tui

.PHONY: install
install:
	go install github.com/takaishi/aws-tui/cmd/...

.PHONY: test
test:
	go test -race ./...

# Regenerate CREDITS (dependency license notices).
# go-localereader ships no license file in its tagged module, so its
# upstream MIT license is kept in .credits-extra and appended manually.
.PHONY: credits
credits:
	go run github.com/Songmu/gocredits/cmd/gocredits@latest -skip-missing -w .
	cat .credits-extra >> CREDITS

# Fail if a dependency linked into the binaries has a forbidden/unknown license.
.PHONY: license-check
license-check:
	go run github.com/google/go-licenses@latest check ./cmd/...
