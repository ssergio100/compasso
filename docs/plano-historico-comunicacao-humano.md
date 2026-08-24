# Plano de melhoria — histórico humano da comunicação

## Objetivo

Transformar a página **Comunicação** da interface administrativa vigente
`docs/prototypes/admin-ui-rhythm` em um histórico que qualquer responsável
consiga entender apenas lendo.

Uma ação do administrador deve aparecer como uma única história, do pedido ao
resultado. Consultas automáticas, heartbeats e reenvios deixam de ocupar linhas
independentes e passam a ser etapas internas dessa história.

Este plano se aplica somente a:

- `docs/prototypes/admin-ui-rhythm`, a interface administrativa vigente;
- `server/web` e `server/storage`, que coordenam e registram as operações;
- `protocol/v1`, `agent/syncclient` e `agent/storage` somente quando for
  necessário confirmar entrega, aplicação ou falha no computador.

Não contempla `admin-ui`, `local-ui` nem outros protótipos.

## Problema atual

Hoje o histórico registra requisições HTTP. No bônus remoto, uma única ação
humana pode produzir:

1. um `POST /bonus`;
2. vários `GET /commands/{operation_id}` feitos pela interface a cada dois
   segundos;
3. um ou mais heartbeats do agente;
4. respostas do servidor ao heartbeat;
5. um `GET /devices/{device_id}` para atualizar a tela.

Todas essas ocorrências aparecem lado a lado, embora só exista uma intenção do
usuário: por exemplo, **adicionar 15 minutos ao Zorin**. Para compreender o
resultado é necessário conhecer endpoints, UUIDs, polling, heartbeat e status
HTTP.

O histórico também não torna explícitas as respostas para as perguntas mais
importantes:

- quem iniciou a ação;
- o que foi pedido;
- se o servidor guardou o pedido;
- se o computador estava disponível;
- se o servidor tentou entregar a ação;
- se o agente aplicou a alteração;
- quanto tempo levou;
- quantas tentativas reais foram necessárias;
- se a operação continua pendente ou precisa de atenção.

## Princípios do novo histórico

1. **Uma intenção, uma história.** Uma concessão de bônus gera um único item,
   atualizado até chegar a um resultado.
2. **O servidor aparece como coordenador.** A interface mostra que o servidor
   recebe, guarda, entrega e recebe a confirmação do computador.
3. **Sucesso só após confirmação.** Resposta `202` significa “pedido guardado”,
   nunca “aplicado no computador”.
4. **Tentativa tem significado humano.** Contar somente vezes em que o servidor
   incluiu a operação em uma resposta ao agente. Polling da interface, SSE,
   keep-alive e consultas de tela não são tentativas de entrega.
5. **Espera não é falha.** Computador offline ou confirmação demorada deve ser
   apresentado como “aguardando”, com a razão conhecida.
6. **Texto primeiro, técnica sob demanda.** Rotas, UUIDs, revisões, status HTTP e
   correlações ficam em “Detalhes técnicos”, fechado por padrão.
7. **Sem afirmações que o sistema não possa provar.** “Enviado” significa que o
   servidor colocou a operação na resposta; “aplicado” exige confirmação
   durável do agente.
8. **Idempotência visível no resultado, invisível no ruído.** Reenvios da mesma
   operação atualizam a contagem, mas não criam novas histórias nem novo bônus.
9. **Eventos técnicos continuam disponíveis.** O diagnóstico bruto permanece
   separado do histórico humano, com retenção e acesso próprios.
10. **A criação também é idempotente.** A interface gera uma chave antes do
    envio; repetir a mesma requisição por clique duplo, timeout ou reconexão
    devolve a operação já criada em vez de conceder o benefício novamente.

## Mapa da comunicação

```text
Administrador
    │ pede “Adicionar 15 min ao Zorin”
    ▼
Interface administrativa
    │ envia a intenção uma única vez
    ▼
Servidor
    │ valida, persiste e cria a operação
    │ aguarda o próximo contato do computador
    ▼
Agente no Zorin ── heartbeat periódico ──► Servidor
    ▲                                      │
    └──── política/comando pendente ───────┘
    │ aplica e grava localmente
    └──── confirmação no heartbeat seguinte ──► Servidor
                                                  │ conclui a operação
Interface ◄──────── evento em tempo real ─────────┘
```

