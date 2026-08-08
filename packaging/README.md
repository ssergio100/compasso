# Packaging

Artefatos de instalação do cliente Linux.

- `systemd/tempo-agent.service`: unidade do daemon da fase 3, iniciada sem
  dependência de rede e reiniciada automaticamente em falhas.
- `config/tempo-agent.toml`: configuração-base de instalação; o nome da conta
  controlada deve ser substituído antes de habilitar o serviço.

O gate PAM da fase 4 é instalado por `tempo-pam-setup`. Ele cria o backup
`gdm-password.compasso.bak` antes da alteração e o utiliza na desinstalação.
Os pacotes finais serão adicionados na fase correspondente.
