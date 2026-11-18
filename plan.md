O projeto pode evoluir para uma plataforma SaaS de disparos de e-mail transacionais e campanhas, com processamento assíncrono, controle de usuários e métricas de entrega.

A prioridade deve ser criar uma base segura e confiável antes de adicionar recursos avançados de “inteligência”.

## Direção do produto

O fluxo principal será:

```text
Usuário cria campanha
        ↓
Sistema valida destinatários
        ↓
Mensagens entram na fila Redis
        ↓
Worker Go envia os e-mails
        ↓
Provedor retorna status
        ↓
Sistema registra métricas no MySQL
```

## Fase 1: Definir o MVP

Funcionalidades essenciais:

- Cadastro e login de usuários
- JWT com access token e refresh token
- MFA por aplicativo autenticador ou e-mail
- Organizações ou contas isoladas
- Sistema de permissões baseado em papéis
- Cadastro de contatos
- Importação de contatos via CSV
- Criação de templates de e-mail
- Criação e agendamento de campanhas
- Disparo assíncrono usando Redis
- Histórico de mensagens enviadas
- Status: pendente, enviado, entregue, falhou e cancelado
- Dashboard básico com métricas
- Logs e tratamento de erros

Papéis iniciais:

- `owner`: controle total da conta
- `admin`: gerenciamento operacional
- `editor`: criação de campanhas e templates
- `viewer`: acesso somente para leitura

## Fase 2: Estruturar o backend Go

Organize o backend por responsabilidades:

```text
cmd/
  api/
  worker/

internal/
  auth/
  users/
  organizations/
  permissions/
  contacts/
  campaigns/
  templates/
  email/
  queue/
  metrics/
  database/
```

Componentes recomendados:

- API HTTP para autenticação e gerenciamento
- Worker separado para processamento da fila
- Redis para filas, locks e tarefas temporárias
- MySQL como fonte oficial dos dados
- Migrations versionadas
- Validação de payloads
- Configuração por variáveis de ambiente
- Graceful shutdown
- Contextos com timeout
- Logs estruturados

A separação entre API e worker é importante para que o envio de mensagens não bloqueie as requisições do frontend.

## Fase 3: Modelagem do MySQL

Tabelas principais:

- `users`
- `organizations`
- `organization_members`
- `roles`
- `permissions`
- `role_permissions`
- `refresh_tokens`
- `mfa_configs`
- `contacts`
- `contact_lists`
- `campaigns`
- `email_templates`
- `campaign_recipients`
- `messages`
- `message_events`
- `audit_logs`

Pontos importantes:

- Toda entidade deve possuir `organization_id` quando fizer parte de uma conta.
- E-mails devem ser normalizados e indexados.
- Campanhas e mensagens precisam de estados bem definidos.
- Eventos de envio devem ser armazenados para auditoria e métricas.
- Exclusões importantes devem preferir soft delete.
- Deve existir controle de idempotência para evitar disparos duplicados.

## Fase 4: Autenticação e segurança

Implementar nesta ordem:

1. Login com senha usando Argon2id ou bcrypt.
2. Access token JWT de curta duração.
3. Refresh token armazenado de forma segura e revogável.
4. MFA com TOTP.
5. Recuperação e troca de senha.
6. Rate limiting em login e endpoints sensíveis.
7. Controle de permissões no backend, nunca apenas no frontend.
8. Auditoria de login, alterações de permissões e disparos.
9. Sanitização de HTML dos templates.
10. Proteção contra abuso de campanhas e envio em massa.

O JWT deve identificar o usuário e a organização atual, mas a autorização precisa ser validada novamente no servidor.

## Fase 5: Integração de e-mail

Crie uma abstração de provedor:

```go
type EmailProvider interface {
    Send(ctx context.Context, message EmailMessage) (SendResult, error)
}
```

Assim será possível começar com um provedor e trocar futuramente sem reescrever o sistema.

O módulo deve tratar:

- SMTP ou API de provedor
- Retry com backoff
- Limite de envio
- Erros temporários e permanentes
- Bounce
- Reclamações de spam
- Webhooks de entrega
- Unsubscribe
- SPF, DKIM e DMARC
- Validação de remetente

Para o MVP, é melhor integrar um provedor especializado do que operar SMTP próprio.

## Fase 6: Frontend React + Shadcn UI

Telas prioritárias:

