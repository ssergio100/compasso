# Server API

O `tempo-server` oferece somente a API JSON de administração e sincronização.
Ele não lê, renderiza nem serve a interface web. O painel independente vive em
`admin-ui/` e recebe a URL da API em runtime.

## Desenvolvimento

Crie a configuração local e inicie a API sem credenciais de instalação:

```bash
cp server/config.example.toml server/config.toml
go run ./server/cmd/tempo-server -config server/config.toml
```

Use `http://127.0.0.1:8080/healthz` para verificar o processo. A interface
administrativa detecta o banco novo e oferece a criação do primeiro acesso no
navegador. A configuração inicial deixa de existir após essa criação.

Em produção, `secure_cookies` deve ser `true`. O SQLite definido por
`database_path` precisa ficar em volume persistente do servidor Docker.

## Produção com Docker

O `compose.yaml` executa API e frontend em imagens independentes, como usuários
sem privilégios, com capabilities removidas e filesystem somente leitura. Bind,
portas, origem administrativa, URL pública da API e cookies seguros são
configurados por `.env`, sem dependência de um produto de exposição específico.

Cloudflare Tunnel, proxy reverso, VPN, acesso somente por LAN ou qualquer outra
forma de exposição são decisões externas ao servidor Compasso.

## Fases 7 e 8

O painel implementa login, expiração de sessão, CSRF, dispositivos, cotas
semanais, rotinas, senha local Argon2id, dashboard e histórico. A API de
heartbeat autentica cada dispositivo, consolida consumo/eventos de forma
idempotente e entrega política e comandos. Para clientes novos, ele também
confirma o saldo ao autorizar uma sessão gráfica e só envia outra âncora quando
uma revisão relevante mudar. Presença do agente e presença de sessão gráfica
são estados separados; o painel só anima o contador quando há sessão ativa. O
painel também mostra conexão e revisão aplicada pelo agente.
