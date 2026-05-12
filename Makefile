.PHONY: run build lint test tidy

run:
	SCOREOPS_DATABASE_URL="postgres://titlis:titlis@localhost:15432/titlis" \
	SCOREOPS_INTERNAL_SECRET="dev-secret" \
	go run ./cmd/scoreops

build:
	CGO_ENABLED=0 go build -ldflags="-s -w" -o bin/scoreops ./cmd/scoreops

lint:
	go vet ./...

test:
	go test ./... -count=1 -race

tidy:
	go mod tidy
