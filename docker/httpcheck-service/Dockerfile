FROM registry.hub.docker.com/library/golang:1.14-alpine AS builder

WORKDIR /mondane
RUN apk update && apk add --no-cache gcc musl-dev git make protoc
RUN go get github.com/golang/protobuf/protoc-gen-go
COPY . .
RUN make httpcheck-service

FROM registry.hub.docker.com/library/alpine:latest
COPY --from=builder /mondane/httpcheck-service /httpcheck-service
EXPOSE 80
CMD ["/httpcheck-service"]
