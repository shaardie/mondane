API_SERVICE=api-server
MAIL_SERVICE=mail-server
WORKER_BINARY=worker-server
USER_SERVICE=user-service
USER_CLIENT=user-client

SERVICES=$(API_SERVICE) $(MAIL_SERVICE) $(WORKER_BINARY) $(USER_SERVICE)
CLIENTS=$(USER_CLIENT)



# Default target.
# Build
build: $(SERVICES)

# Run tests and build all binaries.
all: test build

# Run tests
test: mail/api/api.pb.go user/proto/user.pb.go
	go test -v -cover -coverprofile=coverage.out ./...

# Build API_SERVICE
$(API_SERVICE): $(MAIL_SERVICE)
	go build -o $(API_SERVICE) cmd/api/main.go

# Build MAIL_SERVICE
$(MAIL_SERVICE):  mail/api/api.pb.go
	go build -o $(MAIL_SERVICE) cmd/mail/main.go

mail/api/api.pb.go: mail/api/api.proto
	protoc mail/api/api.proto --go_out=plugins=grpc:. --go_opt=paths=source_relative

# Buid WORKER_BINARY
$(WORKER_BINARY):
	go build -o $(WORKER_BINARY) cmd/worker/main.go

user/proto/user.pb.go: user/proto/user.proto
	protoc --proto_path=. --go_out=plugins=grpc:. --go_opt=paths=source_relative user/proto/user.proto

$(USER_SERVICE): user/proto/user.pb.go
	go build -o $(USER_SERVICE) cmd/$(USER_SERVICE)/main.go

$(USER_CLIENT): user/proto/user.pb.go
	go build -o $(USER_CLIENT) cmd/$(USER_CLIENT)/main.go

docker-images:
	docker build -t mondane/user-service -f docker/user-service/Dockerfile .

clean:
	go clean
	rm -rf $(SERVICES) $(CLIENTS)\
		mail/api/*.pb.go \
		user/proto/user.pb.go user/proto/user.pb.micro.go \
		coverage.out

.PHONY: all test build $(SERVICES) $(CLIENTS)
