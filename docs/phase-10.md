# Fase 10 — teste ponta a ponta e piloto

> **Registro histórico:** os ensaios abaixo descrevem versões que encerravam a
> sessão. O agente atual usa `loginctl lock-session`, conforme `agent/README.md`;
> as referências a logout permanecem somente como histórico dos pilotos.

## Implementação

- alertas de cota e rotina são enviados à sessão gráfica do usuário controlado;
- o envio de um alerta possui timeout e nunca interrompe a aplicação da política;
- a semana acelerada cobre as sete cotas diárias, rotina atravessando meia-noite
  e vigilância pausada;
- uma senha nova só substitui a antiga depois de sincronização bem-sucedida;
- o pacote piloto inclui instalação, desinstalação e recuperação temporizada;
- a instalação não ativa o gate PAM; a autenticação continua permitida e o
  bloqueio deve usar logout seguro depois que a sessão estiver estabelecida;
- [x] Como passo intermediário, HTML e CSS deixaram de ser embutidos no binário
  e o recarregamento dos templates externos sem reinício foi validado em teste
  automatizado.
- [x] Painel extraído para uma aplicação independente que consome somente a
  API JSON. Backend e frontend foram construídos em imagens próprias, e a
  reconstrução da interface sem reiniciar a API foi validada. O registro está em
  `docs/admin-frontend-decoupling-plan.md`.

## Ordem segura dos ensaios no Zorin

Os testes são divididos em etapas. Não avance para logout antes de validar
interface, alerta e recuperação. Mantenha **Pausar vigilância** ativo enquanto
instala ou atualiza componentes.

### 1. Instalar o cliente

- [x] Componentes piloto instalados no Zorin e `tempo-agent.service` confirmado como ativo.

```bash
cd /home/sergio/projetos/compasso
make package-client
sudo apt install ./dist/compasso-client_0.1.0~pilot19_amd64.deb
```

Esse passo instala o serviço e a interface GTK sem modificar PAM. Uma instalação
nova permanece parada até o pareamento pelo aplicativo **Compasso**.

### 2. Validar interface e alerta sem risco de bloqueio de login

- [x] Diálogo GTK validado no Zorin com D-Bus disponível, senha incorreta, rate limit e bônus correto de 15 minutos.
- [x] Painel mantém a cota diária fixa e adiciona tempo extra somente ao tempo restante, sem campo separado de bônus (teste automatizado com cota 08:00 e restante 08:10 após adicionar 10 minutos).
- [x] Contadores visuais mostram segundos: tempo restante decresce e tempo usado aumenta quando o cliente está online e a vigilância está contabilizando.
- [x] Endpoint JSON administrativo fornece status vivo para a interface atual e inicia o contrato necessário ao frontend independente.
- [x] Contadores consultam o servidor em segundo plano a cada dois segundos e incorporam tempo extra local sem recarregar a página.
- [x] Tempo extra concedido no agente apareceu automaticamente no painel do Zorin, sem atualização manual da página.
- [x] Uma amostra remota anterior ao contador visual não devolve segundos ao
  saldo (`09:59 → 10:00`); mudanças reais de cota ou bônus e a virada do dia
  continuam aceitas (teste automatizado).
- [x] Heartbeat sem revisão nova não envia nem reaplica saldo; uma alteração
  real de bônus gera nova âncora confirmada (teste integrado automatizado).
- [x] O painel diferencia agente online de sessão gráfica ativa e não anima o
  contador depois do logout (teste HTTP automatizado).
- [x] O daemon sincronizado consome apenas a âncora confirmada e o tempo
  monotônico posterior, sem recalcular a cota ou o bônus a cada segundo (teste
  automatizado).
- [x] Login sem saldo aguarda uma sincronização concluída depois que a sessão
  aparece; tempo recebido mantém a sessão e uma resposta ainda bloqueada libera
  o logout seguro (testes automatizados).