O navegador não conversa diretamente com o agente. O servidor é a fonte de
verdade e o ponto de encontro dos dois lados.

## Modelo humano da operação

### Estados

| Estado interno | Texto principal | Quando usar |
| --- | --- | --- |
| `received` | Pedido recebido pelo servidor | A API validou e persistiu a ação |
| `waiting_device` | Aguardando o computador | Ainda não houve contato capaz de transportar a ação |
| `offered` | Enviado ao computador; aguardando confirmação | A operação foi incluída em uma resposta de heartbeat |
| `completed` | Concluído | O agente confirmou aplicação durável ou a revisão aplicada foi observada |
| `rejected` | Não foi possível registrar | A API recusou a solicitação antes de criar a operação |
| `attention` | Ainda não confirmado | O prazo de acompanhamento venceu, houve incompatibilidade ou repetidas falhas |
| `failed` | Não foi possível aplicar | O agente informou uma falha definitiva e segura para exibição |

`attention` não encerra necessariamente a operação. Ao reconectar e confirmar,
o mesmo item deve mudar para `completed`.

### Etapas persistidas

Cada operação deve manter uma linha do tempo compacta:

1. **Administrador pediu** — ação, valor, computador e horário.
2. **Servidor recebeu** — validação e persistência concluídas.
3. **Servidor enviou** — horário da primeira oferta e número de ofertas.
4. **Computador confirmou** — aplicação durável reconhecida pelo servidor.
5. **Interface atualizada** — entrega do resultado ao painel; esta etapa é de
   apresentação e não altera o sucesso de domínio.

Etapas repetidas não geram novas linhas. A etapa “Servidor enviou” passa de
“1 tentativa” para “3 tentativas”, por exemplo.

### Exemplo fechado

```text
15 min adicionados ao Zorin                                      Concluído
Solicitado por Administrador às 20:54 · concluído em 14 segundos

✓ O servidor recebeu e guardou o pedido.
✓ O servidor enviou o pedido ao Zorin.
✓ O Zorin aplicou o tempo e confirmou ao servidor.

Concluído após 1 tentativa de entrega.
```

### Exemplo aguardando computador offline

```text
15 min para o Zorin                                      Aguardando computador
O pedido está guardado e será enviado quando o Zorin voltar a se conectar.
Solicitado há 3 minutos · nenhuma tentativa de entrega ainda
```

### Exemplo com reenvio

```text
Pausa do monitoramento do Zorin                                  Concluído
O Zorin confirmou a alteração após 3 tentativas de entrega.
```

### Exemplo que denuncia duas submissões

Duas chamadas distintas de criação devem produzir duas histórias fáceis de
comparar:

```text
15 min adicionados ao Zorin · 20:54:11                           Concluído
15 min adicionados ao Zorin · 20:55:12                           Concluído
```

Assim uma duplicidade humana ou da interface fica evidente, sem ser confundida
com tentativas de sincronização da mesma operação.

## Arquitetura de dados

### Separar atividade humana de transporte técnico

Manter dois conceitos:

- **operação:** representação durável da intenção e do resultado, usada pela
  tela principal;
- **evento técnico:** amostra de requisição, heartbeat ou erro de transporte,
  usada somente no diagnóstico avançado.

O `communication_log` atual pode continuar como registro técnico durante a
migração, mas não deve alimentar diretamente a lista humana.

### Projeção de operação

Criar uma estrutura persistente, por exemplo `operation_activity`, com:

- `id`: o mesmo identificador opaco do comando quando existir;
- `device_id`;
- `kind`: bônus, pausa, retomada, bloqueio, política, rotina, senha etc.;
- `origin`: administrador, agente local ou sistema;
- `requested_by`: nome não sensível do administrador, quando disponível;
- `status`;
- campos de negócio sanitizados, como `bonus_seconds` ou `command_kind`;
- `expected_policy_revision`, quando a confirmação ocorrer por revisão;
- `delivery_attempts`;
- `created_at`, `first_offered_at`, `last_offered_at`, `acknowledged_at`;
- `attention_code` ou `failure_code` estável, sem texto sensível;
- `updated_at`.

