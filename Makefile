.PHONY: build
build:
	go build -o dist/sstui ./cmd/sstui

.PHONY: install
install:
	go install github.com/takaishi/aws-parameter-store-tui/cmd/sstui

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

# Fail if a dependency linked into the binary has a forbidden/unknown license.
.PHONY: license-check
license-check:
	go run github.com/google/go-licenses@latest check ./cmd/sstui
