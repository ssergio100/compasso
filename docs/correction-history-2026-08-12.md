# Histórico de correções — 12/08/2026

Este documento registra os incidentes investigados, suas causas-raiz, as
correções implementadas e as validações realizadas no cliente e no servidor
Compasso em 12 de agosto de 2026.

## 1. Bloqueio antes da configuração inicial

### Sintoma

Ao instalar uma nova versão do cliente antes de concluir o cadastro na API, o
agente bloqueou a sessão por falta de créditos.

### Causa-raiz

O pacote havia preservado `/var/lib/tempo-agent/tempo-agent.db` de uma
instalação anterior. Mesmo sem `server_url`, `device_id` e `device_token`
completos, o daemon abria esse banco e tratava a política antiga como autoridade
offline.

### Correção inicial (`pilot21`)

- O agente sem credenciais completas encerra sem abrir o banco e sem aplicar
  política.
- Durante uma configuração ainda não confirmada, o sincronizador pode se
  comunicar com o servidor, mas o daemon não aplica política.
- Uma instalação Debian nova remove o marcador e o banco residuais.
- Uma atualização normal de uma instalação já confirmada preserva o estado
  offline válido.
- Foram adicionados testes garantindo que um agente não configurado não cria
  nem abre o banco local.

Artefato gerado na etapa: `compasso-client_0.1.0~pilot21_amd64.deb`.

## 2. Exibição incorreta de tempo disponível

### Sintoma

O painel mostrava `0% do tempo disponível` apesar de existir tempo configurado
para o dispositivo.

### Causa-raiz

A API já calculava `remaining_seconds`, mas a interface remota fixava
`bonusMinutes` em zero. Quando o limite diário também era zero, o frontend
calculava o percentual com um total efetivo igual a zero.

### Correção

- A API passou a expor `bonus_seconds` no status do dispositivo.
- O contrato TypeScript passou a exigir esse campo.
- O mapeador remoto converte o bônus para minutos e o inclui no total efetivo.
- Foi adicionado um teste para tempo concedido antes da primeira sincronização.

Durante a investigação também foi confirmado que, naquele momento, os bônus
existentes pertenciam ao perfil **Trabalho**, e não ao perfil **Zorin**.
Posteriormente, o Zorin recebeu uma cota semanal de 18.900 segundos para
quarta-feira, equivalente a 5h15. Essa cota não é um bônus adicional.

## 3. Publicação da API e do painel no Dell

Servidor: `sergio@192.168.18.10` (`srv-dell`).

### Backup

Antes da atualização, API e SQLite Web foram brevemente parados e uma cópia
consistente do volume foi criada em:

```text
/srv/docker/compose/compasso/.deployment-backups/20260812T041724Z/tempo-server-data.tar.gz
```

Também foram preservados os fontes anteriores da API, o `.env` e os arquivos
anteriores da interface administrativa no mesmo diretório de backup.

### API

- Os fontes foram enviados ao diretório
  `/srv/docker/compose/compasso`.
- Foi construída a imagem `compasso-api:0.1.0-pilot21`.
- O container anterior permaneceu disponível para rollback.
- O novo container passou no healthcheck `/healthz`.
- O banco permaneceu no volume externo
  `/srv/docker/volumes/compasso/server`.

### Interface administrativa

- O build estático foi publicado em `/srv/sites/compasso-admin-ui`.
- O `runtime-config.js` existente foi preservado; ele continua apontando para a
  API no mesmo host, porta `8181`.
- API e frontend continuaram em containers independentes.

## 4. Falha de comunicação do novo cadastro

### Sintoma

O agente não conseguia concluir o primeiro heartbeat do perfil Zorin.

### Evidência

O journal registrou:

```text
local revision 6 is newer than server revision 4
```

O perfil Zorin possuía `policy_revision=4`, enquanto o banco local preservado
possuía `policy_revision=6` de outro cadastro. A API rejeitou corretamente o
heartbeat para impedir a mistura de estados entre dispositivos.

Houve também uma falha DNS anterior e transitória, mas ela não era a causa do
erro final: o heartbeat mais recente alcançou a API e foi rejeitado por conflito
de revisão.

## 5. Vínculo permanente entre banco e cadastro (`pilot22`)

Um reset manual recorrente foi considerado inaceitável. A causa foi corrigida
no modelo de persistência do agente.

### Implementação

- A migration local `0004_enrollment.sql` cria uma identidade persistente para
  o cadastro.
- O banco passa a ser vinculado ao par `server_url + device_id`.
- Estado legado sem identidade só é confiável quando a instalação já estava
  confirmada, preservando atualizações normais e o funcionamento offline.
- Em primeiro cadastro, cadastro não confirmado ou troca de dispositivo, o
  agente limpa em uma única transação:
  - política e cotas;
  - rotinas;
  - uso diário;
  - bônus e eventos pendentes;
  - comandos aplicados;
  - saldo de sessão confirmado.
- Após a limpeza, revisões menores do novo servidor podem ser aplicadas
  normalmente.
- No mesmo servidor e dispositivo, reinícios e atualizações preservam todo o
  estado offline.
- A confirmação administrativa continua sendo responsabilidade do configurador
  privilegiado. O daemon não grava em `/etc`, preservando o hardening
  `ProtectSystem=strict`.

### Testes adicionados

- Estado legado não confirmado é removido.
- Uma revisão 4 pode substituir corretamente o estado antigo de revisão 6.
- Atualização confirmada sem identidade anterior preserva a política.
- Reinício com o mesmo vínculo preserva o estado.
- Troca de `device_id` elimina o estado do dispositivo anterior.

## 6. Validação real do Zorin

O `pilot22` foi instalado nesta máquina. No primeiro início, o agente registrou:

```text
previous enrollment state cleared before initial synchronization
synchronization online
```

Depois da confirmação da configuração, o ciclo registrou:

```text
decision=allowed
remaining_seconds=18900
```

O servidor confirmou:

- dispositivo: `Zorin`;
- `policy_revision=4`;
- `applied_policy_revision=4`;
- data local: `2026-08-12`;
- uso: `0` segundos;
- saldo disponível: `18.900` segundos, ou 5h15;
- comunicação e heartbeat ativos.

## 7. Estado final e artefato

Todos os testes Go passaram, incluindo agente, armazenamento, sincronização e
servidor. Os 29 testes do frontend passaram e o build Vite de produção foi
concluído. Os pacotes Debian foram validados sem instalação antes dos ensaios
reais.

Artefato final do cliente:

```text
dist/compasso-client_0.1.0~pilot22_amd64.deb
SHA-256: 2da6085edc4de6ac0d71bcf4deb95f0dfffd5bb39d72db25cb74a90b6bc16c38
```

## 8. Regras preservadas para alterações futuras

- Um agente sem cadastro completo nunca aplica política local.
- Política offline só é válida para o mesmo servidor e dispositivo que a
  originaram.
- Comandos remotos `BLOCK` e `PAUSE` não possuem autoridade offline.
- Receber uma ordem de bloqueio não significa que ela foi aplicada; o servidor
  deve usar o estado real reportado pelo agente.
- Bloqueio usa `loginctl lock-session`, não logout, preservando aplicativos e
  documentos abertos.
- Atualizações do frontend não exigem reiniciar a API quando o contrato JSON não
  muda. Alterações no contrato exigem publicar os dois componentes, ainda que
  permaneçam implantados separadamente.
