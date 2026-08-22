FROM golang:1.26-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o distributed-storage .
RUN go build -o storage-node ./cmd/storage-node

FROM alpine:3.21
WORKDIR /app
COPY --from=builder /app/distributed-storage .
COPY --from=builder /app/storage-node ./storage-node

RUN mkdir -p /app/storage

EXPOSE 8080
EXPOSE 8081
CMD ["./distributed-storage"]