# Packaging

Artefatos de instalação do cliente Linux.

- `systemd/tempo-agent.service`: unidade do daemon, iniciada sem
  dependência de rede e reiniciada automaticamente em falhas.
- `config/tempo-agent.toml`: configuração-base de instalação; o nome da conta
  controlada deve ser substituído antes de habilitar o serviço.

O pacote Debian do cliente é criado por `make package-client`. O arquivo em
`dist/compasso-client_<versão>_amd64.deb` contém os binários nativos, as
interfaces locais, a unidade systemd, as políticas D-Bus/Polkit e a configuração
gráfica de primeira execução. Suas dependências são declaradas no próprio pacote
para instalação pelo gerenciador gráfico. Docker não integra o pacote e não é
uma dependência do cliente.

O agente usa `loginctl lock-session` para bloquear a sessão gráfica sem encerrar
aplicativos. A unidade systemd concede apenas os acessos necessários ao banco,
ao logind e à API D-Bus local.

Uma instalação nova mantém o serviço desabilitado até a configuração inicial.
Atualizações preservam e reiniciam clientes já configurados. O assistente é
iniciado automaticamente enquanto uma confirmação estiver pendente; após a
autorização administrativa, grava a configuração como `0600` e inicia o serviço.

`dbus/br.com.tempo.Agent.conf` autoriza chamadas locais ao método de bônus. O
menu publica somente o aplicativo **Compasso**; a configuração avançada é
aberta pela engrenagem da janela principal.
