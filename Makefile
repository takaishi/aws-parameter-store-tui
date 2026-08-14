.PHONY: build
build:
	go build -o dist/aws-parameter-store-tui ./cmd/aws-parameter-store-tui
	go build -o dist/aws-secrets-manager-tui ./cmd/aws-secrets-manager-tui
	go build -o dist/aws-ecs-tui ./cmd/aws-ecs-tui
	go build -o dist/aws-security-group-tui ./cmd/aws-security-group-tui
	go build -o dist/aws-ec2-tui ./cmd/aws-ec2-tui
	go build -o dist/aws-route53-tui ./cmd/aws-route53-tui
	go build -o dist/aws-cloudwatch-logs-tui ./cmd/aws-cloudwatch-logs-tui

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
