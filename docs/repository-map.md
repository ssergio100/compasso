# Mapa do repositório

O Compasso é um monorepo: agente, servidor, interfaces, protocolo e arquivos de
distribuição evoluem juntos, embora sejam entregues como artefatos separados.

## Diretórios de produto

| Caminho | Para que serve | Entra na distribuição? |
| --- | --- | --- |
| `agent/` | daemon privilegiado, regras, sessão Linux, sincronização e banco local | pacote `compasso-client` |
| `server/` | API administrativa, heartbeat, autenticação e banco central | pacote `compasso-server` |
| `protocol/` | tipos do contrato de sincronização compartilhados por agente e servidor | compilado nos binários |
| `admin-ui/` | interface administrativa React; é a única implementação vigente do painel | build estático independente |
| `local-ui/` | configuração inicial e bônus local em GTK 4 | pacote `compasso-client` |

### Pacotes do agente

| Caminho | Responsabilidade |
| --- | --- |
| `agent/cmd/tempo-agent/` | ponto de entrada do serviço |
| `agent/cmd/tempo-agent-configure/` | helper privilegiado usado pelo assistente gráfico |
| `agent/daemon/` | coordenação do ciclo de política, sessão, persistência e sync |
| `agent/policy/` | cálculo puro de permissão, bloqueio e próximos eventos |
| `agent/session/` | descoberta e bloqueio de sessões por `loginctl` |
| `agent/alert/` | entrega de alertas à sessão gráfica controlada |
| `agent/storage/` | SQLite, migrações e estado durável do cliente |
| `agent/syncclient/` | cliente HTTP do heartbeat e confirmação de comandos |
| `agent/syncstatus/` | estado sanitizado de comunicação exposto à UI local |
| `agent/localapi/` | serviço D-Bus para bônus e diagnóstico local |
| `agent/localauth/` | senha local Argon2id e limitação de tentativas |
| `agent/setup/` | validação e gravação segura da configuração inicial |
| `agent/config/` | leitura e validação do TOML do agente |

### Pacotes do servidor

| Caminho | Responsabilidade |
| --- | --- |
| `server/cmd/tempo-server/` | ponto de entrada da API |
| `server/web/` | rotas HTTP, sessão administrativa, CSRF, SSE e heartbeat |
| `server/storage/` | domínio, consultas SQLite e migrações centrais |
| `server/config/` | configuração TOML e variáveis de ambiente |

### Interfaces

| Caminho | Responsabilidade |
| --- | --- |
| `admin-ui/src/features/` | páginas e fluxos do painel por domínio |
| `admin-ui/src/communication/` | atividades humanas e diagnóstico de comunicação |
| `admin-ui/src/hooks/` | stream SSE e notificações |
| `admin-ui/src/assets/` | ilustrações realmente usadas pelo build |
| `admin-ui/src/mock.ts` | dados fictícios do modo de demonstração |
| `local-ui/configure_agent.py` | pareamento e escolha da conta controlada |
| `local-ui/bonus_dialog.py` | bônus local e acesso à configuração avançada |

## Distribuição e operação

| Caminho | Para que serve |
| --- | --- |
| `packaging/` | metadados Debian, systemd, D-Bus, Polkit, AppStream e ícone |
| `deploy/admin-ui/` | Nginx e Compose para servir o build estático do painel |
| `scripts/` | builds, testes de pacotes, publicação, backup, restauração e atualização |
| `compose.yaml` | build e execução da API do servidor |
| `.env.server.example` | parâmetros de exemplo para o Compose da API |

O pacote do servidor contém somente a API e suas ferramentas operacionais. O
painel é compilado e implantado separadamente; veja [deploy/README.md](../deploy/README.md).

## Documentação e automação

| Caminho | Para que serve |
| --- | --- |
| `docs/` | guias vigentes, arquitetura e índice documental |
| `.github/workflows/ci.yml` | validação automatizada de pushes e pull requests |
| `AGENTS.md` | regras de escopo para assistentes de desenvolvimento |
| `admin-ui/AGENTS.md` | critérios adicionais de coerência visual do painel |
| `Makefile` | comandos públicos de build, teste e empacotamento |
| `VERSION` | versão semântica do código em desenvolvimento |

## Arquivos locais que não pertencem ao Git

| Caminho/padrão | Conteúdo esperado |
| --- | --- |
| `bin/` | binários compilados localmente |
| `dist/` | pacotes Debian e checksums |
| `var/` | bancos e estado de execução local |
| `secrets/` | chaves ou credenciais locais, se necessárias |
| `.private/` | histórico, notas, diagnósticos, materiais de origem e deploys pessoais |
| `.playwright-cli/` | capturas e logs transitórios de navegador |
| `admin-ui/node_modules/` | dependências instaladas pelo npm |
| `admin-ui/dist/` | build estático gerado pelo Vite |
| `**/config.toml` e `.env*` | configurações concretas de cada ambiente |

Os exemplos rastreados usam nomes terminados em `.example`. Antes de publicar
um commit, use `git status --short --ignored` para conferir as duas fronteiras.
