FROM registry.hub.docker.com/library/golang:1.14-alpine AS builder

WORKDIR /mondane
RUN apk update && apk add --no-cache gcc musl-dev git make protoc protobuf-dev
RUN go get github.com/golang/protobuf/protoc-gen-go
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN make user-service

FROM registry.hub.docker.com/library/alpine:latest
COPY --from=builder /mondane/user-service /user-service
EXPOSE 8082
CMD ["/user-service"]
