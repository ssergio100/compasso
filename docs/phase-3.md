# Fase 3 — daemon systemd e sessões Linux

## Escopo implementado

- executável `tempo-agent` com encerramento limpo por `SIGTERM`/`SIGINT`;
- configuração explícita da conta Linux controlada;
- descoberta de sessões gráficas locais pelo `systemd-logind` via `loginctl`;
- contabilização monotônica somente durante uma sessão permitida;
- checkpoint SQLite e divisão correta do consumo na virada do dia;
- solicitação de logout ordenado ao gerenciador de cada sessão gráfica
  controlada quando cota, rotina ou bloqueio manual impedir o uso;
- a implementação original repetia imediatamente o encerramento para qualquer
  sessão que aparecesse durante o bloqueio; o teste real revelou que isso podia
  interromper a abertura do Plasma e deixar tela preta com cursor;
- a correção aguarda o logind informar estado `active` e mais dez segundos de
  estabilização antes de solicitar logout. KDE e GNOME são acionados por D-Bus
  e não existe fallback para `loginctl terminate-session`;
- unidade systemd sem dependência de rede e com reinício em falha.

O agente e a unidade não contêm domínio, endereço de servidor nem regra de
Cloudflare. A sincronização e sua configuração pertencem às fases 7 e 8.

## Validação automatizada

Execute:

```bash
make test
```

Os testes originais cobrem expiração de cota, início de rotina, pausa antes da
expiração, ausência de sessão gráfica, múltiplas tentativas de sessão durante
bloqueio, virada do dia, atraso do ciclo e falha temporária ao solicitar logout.
O cenário de relogin foi substituído por testes que confirmam que sessões em
estado `opening` não são encerradas e que a estabilização começa somente quando
o estado muda para `active`.

## Validação pendente no Zorin

Os critérios abaixo exigem a máquina cliente real e permanecem abertos como
testes de integração, embora os cenários funcionais de cota, rotina e pausa já
tenham passado nos testes automatizados:

1. confirmar as propriedades reportadas pelo logind para a sessão do Zorin;
2. instalar a política de teste e iniciar o serviço sem rede;
3. matar o processo e confirmar `Restart=on-failure`;
4. expirar uma cota curta e confirmar o encerramento da sessão;
5. iniciar uma rotina curta e confirmar o encerramento da sessão;
6. pausar antes do bloqueio e confirmar que não há logout nem consumo.

A autenticação no greeter permanece permitida. O agente agora espera a nova
sessão gráfica ficar estabelecida antes de solicitar logout. A confirmação do
retorno normal ao greeter, sem tela preta, continua pendente em máquina real.

## Instalação manual provisória

Até a criação do pacote com rollback, os artefatos podem ser preparados com:

```bash
make build-agent
sudo install -m 0755 bin/tempo-agent /usr/sbin/tempo-agent
sudo install -d -m 0755 /etc/tempo-agent
sudo install -m 0600 packaging/config/tempo-agent.toml /etc/tempo-agent/config.toml
sudo install -m 0644 packaging/systemd/tempo-agent.service /etc/systemd/system/tempo-agent.service
```

Edite `controlled_user` antes de executar `systemctl daemon-reload` e habilitar
o serviço. O daemon permanece ativo, mas não aplica bloqueios enquanto o banco
não contiver uma política válida. A carga remota de políticas será adicionada
na sincronização; nos testes automatizados, a política é preparada pelo próprio
teste.
