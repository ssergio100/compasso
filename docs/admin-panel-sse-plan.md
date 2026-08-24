# Plano de implementação — eventos em tempo real com SSE

## Objetivo

Substituir o polling de estado do painel administrativo por um fluxo de
Server-Sent Events (SSE). O servidor passa a empurrar para o `admin-ui-rhythm`
as mudanças que chegam dos agentes (heartbeat, transições de estado, offline),
sem que o navegador precise ficar consultando se algo mudou.

Essa mudança atende exclusivamente:

- `server/web`: novo endpoint SSE e um hub de publicação de eventos;
- `docs/prototypes/admin-ui-rhythm`: consumidor do stream no painel vigente.

Não altera o protocolo de heartbeat do agente (`protocol/v1`), o agente nem a
`local-ui`.

## Decisão de tecnologia

A comunicação contínua do Compasso é servidor → painel (estado dinâmico dos
dispositivos). As ações administrativas já são HTTP comum (`GET`, `POST`,
`PUT`, `DELETE`). Portanto:

- **SSE** para informações dinâmicas do dispositivo em exibição;
- **HTTP convencional** permanece para toda ação administrativa e consulta
  pontual.

WebSocket não é necessário: não há conversa bidirecional contínua, e o
`EventSource` do navegador fornece reconexão automática. A autenticação por
cookie de sessão (`tempo_admin_session`) já é transportada pelo navegador,
dispensando cabeçalhos customizados no stream.

## Arquitetura alvo

```text
Agente ──POST /api/v1/device/heartbeat──► Servidor
                                            │  ReceiveHeartbeat (persistência)
                                            │  └──► hub.Publish(device_updated)
                                            ▼
                                        hub de eventos
                                            │
Painel ◄──────── GET /api/v1/admin/devices/{id}/stream ──┘
          SSE: status (snapshot) + keep-alive
```

Fluxos de escrita continuam por HTTP e, quando o agente reconhece no heartbeat
seguinte, o estado aplicado chega pelo SSE:

```text
Admin ──POST /bonus, /commands, /policy──► Servidor (enfileira/grava)
Agente ──next heartbeat──► Servidor (reconhece) ──► SSE ──► Painel
```

## Protocolo SSE

### Endpoint

```
GET /api/v1/admin/devices/{device_id}/stream
```

Somente `GET`. Exige sessão administrativa válida (cookie
`tempo_admin_session`); sem sessão, responde `401 application/json`.

### Cabeçalhos da resposta

```http
HTTP/1.1 200 OK
Content-Type: text/event-stream
Cache-Control: no-store
Connection: keep-alive
X-Accel-Buffering: no
Access-Control-Allow-Origin: <origem configurada>
Access-Control-Allow-Credentials: true
```

### Eventos

| Evento | Quando | `data` |
| --- | --- | --- |
| `hello` | assinatura do stream | snapshot atual de `deviceLiveStatus` |
| `status` | heartbeat processado | snapshot atual de `deviceLiveStatus` |
| `device_offline` | timeout de `online_timeout` expirado sem heartbeat | snapshot com `online: false` |
| `communication` | novo registro de comunicação do dispositivo | `CommunicationLog` (mesmo JSON do `GET .../communication`) |

O `data` é o mesmo JSON de `GET /api/v1/admin/devices/{id}/status`
(`deviceLiveStatus` em `server/web/status.go`). O painel aplica o snapshot
diretamente sobre o dispositivo selecionado, sem reconciliação de contador
local (o `admin-ui-rhythm` exibe o valor enviado pelo servidor).

O evento `hello` hidrata o painel na primeira conexão e após reconexões,
eliminando a necessidade de refetch para recuperar eventos perdidos.

### Keep-alive

Comentário `: ping` a cada 15 segundos enquanto não houver eventos, para
impedir que proxies encerrem a conexão ociosa. A cada envio também é
revalidada a sessão; se a sessão expirou, o stream é encerrado.

## Servidor

### Hub de eventos — `server/web/events.go` (novo)

- `hub` mantém, por `device_id`, o conjunto de assinantes (canais com buffer
  pequeno, ex.: 8).
- `Subscribe(deviceID) (chan Event, unsubscribe func())`.
- `Publish(deviceID, Event)`: entrega a todos os assinantes do dispositivo;
  assinantes lentos ou desconectados são descartados sem bloquear o emissor.
- `HasSubscribers(deviceID) bool` para publicação sob demanda.

O `hub` vive no `App` (`server/web/app.go`) e é criado em `New`.

### Handler SSE — `server/web/events.go`

