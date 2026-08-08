# Packaging

Artefatos de instalação do cliente Linux.

- `systemd/tempo-agent.service`: unidade do daemon da fase 3, iniciada sem
  dependência de rede e reiniciada automaticamente em falhas.
- `config/tempo-agent.toml`: configuração-base de instalação; o nome da conta
  controlada deve ser substituído antes de habilitar o serviço.

O gate PAM da fase 4 é instalado por `tempo-pam-setup`. Ele cria o backup
`gdm-password.compasso.bak` antes da alteração e o utiliza na desinstalação.
Os pacotes finais serão adicionados na fase correspondente.

Na fase 6, `dbus/br.com.tempo.Agent.conf` autoriza chamadas locais ao método de
bônus e `applications/br.com.tempo.LocalBonus.desktop` adiciona o diálogo GTK ao
menu do Zorin.

Na fase 9, `scripts/install-agent-securely.sh` instala apenas o agente com
configuração e estado pertencentes a root, valida o token no journal e habilita
a unidade endurecida. Ele não altera PAM. O exemplo `cloudflared` publica os
subdomínios HTTPS mantendo a origem HTTP restrita ao loopback do servidor.
