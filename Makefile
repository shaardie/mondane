USER_SERVICE=user-service
USER_CLIENT=user-client

MAIL_SERVICE=mail-service
MAIL_CLIENT=mail-client

HTTPCHECK_SERVICE=httpcheck-service
HTTPCHECK_CLIENT=httpcheck-client

ALERT_SERVICE=alert-service
ALERT_CLIENT=alert-client

SERVICES=$(USER_SERVICE) $(MAIL_SERVICE) $(HTTPCHECK_SERVICE) $(ALERT_SERVICE)
CLIENTS=$(USER_CLIENT) $(MAIL_CLIENT) $(HTTPCHECK_CLIENT) $(ALERT_CLIENT)



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

httpcheck/proto/httpcheck.pb.go: httpcheck/proto/httpcheck.proto
	protoc --proto_path=. --go_out=plugins=grpc:. --go_opt=paths=source_relative httpcheck/proto/httpcheck.proto

$(HTTPCHECK_SERVICE): httpcheck/proto/httpcheck.pb.go
	go build -o $(HTTPCHECK_SERVICE) cmd/$(HTTPCHECK_SERVICE)/main.go

$(HTTPCHECK_CLIENT): httpcheck/proto/httpcheck.pb.go
	go build -o $(HTTPCHECK_CLIENT) cmd/$(HTTPCHECK_CLIENT)/main.go

alert/proto/alert.pb.go: alert/proto/alert.proto mail/proto/mail.pb.go user/proto/user.pb.go
	protoc --proto_path=. --go_out=plugins=grpc:. --go_opt=paths=source_relative alert/proto/alert.proto

$(ALERT_SERVICE): alert/proto/alert.pb.go
	go build -o $(ALERT_SERVICE) cmd/$(ALERT_SERVICE)/main.go

$(ALERT_CLIENT): alert/proto/alert.pb.go
	go build -o $(ALERT_CLIENT) cmd/$(ALERT_CLIENT)/main.go

clean:
	go clean
	rm -rf $(SERVICES) $(CLIENTS)\
		user/proto/user.pb.go \
		mail/proto/mail.pb.go \
		alert/proto/alert.pb.go \
		httpcheck/proto/httpcheck.pb.go

.PHONY: all build $(SERVICES) $(CLIENTS)
