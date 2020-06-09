API_BINARY=api-server
MAIL_BINARY=mail-server

# Default target.
# Run tests and build all binaries.
all: test build

# Run tests
test:
	go test -v -cover -coverprofile=coverage.out ./...

# Build
build: $(API_BINARY) $(MAIL_BINARY)
	go build -o $(API_BINARY) cmd/api/main.go

# Build API_BINARY
$(API_BINARY):
	go build -o $(API_BINARY) cmd/api/main.go

# Build MAIL_BINARY
$(MAIL_BINARY):
	go build -o $(MAIL_BINARY) cmd/mail/main.go

.PHONY: all test build $(API_BINARY) $(MAIL_BINARY)
