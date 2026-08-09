# Packaging

Artefatos de instalação do cliente Linux.

- `systemd/tempo-agent.service`: unidade do daemon da fase 3, iniciada sem
  dependência de rede e reiniciada automaticamente em falhas.
- `config/tempo-agent.toml`: configuração-base de instalação; o nome da conta
  controlada deve ser substituído antes de habilitar o serviço.

O gate PAM criado no protótipo da fase 4 permanece desativado e não deve ser
instalado: o requisito atual permite autenticar e aplica logout seguro depois
que a sessão gráfica estiver estabelecida. O helper legado é mantido apenas
para restaurar instalações de ensaio anteriores.
O pacote Debian da fase 10 é criado por `make package-client`. O arquivo
`dist/compasso-client_0.1.0~pilot12_amd64.deb` contém os binários nativos, as
interfaces locais, a unidade systemd, as políticas D-Bus/Polkit e a configuração
gráfica de primeira execução. Suas dependências são declaradas no próprio pacote
para instalação pelo gerenciador gráfico. Docker não integra o pacote e não é
uma dependência do cliente. O tarball legado ainda pode ser criado com
`make package-client-tar`.

O pacote também instala `compasso-session-logout` em `/usr/libexec`. O daemon
o executa no contexto do usuário; ali ele descobre capacidades de logout normal
no D-Bus. O helper não encerra processos e não identifica o desktop por
variáveis de ambiente.

Toda instalação ou atualização para o serviço e o mantém desabilitado, mesmo se
houver credenciais antigas. O atalho **Compasso — Configurar agente** também é
iniciado automaticamente no próximo login enquanto uma nova confirmação estiver
pendente. Após a revisão explícita e a autorização administrativa, o helper
grava a configuração como `0600` e inicia o serviço.

Na fase 6, `dbus/br.com.tempo.Agent.conf` autoriza chamadas locais ao método de
bônus e `applications/br.com.tempo.LocalBonus.desktop` adiciona o diálogo GTK ao
menu do Zorin.

Na fase 9, `scripts/install-agent-securely.sh` instala apenas o agente com
configuração e estado pertencentes a root, valida o token no journal e habilita
a unidade endurecida. Ele não altera PAM. O exemplo `cloudflared` publica os
subdomínios HTTPS mantendo a origem HTTP restrita ao loopback do servidor.

Na fase 10, `scripts/install-pilot-components.sh` acrescenta a interface e os
helpers sem ativar o PAM. `tempo-uninstall` restaura eventual resíduo PAM de
ensaio anterior e preserva configuração e estado para permitir reinstalação ou
diagnóstico.