Se for necessária auditoria de cada mudança de estado, complementar com
`operation_activity_event`. A operação é a visão atual; os eventos são a linha
do tempo. Não duplicar saldo, política ou comando nessa estrutura: as tabelas de
domínio continuam sendo a fonte de verdade.

### Instrumentar `device_command`

Adicionar ao comando pelo menos:

- `delivery_attempts`;
- `first_offered_at`;
- `last_offered_at`.

Incrementar a tentativa quando o servidor incluir o comando na resposta de um
heartbeat. A redação deve ser precisa: antes do `command_ack`, o servidor sabe
que ofereceu a operação, mas ainda não pode garantir que o agente recebeu.

Quando `command_acks` trouxer o identificador, atualizar `acknowledged_at`,
concluir a operação e publicar a mudança em tempo real.

### Operações confirmadas por revisão

Políticas, limites, rotinas e senha local não precisam necessariamente virar
comandos individuais. Registrar a revisão esperada na operação e concluí-la
quando um heartbeat informar `applied_policy_revision >= expected_revision`.

Pausa, retomada, bloqueio e desbloqueio devem receber um identificador de
operação retornado pela API, como já ocorre com o bônus. Hoje o endpoint de
comandos retorna apenas uma mensagem; o contrato deve passar a retornar
`operation_id`.

Renomear dispositivo, emitir ou revogar token e excluir dados são operações
somente do servidor. Elas podem ser concluídas imediatamente e devem dizer
explicitamente “Alteração salva no servidor”; não mencionar o agente.

### Confirmação de falha pelo agente

Na primeira entrega, o protocolo atual já permite provar sucesso: o agente só
inclui o comando em `command_acks` depois de persistir a aplicação.

Para distinguir “continua tentando” de uma falha definitiva no agente, planejar
uma evolução compatível do protocolo com resultados estruturados:

```json
{
  "command_results": [
    {
      "id": "operation-id",
      "status": "applied",
      "code": "ok",
      "applied_at": "..."
    }
  ]
}
```

Os códigos precisam ser fechados e traduzidos pelo servidor/interface. Nunca
transportar mensagem arbitrária que possa conter caminhos, usuários, tokens ou
outros dados locais. Durante a compatibilidade, `command_acks` continua valendo
como `applied`.

## API e tempo real

### Criação

Toda ação assíncrona deve retornar um envelope comum:

```json
{
  "operation_id": "...",
  "status": "received",
  "message": "Pedido recebido pelo servidor."
}
```

A interface envia uma `Idempotency-Key` opaca, criada uma vez por confirmação
do usuário. O servidor persiste a chave vinculada ao dispositivo, tipo e
administrador e devolve a mesma operação quando recebe novamente a mesma
requisição. Reutilizar uma chave com conteúdo diferente deve ser rejeitado.

O botão também fica indisponível imediatamente por uma trava síncrona, antes da
próxima renderização do React. A trava melhora a experiência, mas a garantia de
não duplicação pertence ao servidor.

### Consulta

Disponibilizar uma representação humana da operação:

```text
GET /api/v1/admin/devices/{device_id}/operations/{operation_id}
GET /api/v1/admin/devices/{device_id}/operations?limit=...
```

Ela deve trazer o estado, os campos de negócio sanitizados, as etapas e a
contagem de tentativas. O frontend não deve montar significado a partir de
rotas HTTP.

### SSE

Estender o stream já planejado/implementado para publicar
`operation_updated` sempre que uma operação mudar de estado ou de contagem.

O fluxo normal do frontend passa a ser:

1. criar a operação por HTTP;
2. inserir ou atualizar uma única história na tela;
3. receber `operation_updated` até a conclusão;
4. reconciliar a listagem na reconexão do SSE.

