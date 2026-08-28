# Agent

O `tempo-agent` é o daemon privilegiado instalado no computador controlado. Ele
avalia a última política local, contabiliza sessões gráficas e usa
`loginctl lock-session` quando uma regra bloquear o uso, preservando os
aplicativos abertos.

## Comportamento da fase 3

- somente sessões gráficas locais `x11` ou `wayland` da conta configurada são
  contabilizadas;
- tela bloqueada continua contando enquanto a sessão gráfica existir;
- TTY, SSH, sessões remotas, greeter e contas diferentes não contam;
- a ausência de rede não interfere no ciclo local;
- consumo é salvo a cada cinco segundos por padrão e no desligamento normal;
- uma sessão já em uso é encerrada quando a política muda de liberada para
  bloqueada;
- uma sessão que surge durante um bloqueio não é encerrada durante `opening`;
  depois de chegar a `active`, o agente aguarda dez segundos de estabilização.
- quando a sincronização está configurada, uma sessão nova sem saldo também
  aguarda o primeiro heartbeat concluído depois do login. Uma resposta com
  tempo libera a sessão; uma resposta que mantenha o bloqueio permite o logout.
  Falha de rede não conta como resposta e mantém a sessão aberta enquanto o
  agente tenta novamente.

O requisito atual permite a autenticação. Quando a política continuar
bloqueando, o agente espera a sessão gráfica ficar estabelecida e solicita o
bloqueio de tela pelo logind. Não encerra a sessão e não usa
`loginctl terminate-session` como fallback.

## Execução de desenvolvimento

Copie `config.example.toml`, ajuste `controlled_user` e use um banco que já
contenha uma política válida:

```bash
go run ./agent/cmd/tempo-agent -config ./agent/config.toml
```

O pacote `policy` contém o motor puro, `storage` mantém SQLite e checkpoints,
`session` usa logind para descoberta, e `sessionlogout` seleciona adaptadores
pelas capacidades do D-Bus da sessão; `daemon` coordena o ciclo. A unidade de produção
está em `packaging/systemd/tempo-agent.service`.

## Sincronização da fase 8

Quando `server_url`, `device_id` e `device_token` estão configurados, o pacote
`syncclient` envia heartbeat, consumo e eventos pendentes. Políticas completas
são aplicadas por revisão e comandos são confirmados de forma durável. Se os
três valores estiverem vazios ou o servidor estiver indisponível, o daemon
continua aplicando integralmente o estado local.

O heartbeat anuncia `X-Compasso-Protocol-Version: 2` e as capacidades
`next-heartbeat-seconds` e `command-ack-receipts`. Quando o servidor devolve
`next_heartbeat_seconds`,
o agente usa esse intervalo no ciclo normal seguinte, limitado entre 1 segundo
e 10 minutos. Campo ausente ou inválido usa o fallback embutido de 3 segundos;
o valor não é persistido, e o `heartbeat_interval` local legado não controla o
processo instalado. Em um bônus remoto, o agente persiste a nova âncora antes
de registrar o comando como aplicado. O
reconhecimento enviado no heartbeat seguinte significa, portanto, que o saldo
autorizado já está durável; a data usada é a mesma data local enviada no
heartbeat, inclusive perto da meia-noite.

Comandos de controle são persistidos separadamente e só são reconhecidos
depois de o daemon observar o efeito em `LockedHint`. O servidor devolve os IDs
de comando que recebeu para o agente remover esses reconhecimentos locais, sem
retransmiti-los indefinidamente.

Uma sessão nova solicita ao servidor uma âncora com saldo confirmado. A
identidade combina o namespace privado do ciclo do serviço e a sessão logind.
Depois da resposta, o daemon
subtrai somente o uso monotônico posterior; heartbeats sem mudança não reaplicam
saldo. O heartbeat também informa presença gráfica separadamente do estado
online, permitindo que o painel pare o contador depois do logout.

Para o ensaio sem instalação, siga `docs/phase-8.md`. Mantenha a vigilância
pausada para não encerrar a própria sessão de desenvolvimento.
