FROM golang:1.14-alpine AS builder

WORKDIR /mondane
RUN apk update && apk add --no-cache gcc musl-dev git make protoc
RUN go get github.com/golang/protobuf/protoc-gen-go
COPY . .
RUN make user-service

FROM alpine:latest
COPY --from=builder /mondane/user-service /user-service
EXPOSE 8080
CMD ["/user-service"]