Remover o polling de dois segundos de `synchronizeBonus`. Se for mantida uma
consulta de recuperação após queda do SSE, ela não gera item humano, não conta
como tentativa e usa intervalo progressivo, não contínuo.

Centralizar a conexão SSE em um serviço/hook por sessão e dispositivo. A tela
de estado e a tela de atividade devem consumir o mesmo fluxo, em vez de abrir
conexões independentes.

## Política do registro técnico

O middleware administrativo deixa de gerar atividade humana para:

- `GET device`;
- `GET status`;
- `GET commands/{operation_id}`;
- `GET operations`;
- `GET communication`;
- conexão, keep-alive e reconexão do stream.

Essas chamadas podem alimentar métricas agregadas ou um log técnico com
retenção curta. Também não criar uma linha humana para todo heartbeat normal.

Criar atividade humana para:

- uma nova intenção administrativa;
- mudança relevante de estado da operação;
- computador que ficou offline ou voltou;
- falha de autenticação ou incompatibilidade que exija ação;
- evento local relevante enviado pelo agente;
- conclusão confirmada pelo agente.

No diagnóstico técnico, permitir visualizar rota, correlação, status HTTP,
revisões e horários. Identificadores devem ter botão “Copiar”, mas nunca ser o
título ou a informação principal.

## Interface

### Organização

Renomear a navegação principal de **Comunicação** para **Atividade** ou
**Histórico**. Dentro dela:

- aba padrão **Ações e resultados**;
- aba secundária **Diagnóstico técnico**;
- filtros humanos: Todos, Aguardando, Concluídos, Precisam de atenção;
- agrupamento por dia;
- busca por computador, tipo de ação ou valor, não por UUID como caso comum.

### Linha fechada

Exibir somente:

- frase completa no passado ou estado de espera;
- computador;
- horário;
- resultado em linguagem simples;
- duração e tentativas quando forem relevantes.

Não exibir na linha fechada: `GET`, `POST`, endpoint, `202`, UUID, revisão ou
“heartbeat”.

### Detalhe aberto

Mostrar uma linha do tempo com três atores identificáveis:

- **Administrador/interface:** fez o pedido;
- **Servidor Compasso:** validou, guardou e ofereceu a alteração;
- **Computador/agente:** aplicou e confirmou.

Depois da explicação humana, oferecer “Ver detalhes técnicos” com os dados
necessários ao suporte.

### Vocabulário obrigatório

- “Pedido recebido” em vez de “202 Accepted”.
- “Aguardando o computador” em vez de “pending”.
- “Servidor enviou ao computador” em vez de “heartbeat_response”.
- “Computador aplicou e confirmou” em vez de “command_acknowledgement”.
- “Tentativa de entrega” somente para uma oferta servidor → agente.
- “Consulta automática” somente no diagnóstico, se necessário.

## Plano de implementação

### Etapa 1 — contrato e linguagem

- Inventariar bônus, controles, políticas, rotinas, senha, dispositivo e
  eventos locais.
- Definir para cada tipo: início, evidência de entrega, evidência de aplicação,
  espera, atenção e falha.
- Fechar o vocabulário português e os códigos internos estáveis.
- Criar testes de texto com exemplos reais antes dos componentes visuais.

### Etapa 2 — persistência da operação

- Criar a migração da projeção de operações e dos campos de tentativa.
- Fazer criação e mudança de estado na mesma transação do comando/política.
- Preservar operações em reinício do servidor.
- Garantir idempotência por `operation_id`.
- Persistir e validar a chave idempotente usada na criação.
- Manter detalhes sensíveis fora da projeção e de seus eventos.

### Etapa 3 — instrumentação servidor–agente

- Contar ofertas de comandos no heartbeat.
- Concluir comandos por `command_acks`.
- Concluir mudanças de política pela revisão aplicada.
- Publicar transições de online/offline sem registrar todo heartbeat.
- Acrescentar resultado estruturado do agente em fase compatível posterior,
  caso seja necessário explicar falhas definitivas.

### Etapa 4 — API e SSE de operações

