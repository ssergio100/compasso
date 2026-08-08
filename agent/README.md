# Agent

O `tempo-agent` é o daemon privilegiado instalado no computador controlado. Ele
avalia a última política local, contabiliza sessões gráficas e solicita ao
`systemd-logind` o encerramento da sessão quando uma regra bloquear o uso.

## Comportamento da fase 3

- somente sessões gráficas locais `x11` ou `wayland` da conta configurada são
  contabilizadas;
- tela bloqueada continua contando enquanto a sessão gráfica existir;
- TTY, SSH, sessões remotas, greeter e contas diferentes não contam;
- a ausência de rede não interfere no ciclo local;
- consumo é salvo a cada cinco segundos por padrão e no desligamento normal;
- ao iniciar bloqueado, ou surgir uma nova sessão durante um bloqueio, o agente
  solicita imediatamente o logout ao logind.

Impedir um novo login de forma antecipada pertence ao gate PAM da fase 4. Até
ele ser instalado, uma sessão que reapareça será encerrada pelo próximo ciclo.

## Execução de desenvolvimento

Copie `config.example.toml`, ajuste `controlled_user` e use um banco que já
contenha uma política válida:

```bash
go run ./agent/cmd/tempo-agent -config ./agent/config.toml
```

O pacote `policy` contém o motor puro, `storage` mantém SQLite e checkpoints,
`session` integra com logind e `daemon` coordena o ciclo. A unidade de produção
está em `packaging/systemd/tempo-agent.service`.
