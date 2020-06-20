FROM golang:1.14-alpine AS builder

WORKDIR /mondane
RUN apk update && apk add --no-cache gcc musl-dev git make protoc
RUN go get github.com/golang/protobuf/protoc-gen-go
COPY . .
RUN make mail-service

FROM alpine:latest
COPY --from=builder /mondane/mail-service /mail-service
EXPOSE 8080
CMD ["/mail-service"]