- [x] Mensagens administrativas de sucesso desaparecem automaticamente e não reaparecem ao atualizar a página.
- [x] Alerta visual validado na sessão gráfica do Zorin antes do fim da cota.
- [x] No `pilot11` em Debian 13 KDE, o alerta foi entregue na sessão real antes
  do fim da cota e sua apresentação visual foi aprovada; o teste manual de
  `notify-send` também chegou ao usuário.
- [ ] Acrescentar retorno sonoro confiável; no KDE, a apresentação visual ficou
  boa, mas não houve som mesmo com urgência crítica e dica freedesktop de áudio.
- [x] No ciclo real do `pilot12` com tempo extra, o alerta visual fixo de cinco
  minutos apareceu corretamente; novamente não houve som.
- [x] No mesmo ciclo real, o alerta visual fixo de um minuto apareceu
  corretamente.

Abra **Compasso — Adicionar tempo** pelo menu do Zorin. O ensaio de alerta será
feito com uma cota curta e vigilância retomada, mas o PAM continuará ausente.
Depois que o aviso aparecer, pause novamente a vigilância no painel.

- [x] No Debian 13 KDE, a rotina **Dormir** ativa prevaleceu sobre o tempo extra:
  o login foi permitido e o `pilot12` retornou ao SDDM por logout normal, sem
  tela preta (teste real).
- [ ] Encerrar a rotina **Dormir**, entrar novamente e confirmar que o saldo
  extra permaneceu disponível.
- [x] Tempo extra é crédito do dia local ativo no momento da entrega ao agente;
  o saldo remanescente, incluindo bônus, zera na virada do dia.

### 3. Validar o novo logout seguro

O gate PAM não deve ser instalado. O teste deve permitir a autenticação,
aguardar a sessão gráfica ficar estabelecida e confirmar que o logout retorna
ao greeter sem apresentar tela preta. Antes do ensaio, mantenha disponível uma
TTY ou acesso SSH para parar o agente:

```bash
sudo systemctl stop tempo-agent.service
```

Parar ou desabilitar o agente não deve encerrar a sessão:

```bash
sudo systemctl disable --now tempo-agent.service
```

Somente se a interface gráfica já estiver presa em tela preta, reinicie o
display manager. **Esse comando encerra todas as sessões gráficas abertas e
provoca logout**, portanto não faz parte da simples parada do agente:

```bash
sudo systemctl restart display-manager.service
```

- [x] Recuperação do ensaio com `pilot6` executada: o agente foi desabilitado e
  o reinício explícito do display manager encerrou a sessão, como esperado para
  esse comando administrativo.

O atraso de estabilização passou nos testes automatizados. O teste real de
retorno ao SDDM pelo novo helper passou no Debian 13 KDE, sem tela preta. Ainda
permanece pendente o ensaio ponta a ponta em que o próprio agente detecta o
bloqueio e aciona esse helper, além da validação em GDM.

- [x] `compasso-session-logout` executado no contexto da sessão real encerrou o
  KDE normalmente, retornou ao SDDM e permitiu novo login completo.

### 4. Desinstalação e rollback

```bash
sudo /usr/sbin/tempo-uninstall
```

Se houver resíduo de um ensaio antigo do gate PAM, o desinstalador restaura
primeiro o arquivo original. A configuração e o banco permanecem em `/etc/tempo-agent` e
`/var/lib/tempo-agent`, para diagnóstico e reinstalação.

## Pacote transportável

```bash
make package-deb
```

O `.deb` e seu SHA-256 são gravados em `dist/`. O pacote pode ser transferido
para outra máquina e aberto pelo instalador gráfico. As dependências são
apresentadas pelo gerenciador de pacotes e o agente é instalado como serviço
nativo; Docker não é utilizado no cliente e o gate PAM permanece desativado.

- [x] Candidato `compasso-client_0.1.0~pilot9_amd64.deb` construído e validado
  sem instalação, incluindo o helper de logout e ausência de `libgo`.
