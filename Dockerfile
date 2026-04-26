# //notifiq/Dockerfile

# Build stage
FROM golang:1.26-alpine AS builder

WORKDIR /app

# Copiar arquivos de dependências primeiro (cache)
COPY go.mod go.sum ./
RUN go mod download

# Copiar código fonte
COPY . .

# Compilar
RUN CGO_ENABLED=0 GOOS=linux go build -o notifiq ./cmd/main.go

# Final stage
FROM alpine:latest

WORKDIR /app

# Copiar binário do builder
COPY --from=builder /app/notifiq .

# Porta exposta
EXPOSE 8080

# Comando de execução
CMD ["./notifiq"]