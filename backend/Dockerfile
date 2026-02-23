# Build stage
FROM golang:1.23-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o rms-backend ./cmd/server

# Final stage
FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/rms-backend .
COPY --from=builder /app/.env .  
EXPOSE 8080
CMD ["./rms-backend"]