# syntax=docker/dockerfile:1
FROM golang:1.26.10-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /app/server ./cmd \
    && CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /app/migrate ./cmd/migrate

FROM alpine:3.24
RUN apk add --no-cache ca-certificates tzdata \
    && addgroup -g 1001 app && adduser -u 1001 -G app -S app
ENV TZ=America/Fortaleza
COPY --from=build --chown=app:app /app/server /server
COPY --from=build --chown=app:app /app/migrate /migrate
USER app
EXPOSE 8080
ENTRYPOINT ["/server"]
