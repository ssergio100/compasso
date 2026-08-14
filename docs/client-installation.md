# Compasso Client — instalação e operação

## Instalação

Instale o pacote na máquina que será controlada:

```bash
sudo apt install ./compasso-client_<versão>_amd64.deb
```

Uma instalação nova mantém `tempo-agent.service` desabilitado. Abra
**Compasso**, clique na engrenagem e informe a conta Linux controlada, o endereço
HTTPS do servidor, `device_id` e `device_token`. O assistente só conclui depois
que o servidor aceitar o primeiro heartbeat.

## Uso

O aplicativo **Compasso** abre a concessão local de tempo. A engrenagem no canto
inferior direito abre novamente a configuração de servidor e credenciais.

O agente roda como serviço systemd e continua aplicando a última autorização
válida durante interrupções de rede:

```bash
systemctl status tempo-agent.service
journalctl -u tempo-agent.service --no-pager -n 50
```

## Atualização e remoção

Atualizações preservam configuração, banco local e estado do serviço:

```bash
sudo apt install ./compasso-client_<nova-versão>_amd64.deb
```

Remover o pacote mantém configuração e estado. Para apagá-los também, use:

```bash
sudo apt purge compasso-client
```
