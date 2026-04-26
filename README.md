# NotifiQ

Sistema de notificação de eventos com Go, Redis e Docker.

## Arquitetura

- **API HTTP**: Recebe eventos via POST `/events`
- **Redis Queue**: Fila para processamento assíncrono
- **Consumer**: Processa eventos da fila em background

## Pré-requisitos

- Go 1.26+
- Docker & Docker Compose
- Redis (via Docker)

## Estrutura

```
notifiq/
├── cmd/main.go # Entry point
├── internal/
│   ├── api/handler.go # HTTP handlers
│   ├── event/event.go # Event model
│   ├── notifier/ # Notificadores
│   └── queue/redis.go # Redis queue implementation
├── Dockerfile
├── docker-compose.yml
└── go.mod
```


## Rodando Localmente

```bash
# 1. Iniciar Redis
docker compose up -d redis

# 2. Rodar a aplicação
go run ./cmd/main.go

# 3. Testar
curl -X POST http://localhost:8080/events \
  -H "Content-Type: application/json" \
  -d '{"id":"1","type":"user.created","payload":{"user":"john"}}'

```

# Subir tudo (app + redis)
docker compose up -d --build

# Ver logs
docker compose logs -f notifiq

# Parar
docker compose down

Desenvolvido por Matheus Lima.
