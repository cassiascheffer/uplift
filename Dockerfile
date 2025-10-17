# syntax = docker/dockerfile:1

# Build frontend assets
FROM node:20-alpine AS frontend
WORKDIR /app
COPY package*.json ./
RUN npm ci
COPY . .
RUN npm run build

# Build Go binary
FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY go.* ./
RUN go mod download
COPY . .
RUN go build -o uplift ./cmd/server

# Production image
FROM alpine:latest
RUN apk --no-cache add ca-certificates tzdata
WORKDIR /app

# Copy Go binary
COPY --from=builder /app/uplift .

# Copy built frontend assets to static directory
COPY --from=frontend /app/dist ./static

# Expose port
EXPOSE 8080

# Run the application
CMD ["./uplift"]