- Padronizar `operation_id` nas ações assíncronas.
- Padronizar `Idempotency-Key` e o comportamento de repetição/conflito.
- Criar endpoints de listagem e detalhe de operações.
- Publicar `operation_updated` no SSE.
- Reconciliar eventos perdidos após reconexão.
- Compartilhar uma única conexão SSE entre estado e atividade.
- Remover o polling frequente do bônus.

### Etapa 5 — experiência humana

- Construir “Ações e resultados” com uma história por operação.
- Implementar estados, linha do tempo e textos definidos neste plano.
- Exibir tentativas somente no resumo final ou quando houver demora/reenvio.
- Manter diagnóstico técnico separado e recolhido.
- Tornar duplicidades reais visualmente óbvias.

### Etapa 6 — redução e retenção do ruído

- Parar de transformar leituras automáticas em atividade humana.
- Reduzir heartbeats normais a métricas agregadas.
- Definir retenção independente para operações humanas e logs técnicos.
- Medir volume antes/depois e confirmar que uma operação comum gera um item.

### Etapa 7 — validação e implantação

- Testes unitários do lifecycle e da idempotência no storage.
- Testes de integração cobrindo dois heartbeats: entrega e confirmação.
- Testes de SSE, reconexão e deduplicação.
- Testes da interface com linguagem e acessibilidade.
- Implantar primeiro para bônus, validar com registros reais e então expandir
  para controles, políticas e rotinas.

## Cenários obrigatórios de teste

1. Bônus confirmado na primeira oferta.
2. Bônus entregue após várias ofertas sem duplicar o saldo.
3. Computador offline no momento do pedido e confirmação após reconexão.
4. Painel fechado durante a operação e reconstrução correta ao reabrir.
5. SSE desconectado e reconciliação sem duplicar a história.
6. Duas submissões reais de 15 minutos exibidas como duas operações.
7. Clique ou reenvio HTTP com a mesma chave idempotente criando uma operação.
8. Política concluída somente quando a revisão aplicada for observada.
9. Falha/rejeição distinguida de simples espera.
10. Servidor reiniciado com operação pendente.
11. Agente antigo sem capacidade nova, com orientação compreensível.
12. Nenhum segredo presente em operação, evento, SSE ou detalhe técnico.
13. Clique duplo e reenvio HTTP com a mesma chave retornando a mesma operação.
14. Reutilização da chave com minutos ou ação diferentes sendo rejeitada.

## Migração do histórico existente

Os registros atuais não guardam correlação suficiente para reconstruir com
segurança todas as etapas de cada operação. Não inventar confirmações passadas.

- iniciar o histórico humano a partir da implantação da nova projeção;
- manter os registros anteriores em “Diagnóstico técnico — histórico legado”
  até o fim da retenção configurada;
- migrar somente fatos inequivocamente correlacionáveis;
- indicar claramente quando uma operação antiga possui informação parcial.

## Critérios de aceite

- Uma concessão de bônus produz exatamente um item no histórico humano.
- O item responde, sem conhecimento técnico: quem pediu, o quê, para qual
  computador, o que o servidor fez, se o computador confirmou, quando e em
  quantas tentativas.
- O sucesso só aparece depois da confirmação do agente.
- Oito consultas automáticas não produzem oito linhas nem oito tentativas.
- Reenvios da mesma operação não duplicam bônus nem histórias.
- Duas criações distintas aparecem claramente como duas ações.
- Computador offline é espera explicada, não falha genérica.
- O histórico sobrevive a recarga da página e reinício do servidor.
- O diagnóstico técnico continua disponível sem dominar a experiência.
- Usuário, token, senha, cookie e payload sensível nunca são registrados.

## Ordem recomendada

Começar pelo bônus, pois ele já possui `operation_id`, comando idempotente e
confirmação explícita. Ele estabelece o modelo completo:

```text
pedido → servidor guardou → servidor ofereceu → agente aplicou → agente confirmou
```

Depois aplicar o mesmo modelo a pausa, retomada, bloqueio e desbloqueio. Em
seguida cobrir políticas, limites, rotinas e senha pela revisão aplicada. Por
último reorganizar eventos locais e o diagnóstico técnico.
