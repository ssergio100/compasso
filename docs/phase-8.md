# Fase 8 — heartbeat e sincronização

## Implementação

- heartbeat HTTP versionado em `/api/v1/device/heartbeat`;
- autenticação individual por `device_id` e token aleatório mostrado uma vez;
- política completa por revisão, aplicada transacionalmente no agente;
- consumo absoluto monotônico e eventos locais idempotentes;
- comandos com entrega repetida até confirmação durável;
- capacidade do contrato informada pelo cabeçalho
  `X-Compasso-Protocol-Version: 2`, compatível com servidores antigos;
- retry com backoff sem interromper o motor local;
- estado online/offline, última conexão e revisão aplicada no painel;
- ações remotas de bônus, pausa, retomada e bloqueio.
- saldo confirmado pelo servidor somente na autorização inicial da sessão ou
  quando uma revisão relevante mudar;
- presença de sessão gráfica separada da presença online do agente;
- âncora persistente identificada pelo namespace privado do ciclo do serviço e
  pela sessão logind, consumida
  localmente sem recálculo periódico de cota e bônus.

## Testes automatizados

Executados por `make test`:

- revisão 10 recebe e aplica revisão 11;
- 30 minutos offline preservam política e contabilização;
- bônus local pendente é consolidado uma única vez após reconexão;
- heartbeat duplicado não duplica consumo, bônus ou auditoria;
- redução remota da cota produz decisão imediata de bloqueio;
- credencial incorreta recebe HTTP 401;
- comando é reenviado até ser confirmado e depois sai da fila.
- bônus remoto só é confirmado depois que a nova âncora foi persistida; agente
  antigo recebe HTTP 426 antes de consumir o comando incompatível;
- heartbeat comum não substitui a âncora nem devolve segundos ao contador;
- bônus remoto cria nova âncora e bônus local concorrente não é perdido;
- com um minuto restante, bônus remoto de 30 minutos produz âncora de 31
  minutos e não pode ser apresentado antes do reconhecimento do agente;
- perto da meia-noite, a data UTC do servidor não participa do crédito: a data
  local enviada no heartbeat decide o dia ativo e o saldo remanescente zera na
  virada local;
- agente online sem sessão gráfica mantém o contador do painel parado;
- daemon expira uma âncora de três segundos mesmo com cota local de oito horas.

## Teste integrado seguro no Zorin

Este teste usa servidor e agente na mesma máquina. Ele não instala serviços e
não altera PAM. Para impedir logout acidental da sessão de desenvolvimento,
mantenha a vigilância pausada durante todo o ensaio.

1. Reinicie o servidor com o código atual:

   ```bash
   cd /home/sergio/projetos/compasso
   TEMPO_ADMIN_LOGIN=admin TEMPO_ADMIN_PASSWORD='compasso-teste' \
     go run ./server/cmd/tempo-server -config server/config.toml
   ```

2. Abra `http://127.0.0.1:8081`, entre no dispositivo e clique primeiro em
   **Pausar vigilância**.
3. Na seção **Pareamento**, gere uma credencial. Deixe a página aberta.
4. Em outro terminal, execute e informe os dois valores mostrados:

   ```bash
   cd /home/sergio/projetos/compasso
   cp agent/config.example.toml agent/config.toml
   # Edite agent/config.toml e informe server_url, device_id e device_token.
   go run ./agent/cmd/tempo-agent -config agent/config.toml
   ```

   A mensagem sobre D-Bus indisponível é esperada sem instalação e não impede
   este teste.
5. Após até 10 segundos, atualize o painel e confirme `ONLINE` e que a revisão
   aplicada pelo cliente alcançou a revisão do servidor.
6. Altere uma cota, aguarde até 10 segundos e confirme que as duas revisões
   voltaram a ficar iguais.
7. Encerre apenas o agente com `Ctrl+C`, espere 65 segundos e confirme `OFFLINE`.
8. Inicie novamente o agente com o último comando e confirme o retorno a
   `ONLINE`.

Responda somente:

```text
pareamento: passou
online: passou
revisão: passou
offline: passou
reconexão: passou
```

Em caso de falha, substitua somente o item correspondente por
`falhou — descrição curta`.
