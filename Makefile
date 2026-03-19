APP_NAME := mynamr
MAIN_PACKAGE := ./cmd/mynamr

.PHONY: build test run snapshot-release

build:
	go build -o ./dist/$(APP_NAME) $(MAIN_PACKAGE)

test:
	go test ./...

run:
	go run $(MAIN_PACKAGE)

snapshot-release:
	goreleaser release --snapshot --clean
