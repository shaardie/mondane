USER_SERVICE=user-service
USER_CLIENT=user-client

SERVICES=$(USER_SERVICE)
CLIENTS=$(USER_CLIENT)



# Default target.
# Build
build: $(SERVICES) $(CLIENTS)

# Run tests and build all binaries.
all: build

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
		user/proto/user.pb.go
		coverage.out

.PHONY: all build docker-images $(SERVICES) $(CLIENTS)
