# Build stage
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Copie des fichiers de dépendances
COPY go.mod ./
RUN go mod download

# Copie du code source
COPY . .

# Compilation du binaire statique pour Linux
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o /app/bin/serveoapi ./cmd/api

# Final stage (minimal image)
FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /root/

# Copie du binaire compilé
COPY --from=builder /app/bin/serveoapi .

EXPOSE 8080

ENV PORT=8080
ENV ENV=production

CMD ["./serveoapi"]
