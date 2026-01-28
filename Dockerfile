FROM golang:1.21-alpine AS builder
WORKDIR /app
RUN apk add --no-cache git
COPY go.mod go.sum ./
RUN go mod download
COPY . ./
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o server ./cmd/server

FROM alpine:3.20
WORKDIR /app
RUN adduser -D -g '' appuser
COPY --from=builder /app/server /app/server
COPY internal/db/migrations /app/internal/db/migrations
EXPOSE 8080
USER appuser
ENTRYPOINT ["/app/server"]
