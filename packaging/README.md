# Packaging

Artefatos de instalação do cliente Linux.

- `systemd/tempo-agent.service`: unidade do daemon da fase 3, iniciada sem
  dependência de rede e reiniciada automaticamente em falhas.
- `config/tempo-agent.toml`: configuração-base de instalação; o nome da conta
  controlada deve ser substituído antes de habilitar o serviço.

O gate PAM da fase 4 é instalado por `tempo-pam-setup`. Ele cria o backup
`gdm-password.compasso.bak` antes da alteração e o utiliza na desinstalação.
O pacote piloto da fase 10 é criado por `make package-client`. O arquivo
`dist/compasso-client-0.1.0-pilot-linux-<arquitetura>.tar.gz` contém binários,
interface, unidade systemd, política D-Bus, instalador, desinstalador e a
recuperação automática do login. O instalador configura um cliente novo ou
reaproveita com segurança a configuração root já existente.

Na fase 6, `dbus/br.com.tempo.Agent.conf` autoriza chamadas locais ao método de
bônus e `applications/br.com.tempo.LocalBonus.desktop` adiciona o diálogo GTK ao
menu do Zorin.

Na fase 9, `scripts/install-agent-securely.sh` instala apenas o agente com
configuração e estado pertencentes a root, valida o token no journal e habilita
a unidade endurecida. Ele não altera PAM. O exemplo `cloudflared` publica os
subdomínios HTTPS mantendo a origem HTTP restrita ao loopback do servidor.

Na fase 10, `scripts/install-pilot-components.sh` acrescenta a interface e os
helpers sem ativar o PAM. `tempo-schedule-recovery` deve ser agendado antes de
cada ensaio com bloqueio de login; `tempo-uninstall` restaura primeiro o PAM e
preserva configuração e estado para permitir reinstalação ou diagnóstico.
