# Stage 1: Build React Frontend
FROM node:20-alpine AS frontend-builder
WORKDIR /app/web

# Install frontend dependencies
COPY web/package*.json ./
RUN npm ci

# Build production assets
COPY web/ ./
RUN npm run build

# Stage 2: Build Go Backend
FROM golang:alpine AS backend-builder
ENV GOTOOLCHAIN=auto
WORKDIR /app

# Download Go dependencies
COPY go.mod go.sum ./
RUN go mod download

# Compile static binary
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /app/ctm ./cmd/ctm

# Stage 3: Minimal Distroless / Alpine Runtime
FROM alpine:3.20
RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

# Copy compiled backend binary and frontend static dist
COPY --from=backend-builder /app/ctm /app/ctm
COPY --from=frontend-builder /app/web/dist /app/web/dist

# Persistent data directory for SQLite database
RUN mkdir -p /data

EXPOSE 8080

# In-cluster defaults: SQLite DB at /data, static assets at /app/web/dist
ENTRYPOINT ["/app/ctm", "-port", "8080", "-db", "/data/cluster_timemachine.db", "-static-dir", "/app/web/dist", "-demo=false"]
