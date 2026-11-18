# NotifiQ

Plataforma de campanhas de e-mail assíncronas, com autenticação, MFA, permissões por organização, configuração SMTP protegida e histórico de entrega.

## Stack

- Backend: Go 1.26, `net/http`, MySQL 8.4 e Redis 7.
- Frontend: React, TypeScript, Vite e componentes com padrão visual Shadcn UI.
- Segurança: JWT curto, refresh token rotativo, bcrypt e TOTP MFA.
- Operação: Docker Compose, worker assíncrono, retries e Dead Letter Queue.

## Arquitetura

```text
Browser -> Nginx (web:3000) -> /api/* -> API Go (notifiq:8080) -> MySQL
                                               |
                                               v
                                             Redis -> Worker SMTP
```

O scheduler consulta destinatários no MySQL, marca jobs como enfileirados e publica no Redis. O worker assume o destinatário, envia via SMTP, persiste o resultado e agenda retries. Falhas terminais ficam na Dead Letter Queue.

## Execução 100% containerizada

Pré-requisito: Docker com Compose v2.

```bash
docker compose up -d --build
```

URLs locais:

- Frontend: `http://localhost:3000`
- API direta: `http://localhost:4025`
- Health: `http://localhost:4025/healthz`
- Readiness: `http://localhost:4025/readyz`

O frontend é compilado em uma imagem Node e servido por Nginx. O Nginx encaminha `/api/*` para a API Go. MySQL e Redis ficam somente na rede interna do Compose.

```bash
docker compose ps
docker compose logs -f notifiq
docker compose logs -f web
docker compose down
```

Use `docker compose down -v` somente quando quiser apagar também os dados locais do MySQL.

## Configuração

Os defaults são somente para desenvolvimento. Em produção, substitua todos os segredos:

| Variável | Default | Uso |
| --- | --- | --- |
| `HTTP_ADDR` | `:8080` | Endereço da API |
| `REDIS_ADDR` | `localhost:4000` | Redis |
| `MYSQL_ADDR` | `localhost:3306` | MySQL |
| `MYSQL_DATABASE` | `notifiq` | Banco |
| `MYSQL_USER` | `notifiq` | Usuário MySQL |
| `MYSQL_PASSWORD` | `notifiq` | Senha MySQL |
| `JWT_SECRET` | desenvolvimento | Assinatura JWT, mínimo de 32 caracteres |
| `EMAIL_ENCRYPTION_KEY` | desenvolvimento | Chave base64 AES-GCM |
| `WEBHOOK_SECRET` | desenvolvimento | Assinatura HMAC de webhooks |
| `PASSWORD_RESET_URL` | localhost | Tela de recuperação |
| `INVITE_URL` | localhost | Tela de convite |
| `EMAIL_JOB_QUEUE` | `notifiq:email_jobs` | Fila principal |
| `EMAIL_DEAD_LETTER_QUEUE` | `notifiq:email_jobs:dead_letters` | DLQ |

Cada variável sensível também aceita o sufixo `_FILE`, por exemplo
`JWT_SECRET_FILE=/run/secrets/jwt_secret`. O conteúdo do arquivo tem prioridade
sobre a variável direta, permitindo usar Docker secrets, Kubernetes Secrets ou
um agente de secret manager sem armazenar credenciais no Compose.

## Funcionalidades

- Cadastro, login, logout, refresh token e MFA TOTP.
- Recuperação, redefinição e troca de senha.
- Organizações com `owner`, `admin`, `editor` e `viewer`.
- Convites por e-mail com expiração e uso único.
- Contatos individuais e importação CSV de até 5 MB.
- Templates HTML e preview seguro em `iframe sandbox`.
- Criação, agendamento, edição, cancelamento e duplicação de campanhas.
- Redis, retries com backoff, idempotência do scheduler e DLQ.
- Webhooks assinados para entrega, abertura, clique, bounce e reclamação.
- Histórico por campanha, métricas e auditoria de ações críticas.

## CSV

O arquivo precisa ter as colunas `name` e `email`:

```csv
name,email
Ana Souza,ana@example.com
Carlos Lima,carlos@example.com
```

Linhas inválidas e duplicadas são informadas no resultado. A chave única por organização evita duplicação no banco.

## Rotas principais

Públicas:

- `POST /auth/register`
- `POST /auth/login`
- `POST /auth/refresh`
- `POST /auth/password-reset/request`
- `POST /auth/password-reset/confirm`
- `POST /members/invite/accept`
- `POST /webhooks/email` com `X-Webhook-Signature`

Autenticadas:

- `PUT /auth/password/change`
- `GET/POST /contacts`, `POST /contacts/import`
- `GET/POST /templates`
- `GET/POST /campaigns`
- `PUT /campaigns/update`, `POST /campaigns/cancel`, `POST /campaigns/duplicate`
- `GET/PUT /members`, `POST /members/invite`, `DELETE /members/remove`
- `GET /metrics/delivery`
- `GET /metrics/delivery/history?campaign_id=...`
- `GET /admin/dlq?limit=50` (owner/admin)
- `POST /admin/dlq` com `{ "id": "dlq-..." }` (owner/admin)

Todas as rotas de domínio validam organização e role no backend.

A DLQ recebe um identificador por item, pode ser consultada por administradores
e permite reprocessar uma mensagem para a fila principal. Analise o motivo da
falha antes de reprocessar para evitar loops de envio.

## Desenvolvimento

Infraestrutura:

```bash
docker compose up -d mysql redis
```

Backend:

```bash
go run ./cmd/main.go
```

Frontend:

```bash
cd web
npm ci
npm run dev
```

Testes:

```bash
go test ./...
cd web && npm test -- --run && npm run build
```

Sem Go instalado no host:

```bash
docker run --rm -v "$PWD":/app -w /app golang:1.26-alpine go test ./...
```

## Migrations

As migrations em `db/migrations` são aplicadas automaticamente na inicialização da API, em ordem por nome e registradas em `schema_migrations`.

## Produção

Antes do deployment:

1. Troque todos os defaults de segredo.
2. Configure TLS em um reverse proxy.
3. Configure SPF, DKIM e DMARC.
4. Use um provedor transacional real e configure webhooks.
5. Configure backup e restauração testada do MySQL.
6. Monitore `/readyz`, fila principal, DLQ e taxa de bounce.
7. Restrinja o endpoint de webhook e os acessos administrativos.

O Compose fornece um ambiente completo para desenvolvimento e ambientes pequenos. Para produção, use um provedor transacional, valide SPF/DKIM/DMARC no domínio remetente, monte os segredos via `*_FILE`, centralize os logs e métricas, configure backups e faça uma revisão de segurança antes da exposição pública.

## CI

`.github/workflows/ci.yml` executa `gofmt`, testes Go, testes/build frontend, validação do Compose e build das imagens da API e do frontend.