- Rota registrada em `app.go` e roteada por `adminDeviceAPI` em
  `server/web/admin_api.go` (novo recurso `stream`, como `status`).
- Assina o `hub`, escreve o evento `hello` com o snapshot atual e entra no
  loop: espera eventos do canal, envia `: ping` por timer e encerra quando
  `r.Context().Done()` (cliente desconectou) ou quando a sessão deixar de ser
  válida.
- Não passa pelo registro de comunicação administrativa: `administrativeCommunication`
  em `server/web/communication.go` deve ignorar o recurso `stream`, pois o
  registro ocorre somente quando a conexão é encerrada e geraria durações
  enganosas.
- CORS: reutiliza `corsHeaders` (`server/web/admin_api.go`), que já emite
  `Access-Control-Allow-Origin` e `Access-Control-Allow-Credentials` para
  `/api/v1/admin/`.

### Publicação por heartbeat — `server/web/api.go`

Em `handleHeartbeat`, após `ReceiveHeartbeat` bem-sucedido, publicar:

```go
if a.hub.HasSubscribers(deviceID) {
    _, _, liveStatus, err := a.loadDeviceLiveStatus(r.Context(), deviceID)
    if err == nil {
        a.hub.Publish(deviceID, eventStatus(liveStatus))
    }
}
```

`loadDeviceLiveStatus` só é executado quando há painel assistindo, evitando
custo de banco em heartbeats sem espectadores.

### Publicação de comunicação — `server/web/events.go`

A página de comunicação também deixa de fazer polling: após cada
`AppendCommunicationLog` (heartbeats em `server/web/api.go` e chamadas
administrativas em `server/web/communication.go`), o log gravado é publicado
como evento `communication` no stream do dispositivo (`publishCommunicationLog`),
somente quando há assinantes. O painel consome o evento com `after`/`id`
contínuo, sem nova consulta periódica.

### Detalhes de negócio — `server/web/communication.go`

O middleware `logAdministrativeCommunication` injeta um contexto de detalhes
(`addCommunicationDetail`) que os handlers usam para registrar o significado da
ação: `bonus_minutes`, `command`, `warning_minutes`, `routine_name`,
`routine_action`, `action` e `device_name`. A política de privacidade continua
vigente no storage (`validateCommunicationLog` em `server/storage/communication.go`):
segredos — senhas, tokens, cookies e corpos HTTP — nunca são gravados; apenas
parâmetros de negócio não sensíveis passam a constar no log.

### Detecção de offline — `server/web/events.go`

Heartbeat não detecta ausência; `isOnline` em `server/web/status.go` é
derivado de `last_seen_at` + timeout. É preciso um verificador em segundo
plano:

- `StartOfflineDetector(ctx)` inicia a goroutine (chamado no `main` do
  servidor) com ticker de `online_timeout / 2` (padrão 60s → a cada 30s).
- Usa `ListDevices` do store; para cada dispositivo com
  `now - last_seen_at > online_timeout` e com assinantes, publica
  `device_offline` uma única vez por período de ausência (memoriza o último
  estado publicado e reseta quando um heartbeat volta).

### Configuração e proxy

- O `admin-ui-rhythm` (Vite) conecta-se à origem da API configurada; o
  `EventSource` respeita `connect-src`, então **nenhuma mudança de CSP é
  necessária** além de permitir a origem da API.
- Qualquer proxy reverso à frente da API (ex.: `deploy/`) precisa, para a rota
  do stream: `proxy_buffering off`, `proxy_read_timeout` longo (ex.: 3600s) e
  respeitar `X-Accel-Buffering: no`.

## Frontend — `admin-ui-rhythm`

### `src/api.ts`

Novo método na classe `API`:

```ts
openStream(id: string): EventSource {
  return new EventSource(`${this.base}/api/v1/admin/devices/${encodeURIComponent(id)}/stream`, { withCredentials: true });
}
```

`{ withCredentials: true }` é obrigatório porque o painel e a API vivem em
origens diferentes.

### `src/App.tsx`

- `useEffect` abre um único `EventSource` do dispositivo em exibição
  (`selected.id`) e fecha na troca de dispositivo ou no desmonte.
- Os eventos `hello`, `status` e `device_offline` aplicam o `LiveStatus`
  (`src/types.ts`) sobre o dispositivo em `setDevices`, atualizando
  online/offline, contadores, cota, bônus e `last_seen_at` — sem novo `load()`.
- `hello` hidrata o painel na primeira conexão e após reconexões automáticas
  do `EventSource`, dispensando refetch para recuperar eventos perdidos.
