FROM golang:1.27-alpine AS build
WORKDIR /app
COPY go.mod go.sum ./
COPY vendor ./vendor
COPY . .
# Vendored build: no network access to the Go module proxy required.
RUN CGO_ENABLED=0 go build -mod=vendor -o /api ./cmd/api

FROM alpine:3.20
RUN apk add --no-cache ca-certificates wget \
    && addgroup -S app && adduser -S app -G app
WORKDIR /app
COPY --from=build --chown=app:app /api /app/api
COPY --chown=app:app migrations /app/migrations
EXPOSE 8080
USER app
CMD ["/app/api"]