- [x] Candidato `compasso-server-0.1.0-pilot4.tar.gz` construído com migração de
  presença gráfica; Compose, ausência de segredos e checksums validados.
- [x] Servidor `pilot4` atualizado no Dell antes do ensaio real do cliente.
- [x] Cliente `pilot9` instalado no Debian 13 KDE e mantido offline até a nova
  confirmação explícita pelo assistente gráfico.
- [x] Falha real do `pilot9` identificada: o hardening `ProcSubset=pid` ocultou
  `/proc/sys/kernel/random/boot_id`, provocando o ciclo de reinício antes da
  sincronização. O artefato foi invalidado.
- [x] Candidato `compasso-client_0.1.0~pilot10_amd64.deb` construído e validado
  sem instalação, mantendo `ProtectProc=invisible` e `ProcSubset=pid`.
- [x] O `pilot10` iniciou mantendo o hardening e alcançou o servidor por HTTPS;
  o journal comprovou a resposta HTTP 401. Ao trocar as credenciais, porém, o
  assistente não reiniciava o processo já ativo e ele continuava usando o token
  anterior. O artefato foi invalidado.
- [x] Instalar e ensaiar o `pilot11`: reinício, confirmação online, contagem e
  alerta passaram, mas a expiração não encerrou a sessão porque o helper não
  reconhecia provedores D-Bus registrados para ativação sob demanda. O
  artefato foi invalidado.
- [x] Candidato `compasso-client_0.1.0~pilot11_amd64.deb` construído e validado
  sem instalação.
- [x] `pilot11` instalado e configurado pela interface no Debian 13 KDE; o
  agente confirmou a primeira sincronização e apareceu online (teste real).
- [x] Ensaio de expiração do `pilot11` chegou a `quota_exhausted`, mas o helper
  retornou código 1 e preservou a sessão aberta; não houve tela preta. O logout
  ponta a ponta permanece reprovado até a causa ser corrigida.
- [x] Descoberta corrigida para aceitar provedores D-Bus ativos ou ativáveis, e
  captura detalhada da saída do helper acrescentada (testes automatizados).
- [x] `busctl --user list` confirmou `org.kde.Shutdown` registrado para ativação
  na sessão KDE real, embora o probe antigo não o reconhecesse antes de haver
  um processo proprietário.
- [x] Instalar o `pilot12` e validar o logout ao fim da cota com o provedor D-Bus
  inicialmente inativo (teste real no Debian 13 KDE).
- [x] Confirmar retorno ao SDDM sem tela preta após o logout do `pilot12` (teste
  real).
- [x] Confirmar novo login normal após o logout do `pilot12`; o tempo extra foi
  aplicado e o segundo ciclo terminou em logout limpo (teste real).
- [x] Candidato `compasso-client_0.1.0~pilot12_amd64.deb` construído e validado
  sem instalação.

O pacote sempre para e desabilita o serviço durante instalação ou atualização,
mesmo quando encontra credenciais deixadas por uma versão anterior. A aplicação
**Compasso — Configurar agente** aparece no menu e abre automaticamente no
próximo login até que o usuário revise os dados e confirme explicitamente. Ela
solicita conta controlada, URL, `device_id` e `device_token`, pede autorização
administrativa e somente então inicia o agente. O seletor pré-seleciona a conta
comum atual e exige confirmação explícita do nome da conta que poderá receber
logout. A janela só conclui depois que o servidor aceitar um heartbeat.

O daemon não escolhe um ambiente gráfico. Um helper dentro da sessão do usuário
descobre no D-Bus uma capacidade de logout normal e usa o adaptador disponível.
Se nenhum adaptador funcionar, o agente mantém a sessão aberta; não há
encerramento abrupto por `loginctl terminate-session`. A validação do retorno ao
greeter em máquina real permanece obrigatória antes de aprovar um novo piloto.
