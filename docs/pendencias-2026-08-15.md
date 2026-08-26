# Pendências — 15/08/2026

## Sincronização em tempo real entre interface e servidor

**Status:** não iniciada.

### Direção arquitetural

- Aplicar somente à interface vigente `admin-ui`.
- Manter REST para carga inicial, consultas e comandos do usuário.
- Usar SSE para mudanças enviadas pelo servidor à interface.
- Criar um barramento de eventos em memória no `compasso-api`; produtores de
  eventos não podem depender diretamente do SSE.
- Manter o backend como fonte de verdade, inclusive para online/offline.
- Não usar WebSocket, Redis, broker externo ou polling contínuo de 1–2 segundos
  na v0.1.

### Menor fluxo completo

```text
heartbeat
  -> valida e persiste o estado
  -> EventBus.Publish(DeviceUpdated)
  -> GET /api/v1/admin/events (SSE autenticado)
  -> hook/serviço único no React
  -> atualização apenas do dispositivo afetado
```

Depois desse fluxo, expandir o mesmo mecanismo para alterações de política,
cota, bloqueio, pausa e tempo extra.

### Requisitos essenciais

- Fazer a carga inicial por REST antes de abrir o SSE.
- Reconectar sem exigir F5 e enviar keepalive periódico.
- Não alterar estado de domínio quando o SSE desconectar.
- Não transmitir senhas, hashes, tokens, segredos ou payloads desnecessários.
- Adequar timeout e buffering do servidor/proxy para conexões longas.
- Encapsular `EventSource` em serviço ou hook, sem espalhá-lo pelos componentes.

### Critérios de aceite

- Duas sessões administrativas recebem as mesmas atualizações.
- Heartbeat, online/offline e mudanças administrativas aparecem sem refresh.
- Uma perda temporária do SSE se recupera automaticamente.
- Não existe polling frequente como mecanismo principal de sincronização.
- O produtor publica eventos sem conhecer o SSE.

### Observação para a retomada

Antes de implementar, revisar o `WriteTimeout` atual da API e o proxy que serve
o acesso local. A solução anterior de logs escolheu polling de um segundo por
causa desse timeout; o SSE exige remover corretamente essa limitação para a rota
persistente, e não apenas contorná-la no frontend.
