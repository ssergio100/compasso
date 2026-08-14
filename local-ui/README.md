# Local UI

Interface Python 3/PyGObject/GTK 4 do cliente Compasso.

## Configuração inicial do agente

`configure_agent.py` solicita a conta Linux controlada, URL HTTPS do servidor,
`device_id` e `device_token`. A janela envia os valores pela entrada padrão do
helper privilegiado, após autorização Polkit; o token não é colocado em
argumentos de processo nem em logs. O pacote inicia essa janela no primeiro
login enquanto `/etc/tempo-agent/setup-complete` não existir.

Depois de validar e gravar `/etc/tempo-agent/config.toml` com modo `0600`, o
helper habilita e inicia `tempo-agent.service`.

## Bônus local

`bonus_dialog.py` é a interface GTK 4 para adicionar 15, 30, 60 ou 120 minutos.
A senha é enviada pelo system D-Bus ao `tempo-agent`; somente o agente root
verifica Argon2id e grava o bônus/evento no SQLite.

Para conferir apenas a interface durante o desenvolvimento:

```bash
python3 bonus_dialog.py
```

Sem o agente instalado no system D-Bus, a janela informa que o serviço não está
disponível.
