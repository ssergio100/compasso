# Pendências — 28/08/2026

## Ícone do cliente Compasso

**Status:** concluído em 28/08/2026.

Substituir o ícone genérico de relógio pelo avatar da raposa no cliente Linux
instalado. A mudança deve contemplar tanto o lançador **Compasso**, usado para
adicionar tempo, quanto a janela de configurações aberta pela engrenagem. A
janela de configurações deve continuar oculta do menu de aplicativos.

Pontos técnicos já identificados:

- os arquivos `.desktop` usam atualmente `Icon=preferences-system-time`;
- o avatar vigente está em
  `admin-ui/src/assets/illustrations/avatars/fox.webp`;
- o pacote Debian precisa instalar uma versão adequada no tema de ícones do
  sistema;
- os identificadores GTK e os arquivos `.desktop` precisam permanecer
  associados para que as duas janelas exibam o ícone corretamente.

Implementação concluída:

- o avatar vigente da raposa foi convertido para PNG RGBA 256×256, com cantos
  arredondados, transparência real e sem borda branca, e passou a ser instalado
  em `usr/share/icons/hicolor/256x256/apps`;
- o lançador principal e a configuração inicial usam
  `Icon=br.com.compasso.Compasso`;
- o identificador GTK da janela principal agora corresponde a
  `br.com.compasso.Compasso.desktop`, enquanto a configuração preserva o par
  `br.com.compasso.AgentSetup`;
- as duas janelas definem explicitamente o ícone da aplicação;
- a configuração continua com `NoDisplay=true` e, portanto, oculta do menu de
  aplicativos;
- o teste do pacote Debian valida o ícone, as associações e a ocultação.

## Mensagens de alerta conforme o motivo do bloqueio

**Status:** concluído em 28/08/2026.

Tornar o motivo do bloqueio explícito no título e no corpo das notificações.
Hoje, rotina e fim de cota compartilham títulos genéricos, como **Bloqueio em 5
minutos**, e somente o corpo diferencia a causa. Isso pode levar a pessoa a
interpretar um aviso de rotina como esgotamento do tempo disponível.

Direção esperada:

- para rotina, informar claramente que uma rotina programada começará;
- para fim de cota, informar claramente que o tempo disponível do dia está
  terminando;
- para bloqueio manual, usar uma mensagem própria caso esse fluxo passe a
  apresentar uma notificação;
- manter o motivo compreensível mesmo quando o ambiente gráfico exibir apenas
  o título ou truncar o corpo da notificação;
- evitar textos técnicos como `quota_exhausted` e preservar linguagem adequada
  ao público familiar.

Antes da implementação, definir os textos finais e cobrir com testes os avisos
principal, de cinco minutos e de um minuto para cada motivo programado.

Textos finais adotados (o aviso principal abaixo considera a configuração de
10 minutos; outro valor é interpolado com a mesma redação):

| Motivo | Marco | Título | Corpo |
| --- | --- | --- | --- |
| Rotina | Principal | Rotina programada em 10 minutos | Uma rotina programada começará em 10 minutos. O computador será bloqueado. |
| Rotina | 5 minutos | Rotina programada em 5 minutos | Uma rotina programada começará em 5 minutos. O computador será bloqueado. |
| Rotina | 1 minuto | Rotina programada em 1 minuto | Uma rotina programada começará em 1 minuto. O computador será bloqueado. |
| Cota | Principal | O tempo de hoje termina em 10 minutos | O tempo disponível de hoje terminará em 10 minutos. O computador será bloqueado. |
| Cota | 5 minutos | O tempo de hoje termina em 5 minutos | O tempo disponível de hoje terminará em 5 minutos. O computador será bloqueado. |
| Cota | 1 minuto | O tempo de hoje termina em 1 minuto | O tempo disponível de hoje terminará em 1 minuto. O computador será bloqueado. |

O bloqueio manual permanece imediato e sem contagem regressiva. O formatador
já reserva para esse motivo o título **Bloqueio solicitado pelo responsável** e
o corpo **O responsável solicitou o bloqueio deste computador.**, evitando que
um identificador técnico seja exibido caso esse fluxo venha a notificar.

Validação concluída:

- testes unitários cobrem os três marcos de rotina e cota, além do texto de
  bloqueio manual;
- toda a suíte Go passou;
- os 22 testes da interface GTK passaram;
- o pacote `compasso-client_0.1.0~pilot29_amd64.deb` foi reconstruído e validado
  sem instalação no sistema local.

## Erro ao desinstalar o cliente

**Status:** pendente.

Durante a desinstalação do cliente, é exibida a mensagem:

> Não foi possível localizar `vr,com.compasso.COmpasso.desltop`

Investigar qual etapa da remoção tenta localizar esse identificador incorreto,
corrigir a referência ao arquivo `.desktop` vigente e cobrir o fluxo de
desinstalação com teste do pacote Debian.
