# Local UI

Interface Python 3/PyGObject/GTK 4 para concessão local de bônus e alertas de sessão.

## Configuração inicial do agente

`configure_agent.py` solicita a conta Linux controlada, URL HTTPS do servidor,
`device_id` e `device_token`. A janela envia os valores pela entrada padrão do
helper privilegiado, após autorização Polkit; o token não é colocado em
argumentos de processo nem em logs. O pacote inicia essa janela no primeiro
login enquanto `/etc/tempo-agent/setup-complete` não existir.

Depois de validar e gravar `/etc/tempo-agent/config.toml` com modo `0600`, o
helper habilita e inicia `tempo-agent.service`.

## Fase 5 — Alertas de sessão

Este diretório contém um helper básico de eventos para o agente local. Em produção,
este helper deve rodar na sessão gráfica do usuário controlado e exibir notificações
visuais e sonoras.

### Uso

O helper lê eventos JSON da entrada padrão:

```bash
cat event.json | python3 alert_helper.py
```

Exemplo de evento:

```json
{
  "title": "Bloqueio em 5 minutos",
  "body": "O bloqueio por rotina ocorrerá em 5 minutos.",
  "sound_enabled": true
}
```

## Fase 6 — Bônus local

`bonus_dialog.py` é a interface GTK 4 para adicionar 15, 30, 60 ou 120 minutos.
A senha é enviada pelo system D-Bus ao `tempo-agent`; somente o agente root
verifica Argon2id e grava o bônus/evento no SQLite.

Para conferir apenas a interface durante o desenvolvimento:

```bash
python3 bonus_dialog.py
```

Sem o agente instalado no system D-Bus, a janela informa que o serviço não está
disponível. O teste integrado no Zorin está descrito em `docs/phase-6.md`.
