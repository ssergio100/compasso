# Pendências — 25/08/2026

## Intervalo de heartbeat orientado pelo servidor

**Status:** concluída em 25/08/2026.

### Problema

O intervalo de heartbeat está gravado na configuração local do agente. Em uma
instalação existente, a atualização do pacote preserva esse arquivo e, por isso,
uma mudança de `10s` para `3s` exigiria edição manual. Esse procedimento não é
adequado para o público familiar e não deve ser responsabilidade dos pais.

### Decisão

O servidor deve informar, em cada resposta válida de heartbeat, quanto tempo o
agente deve aguardar antes de enviar o próximo “estou vivo”. O agente aplica o
valor automaticamente, sem configuração ou intervenção do usuário.

Fluxo esperado:

```text
agente envia heartbeat
  -> servidor processa estado e comandos
  -> servidor responde com o intervalo do próximo heartbeat
  -> agente valida o intervalo
  -> agente aguarda esse período e sincroniza novamente
```

### Regras propostas

- Acrescentar ao protocolo de resposta um campo explícito, por exemplo
  `next_heartbeat_seconds`.
- Manter no agente um intervalo seguro embutido para a primeira conexão e para
  respostas que não tragam o campo, preservando compatibilidade durante a
  atualização gradual.
- Validar o valor recebido dentro de limites mínimos e máximos. Uma resposta
  inválida nunca pode criar um loop de requisições nem deixar o agente sem
  comunicação por tempo excessivo.
- O valor enviado pelo servidor deve prevalecer sobre o intervalo legado
  presente em `/etc/tempo-agent/config.toml`; pais não devem editar esse arquivo.
- Não é necessário persistir o intervalo recebido. Após reiniciar, o agente usa
  o valor seguro embutido até o primeiro heartbeat bem-sucedido.
- Manter o heartbeat adicional imediato após o recebimento de comandos, pois
  ele transporta a confirmação sem esperar o intervalo normal.
- Manter a retentativa com backoff quando o servidor estiver indisponível.
- Não expor esse ajuste na interface familiar. Ele é um detalhe operacional do
  servidor e do agente.

### Compatibilidade e atualização

- O servidor deve continuar aceitando agentes antigos que ignorem o novo campo.
- O agente novo deve continuar funcionando temporariamente com servidores que
  ainda não enviem o novo campo.
- A configuração local antiga deve continuar sendo aceita para que uma
  atualização não impeça a inicialização do serviço, mas não deve exigir
  manutenção manual.
- Depois que todos os agentes suportarem o campo, avaliar a retirada de
  `heartbeat_interval` dos arquivos gerados em novas instalações.

### Critérios de aceite

- Uma instalação existente passa a usar o intervalo definido pelo servidor sem
  edição local.
- Alterar o intervalo no servidor afeta os próximos ciclos dos agentes novos.
- Campo ausente ou inválido mantém o agente operando com um valor seguro.
- O intervalo nunca sai dos limites aceitos pelo agente.
- Comandos continuam produzindo uma confirmação imediata.
- Testes cobrem servidor antigo/agente novo, servidor novo/agente antigo,
  limites inválidos, reconexão e reinício do agente.

### Observação para a retomada

Antes de implementar, definir se o servidor enviará um valor global ou se terá
capacidade de escolher intervalos por dispositivo. Para a primeira versão, um
valor global com limites fixos no agente é suficiente e reduz a complexidade.

### Implementação concluída

- adotado um intervalo global do servidor, configurável por
  `heartbeat_interval` ou `TEMPO_HEARTBEAT_INTERVAL`;
- incluído `next_heartbeat_seconds` nas respostas para agentes que anunciam a
  capacidade `next-heartbeat-seconds`;
- mantido fallback embutido de 3 segundos no agente, com limites fixos de 1
  segundo a 10 minutos e sem persistência do valor remoto;
- mantida a leitura da configuração local legada, sem usá-la para controlar o
  processo instalado;
- preservados o heartbeat imediato de confirmação e a retentativa com backoff;
- cobertas por testes a negociação com agentes antigos, a ausência do campo em
  servidores antigos, alterações entre ciclos, limites inválidos, reinício,
  reconexão e confirmação imediata de comandos.
