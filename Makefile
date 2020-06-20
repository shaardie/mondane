USER_SERVICE=user-service
USER_CLIENT=user-client
MAIL_SERVICE=mail-service
MAIL_CLIENT=mail-client

SERVICES=$(USER_SERVICE) $(MAIL_SERVICE)
CLIENTS=$(USER_CLIENT) $(MAIL_CLIENT)



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

mail/proto/mail.pb.go: mail/proto/mail.proto
	protoc --proto_path=. --go_out=plugins=grpc:. --go_opt=paths=source_relative mail/proto/mail.proto

$(MAIL_SERVICE): mail/proto/mail.pb.go
	go build -o $(MAIL_SERVICE) cmd/$(MAIL_SERVICE)/main.go

$(MAIL_CLIENT): mail/proto/mail.pb.go
	go build -o $(MAIL_CLIENT) cmd/$(MAIL_CLIENT)/main.go

docker-images:
	docker build -t mondane/$(USER_SERVICE) -f docker/$(USER_SERVICE)/Dockerfile .
	docker build -t mondane/$(MAIL_SERVICE) -f docker/$(MAIL_SERVICE)/Dockerfile .

clean:
	go clean
	rm -rf $(SERVICES) $(CLIENTS)\
		user/proto/user.pb.go \
		mail/proto/mail.pb.go

.PHONY: all build docker-images $(SERVICES) $(CLIENTS)
