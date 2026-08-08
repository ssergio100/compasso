# Fase 6 — interface local e senha

## Implementação

- diálogo GTK 4 com períodos de 15, 30, 60 e 120 minutos;
- API `AddLocalBonus` no system D-Bus do agente;
- verificador Argon2id em formato PHC, sem armazenar senha em texto puro;
- rate limit progressivo após senha incorreta;
- bônus e evento `bonus_added` gravados atomicamente com UUID;
- operação totalmente local, sem chamada de rede;
- bônus preservado após reinício e sem precedência sobre rotinas.

## Testes automatizados

No repositório, execute somente:

```bash
make test
```

Esse comando cobre todos os checkboxes automatizados da fase 6, incluindo os
testes Python da lógica da interface. Se passar, responda apenas `make test:
passou`. Em caso de falha, envie somente as últimas linhas do erro.

## Teste visual rápido

**Status:** executado com sucesso no Zorin. A janela abriu, manteve a senha
oculta e desativou a operação ao detectar que o agente D-Bus ainda não estava
instalado.

No Zorin, a partir do diretório `local-ui`, execute:

```bash
python3 bonus_dialog.py
```

Confira:

1. a janela abre com título “Adicionar tempo — Compasso”;
2. aparecem as quatro opções de duração;
3. a senha fica oculta;
4. sem o agente instalado, aparece uma mensagem clara de serviço indisponível.

Responda apenas `visual: passou` ou `visual: falhou — descrição curta`.

## Teste integrado no Zorin

O teste completo exige o agente instalado, uma política com verificador de senha
e a regra D-Bus em `packaging/dbus/br.com.tempo.Agent.conf`. Quando a máquina
estiver preparada, o roteiro humano será:

1. abrir “Adicionar tempo — Compasso” pelo menu;
2. escolher 15 minutos, digitar senha errada e confirmar a mensagem;
3. tentar imediatamente e confirmar o rate limit;
4. aguardar dois segundos, informar a senha correta e confirmar “15 minutos”;
5. reiniciar e confirmar que o bônus permanece;
6. repetir durante uma rotina ativa e confirmar que a rotina continua bloqueando.

Esse teste permanece aberto na especificação até ser executado no Zorin.
