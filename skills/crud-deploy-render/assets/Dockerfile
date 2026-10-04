# syntax=docker/dockerfile:1
# One image = Go API + built React SPA. Render builds this on every deploy.

# ---- frontend ----
FROM oven/bun:1 AS web
WORKDIR /web
COPY frontend/package.json frontend/bun.lock ./
RUN bun install --frozen-lockfile
COPY frontend/ ./
RUN bun run build

# ---- backend ----
# Keep the Go version in sync with the `go` line in backend/go.mod.
FROM golang:1.26-alpine AS api
WORKDIR /src
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ ./
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/server ./cmd/server

# ---- runtime ----
FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /app
COPY --from=api /out/server /app/server
COPY --from=web /web/dist /app/public
ENV APP_ENV=production \
    STATIC_DIR=/app/public \
    PORT=10000
EXPOSE 10000
USER nonroot:nonroot
ENTRYPOINT ["/app/server"]
