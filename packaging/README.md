# Packaging

Artefatos de instalação do cliente Linux.

- `systemd/tempo-agent.service`: unidade do daemon da fase 3, iniciada sem
  dependência de rede e reiniciada automaticamente em falhas.
- `config/tempo-agent.toml`: configuração-base de instalação; o nome da conta
  controlada deve ser substituído antes de habilitar o serviço.

As regras PAM, o instalador com rollback e os pacotes serão adicionados nas
fases correspondentes.
