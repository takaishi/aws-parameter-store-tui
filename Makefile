.PHONY: build
build:
	go build -o dist/sstui ./cmd/sstui

.PHONY: install
install:
	go install github.com/takaishi/aws-ss-tui/cmd/sstui

.PHONY: test
test:
	go test -race ./...
