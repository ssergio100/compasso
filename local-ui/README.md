# Local UI

Interface Python 3/PyGObject/GTK 4 para concessão local de bônus e alertas de sessão.

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