- Login
- MFA
- Recuperação de senha
- Dashboard
- Campanhas
- Criar e editar campanha
- Templates
- Contatos
- Usuários e permissões
- Configurações de envio
- Logs e detalhes de uma campanha

Estrutura sugerida:

```text
src/
  app/
  components/
  features/
    auth/
    campaigns/
    contacts/
    templates/
    users/
  lib/
  routes/
  hooks/
```

Boas decisões para o frontend:

- Separar componentes de interface dos componentes de negócio.
- Usar formulários com validação consistente.
- Exibir estados de carregamento, vazio e erro.
- Não armazenar tokens sensíveis de maneira insegura.
- Ocultar ações sem permissão, mas manter a validação no backend.
- Criar uma identidade visual própria, mesmo usando Shadcn UI.

## Fase 7: A “inteligência” do produto

Não começaria com IA generativa. Primeiro construa dados confiáveis.

Recursos inteligentes possíveis:

- Melhor horário de envio por destinatário
- Segmentação automática por comportamento
- Detecção de contatos inativos
- Sugestão de assunto
- Teste A/B de assuntos
- Reenvio automático para quem não abriu
- Limpeza de listas com baixa qualidade
- Detecção de anomalias na taxa de falhas
- Recomendação de frequência de campanhas

A primeira versão inteligente poderia ser um sistema de regras:

```text
Se o contato costuma abrir entre 9h e 11h,
priorizar esse intervalo para o próximo envio.
```

Depois, com dados suficientes, essas regras podem evoluir para modelos estatísticos ou IA.

## Fase 8: Confiabilidade e operação

Antes de considerar o sistema pronto para produção:

- Testes unitários no backend
- Testes de integração com MySQL e Redis
- Testes de autorização por papel
- Testes de expiração e revogação de tokens
- Testes de retry e idempotência
- Testes de carga da fila
- Health checks
- Métricas de API, fila e worker
- Alertas para falhas de envio
- Dead Letter Queue para mensagens que falharam repetidamente
- Backups do MySQL
- Docker Compose para desenvolvimento
- CI executando testes, lint e build

Métricas essenciais:

- Mensagens processadas por minuto
- Tempo médio na fila
- Taxa de entrega
- Taxa de bounce
- Taxa de abertura
- Taxa de clique
- Falhas por provedor
- Campanhas ativas
- Usuários ativos

## Roadmap sugerido

### Etapa 1: Fundação

- Definir escopo do MVP
- Criar `go.mod`
- Configurar MySQL e migrations
- Consolidar Docker Compose
- Criar API e worker
- Definir padrão de erros e logs

### Etapa 2: Conta e segurança

- Usuários
- Organizações
- Login
- JWT
- Refresh tokens
- MFA
- Papéis e permissões

### Etapa 3: Domínio de campanhas

- Contatos
- Listas
- Templates
- Campanhas
- Agendamento
- Validações

### Etapa 4: Envio

- Integração com provedor de e-mail
- Redis Queue
- Worker
- Retry
- Idempotência
- Registro de eventos

### Etapa 5: Frontend

- Autenticação
- Dashboard
- CRUD de campanhas
- Templates
- Contatos
- Usuários e permissões

### Etapa 6: Produção

- Observabilidade
- Testes de carga
- CI/CD
- Backups
- Webhooks
- Limites de uso
- Documentação da API

### Etapa 7: Inteligência

- Segmentação
- A/B testing
- Melhor horário de envio
- Sugestões de conteúdo
- Automação baseada em comportamento

## Principais riscos atuais

- O repositório ainda aparenta estar apenas como esqueleto/documentação.
- O README menciona arquivos que não estão presentes na estrutura atual.
- O Redis está previsto, mas o MySQL ainda precisa ser incorporado à arquitetura.
- Sem idempotência, uma campanha pode enviar e-mails duplicados.
- Sem controle de limites, o sistema pode ser utilizado para spam.
- Sem webhooks, as métricas de entrega ficarão incompletas.
- JWT e permissões mal implementados podem comprometer todas as contas.
- O domínio “inteligente” precisa ser definido por métricas reais, não apenas por uso de IA.

A melhor estratégia é lançar primeiro uma versão pequena, segura e observável: usuários, permissões, contatos, templates, campanhas e envio confiável. Depois disso, os dados coletados permitirão transformar o NotifiQ em um sistema realmente inteligente.
