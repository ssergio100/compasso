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

O painel não é embutido no executável. `assets_directory` aponta para os
templates HTML e arquivos estáticos externos. Durante o desenvolvimento, uma
alteração nesses arquivos aparece ao atualizar o navegador, sem recompilar ou
reiniciar o servidor. Essa fronteira também permite substituir a interface por
um build React ou outra tecnologia no futuro sem misturá-la ao núcleo Go.

Em produção, `secure_cookies` deve ser `true`. O SQLite definido por
`database_path` precisa ficar em volume persistente do servidor Docker.

## Produção com Docker e Cloudflare Tunnel

O arquivo `compose.production.yml` executa o servidor como usuário não-root,
remove capabilities, usa filesystem somente leitura e publica a porta apenas em
`127.0.0.1`. O `cloudflared` encaminha os dois subdomínios HTTPS para essa origem
local conforme `packaging/cloudflared/config.example.yml`.

O cliente usa `https://apicompasso.smresume.com` e o painel administrativo usa
`https://admcompasso.smresume.com`. Esses nomes pertencem à infraestrutura e não
estão fixados no código.

## Fases 7 e 8

O painel implementa login, expiração de sessão, CSRF, dispositivos, cotas
semanais, rotinas, senha local Argon2id, dashboard e histórico. A API de
heartbeat autentica cada dispositivo, consolida consumo/eventos de forma
idempotente e entrega política e comandos. O painel mostra conexão e revisão
aplicada pelo agente.
