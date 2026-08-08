# Server

O `tempo-server` serve o painel administrativo e, a partir da fase 8, a API de
sincronização. Não há domínio fixo no código; Docker, Cloudflare Tunnel e os
subdomínios continuam sendo decisões de implantação.

## Desenvolvimento

Crie a configuração local e informe as credenciais do primeiro administrador
somente no primeiro início:

```bash
cp server/config.example.toml server/config.toml
TEMPO_ADMIN_LOGIN=admin TEMPO_ADMIN_PASSWORD='uma-senha-de-teste' \
  go run ./server/cmd/tempo-server -config server/config.toml
```

Abra `http://127.0.0.1:8080`. Depois que o primeiro administrador existe, as
variáveis podem ser omitidas; elas nunca substituem uma conta já criada.

Em produção, `secure_cookies` deve ser `true`. O SQLite definido por
`database_path` precisa ficar em volume persistente do servidor Docker.

## Fases 7 e 8

O painel implementa login, expiração de sessão, CSRF, dispositivos, cotas
semanais, rotinas, senha local Argon2id, dashboard e histórico. A API de
heartbeat autentica cada dispositivo, consolida consumo/eventos de forma
idempotente e entrega política e comandos. O painel mostra conexão e revisão
aplicada pelo agente.