- `device_offline` apenas marca `online: false` e atualiza a interface.
- Sessão expirada: quando o stream nunca abre por `401`, a autenticação falha
  no fluxo normal e o painel volta ao login; o `EventSource` permanece fechado
  e não cria chamadas em loop.
- Bônus pendente: mantém o polling de `bonusStatus` enquanto o agente não
  confirma (`synchronizeBonus`) — consulta periódica só enquanto há bônus
  pendente.

### `src/communication/CommunicationPage.tsx`

- Abre o `EventSource` do dispositivo ao montar e o fecha ao desmontar/trocar
  de dispositivo.
- O evento `communication` (mesmo JSON da listagem) é adicionado à lista com
  `mergeEvents`, atualizando `maxID` — sem polling de 1 segundo.
- Fallback: o polling incremental (`after=maxID`) roda somente enquanto o
  stream não está conectado; na reconexão (`onopen`) um `refresh()` cobre
  eventuais lacunas. O indicador "Atualização ao vivo" reflete o estado do
  stream.
- A linha visível é uma frase clara (`describeEvent`): identifica quem fez o
  quê — "O administrador concedeu 30 min de bônus a Zorin", "O administrador
  enviou o comando de pausa para Zorin" — e o resultado em linguagem simples
  (Concluído / Com atenção / Falhou). Detalhes técnicos (rota, método,
  correlação, status HTTP, duração) ficam ao clicar na linha.
- Filtro "Ocultar batimentos" (desativado por padrão) reduz o ruído dos
  heartbeats, mantendo a lista focada em ações e respostas.

## Fases de implementação

### Etapa 1 — hub e endpoint no servidor

- [x] Criar `hub` de eventos e integrá-lo ao `App`.
- [x] Implementar `GET /api/v1/admin/devices/{id}/stream` com `hello`,
  keep-alive e revalidação de sessão.
- [x] Excluir o recurso `stream` do registro de comunicação administrativa.
- [x] Registrar testes HTTP: autenticação exigida, `hello` na conexão e
  publicação de evento (`server/web/events_test.go`).

### Etapa 2 — publicação por heartbeat

- [x] Publicar `status` em `handleHeartbeat` somente quando houver assinantes.
- [x] Teste: heartbeat com assinante gera eventos `status` e `communication`.

### Etapa 3 — detecção de offline

- [x] Implementar verificador em segundo plano com ticker (`StartOfflineDetector`).
- [x] Publicar `device_offline` uma vez por período de ausência e resetar no
  próximo heartbeat.

### Etapa 4 — consumidor no `admin-ui-rhythm`

- [x] Adicionar `openStream` ao `src/api.ts` (EventSource com `withCredentials`).
- [x] Aplicar os eventos `hello`/`status`/`device_offline` sobre o dispositivo
  em `src/App.tsx`, sem polling periódico de estado.
- [x] Consumir o evento `communication` em `src/communication/CommunicationPage.tsx`,
  com polling de 1s somente como fallback enquanto o stream estiver desconectado.
- [x] Validar compilação: `npm run typecheck` e `npm run build`.
- [ ] Validar em navegador com painel e API em origens diferentes.

### Etapa 5 — ajustes de servidor, proxy e distribuição

- [x] Zerar o `WriteTimeout` do servidor para não encerrar o stream autenticado
  (`server/cmd/tempo-server/main.go`).
- [ ] Configurar `proxy_buffering off` e timeouts nos proxies à frente da API.
- [ ] Confirmar que a CSP do painel não bloqueia `EventSource`.

## Critérios de pronto

- [ ] Painel de dispositivo atualiza online/offline, contadores e estados de
  BLOCK/PAUSE sem polling periódico de estado.
- [ ] Página de comunicação mostra novos registros sem polling periódico
  (fallback de 1s apenas com o stream desconectado).
- [ ] Uma mudança feita por outra aba ou outro administrador aparece no painel
  aberto sem recarregar a página.
- [ ] Perda de conexão de rede não exige recarregar o painel: o `EventSource`
  reconecta e o `hello` reposiciona o estado.
- [ ] Sessão expirada encerra o stream e o painel volta ao login.
- [ ] O servidor não faz leituras extras de banco por heartbeat sem assinantes.
- [x] Compilação Go (`go test ./...`), `make lint` e `npm run build` do
  `admin-ui-rhythm` aprovam a mudança.

## Fora do escopo

- Alterar o protocolo de heartbeat do agente (`protocol/v1`);
- WebSocket ou comunicação bidirecional;
- Atualização em tempo real da listagem de dispositivos, que permanece sob
  demanda;
- Alterações em `local-ui`;
- Escolha de framework para o frontend.
