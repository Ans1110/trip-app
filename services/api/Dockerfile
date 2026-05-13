FROM golang:1.25-alpine AS builder

WORKDIR /build

RUN apk add --no-cache git ca-certificates

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o tripapp ./cmd/server

FROM alpine:3.20

WORKDIR /app

RUN apk add --no-cache ca-certificates tzdata && addgroup -S app && adduser -S app -G app

COPY --from=builder /build/tripapp .
COPY --from=builder /build/migrations ./migrations

RUN chown -R app:app /app

USER app

EXPOSE 8080

ENTRYPOINT [ "./tripapp" ]