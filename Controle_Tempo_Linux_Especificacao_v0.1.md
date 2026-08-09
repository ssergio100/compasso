# Controle de Tempo Linux

*Especificação funcional, arquitetura, tecnologias e plano de implementação*

**Baseline da versão 0.1**

| **Status**                  | Especificação funcional consolidada                                                                   |
|-----------------------------|-------------------------------------------------------------------------------------------------------|
| **Data-base**               | 08 de agosto de 2026                                                                                  |
| **Plataforma-alvo inicial** | Linux desktop com systemd/logind                                                                      |
| **Objetivo**                | Controle confiável de cota diária, rotinas e intervenções administrativas, funcionando também offline |

> **Princípio central —** O servidor distribui políticas e recebe telemetria, mas o computador cliente continua capaz de decidir e aplicar todas as restrições com a última configuração válida mesmo sem Internet.

> **Documento de referência:** versão Markdown derivada da especificação v0.1 em DOCX, preparada para leitura, busca e reutilização como contexto técnico.

# Sumário

1. Objetivo e escopo da versão 0.1
2. Conceitos e terminologia
3. Requisitos funcionais detalhados
4. Regras de decisão e precedência
5. Arquitetura proposta
6. Tecnologias adotadas
7. Agente Linux e integração com o sistema
8. Persistência e funcionamento offline
9. Sincronização servidor-cliente
10. Segurança e modelo de ameaça
11. Interface administrativa web
12. Interface local
13. Alertas e experiência em jogos
14. Modelo de dados conceitual
15. API e eventos
16. Observabilidade, auditoria e logs
17. Escopo excluído da v0.1
18. Plano de ação com testes por etapa
19. Critérios de aceite da v0.1
20. Referências técnicas

# 1. Objetivo e escopo da versão 0.1

O projeto tem como objetivo controlar o tempo de utilização de um computador Linux por meio de políticas administradas remotamente, aplicadas por um agente local privilegiado e resistentes à perda temporária de conexão com a Internet.

- Definir cotas diárias independentes para cada dia da semana.

- Definir rotinas recorrentes que bloqueiam o uso em determinados dias e horários.

- Encerrar a sessão do usuário controlado quando uma regra de bloqueio entrar em vigor.

- Permitir que o usuário controlado autentique novamente, mas encerrar de forma
  segura a nova sessão depois que ela estiver estabelecida enquanto a política
  continuar bloqueando o acesso.

- Permitir bloqueio imediato por comando remoto.

- Permitir pausar a vigilância, suspendendo temporariamente todas as restrições e a contabilização do tempo.

- Permitir adicionar tempo extra remotamente e também localmente mediante senha do responsável.

- Alertar o usuário antes de bloqueios previsíveis, especialmente para permitir o encerramento seguro de jogos.

- Continuar funcionando com a última política válida quando o servidor estiver inacessível.

- Registrar eventos relevantes para auditoria e diagnóstico.

> **Definição de “tempo de utilização” na v0.1 —** O contador corre enquanto a sessão gráfica do usuário controlado estiver autenticada e liberada pela política. A v0.1 não depende de detecção de teclado/mouse para decidir se o tempo deve ser descontado. Bloquear a tela ou sair da sessão poderá ser tratado como não utilização conforme a integração final com logind; a regra será validada no protótipo de sessão.

# 2. Conceitos e terminologia

| **Termo**           | **Definição**                                                                                                                                        |
|---------------------|------------------------------------------------------------------------------------------------------------------------------------------------------|
| Agente              | Serviço privilegiado executado no cliente Linux. É a autoridade local para avaliar regras, contabilizar tempo, persistir estado e aplicar bloqueios. |
| Servidor / backend  | Aplicação central que expõe a API, autentica usuários e dispositivos, aplica configurações e persiste o histórico. Não hospeda a interface web.       |
| Frontend administrativo | Aplicação web independente que consome somente a API JSON do servidor e possui build e implantação próprios.                                      |
| Cota diária         | Quantidade-base de tempo permitida para um dia da semana.                                                                                            |
| Tempo extra / bônus | Acréscimo pontual ao tempo restante do dia corrente. Não altera a cota diária configurada e não ignora rotinas.                                        |
| Rotina              | Janela recorrente de bloqueio, associada a nome, dias da semana, horário inicial e horário final.                                                    |
| Vigilância ativa    | Estado normal: regras são aplicadas e o tempo permitido é contabilizado.                                                                             |
| Vigilância pausada  | Override administrativo: computador liberado, rotinas ignoradas e tempo diário não contabilizado.                                                    |
| Bloqueio manual     | Comando administrativo para bloquear o computador imediatamente, independentemente de cota ou rotina.                                                |
| Revisão de política | Número monotonicamente crescente que identifica a versão da configuração distribuída ao cliente.                                                     |
| Evento local        | Operação originada no cliente, como bônus local, registrada com identificador único e sincronizada depois.                                           |

# 3. Requisitos funcionais detalhados

## 3.1 Cotas diárias

- Cada dia da semana terá uma cota configurável individualmente, incluindo valor zero.

- A cota é uma configuração permanente semanal; o tempo restante base é a cota do dia menos o tempo utilizado.

- Ao atingir saldo zero, a sessão controlada deve ser encerrada. Enquanto o
  bloqueio permanecer válido, uma nova autenticação é permitida, mas a sessão
  gráfica estabelecida deve receber logout seguro e retornar normalmente ao
  greeter.

- Uma alteração remota da cota do dia atual entra em vigor assim que for sincronizada, podendo aumentar ou reduzir imediatamente o saldo restante.

## 3.2 Tempo extra / bônus

- Pode ser concedido remotamente pela interface web ou localmente mediante senha do responsável.

- É pontual e associado ao dia corrente; aumenta o tempo restante sem modificar a cota semanal configurada.

- Depois de concedido, é consumido junto com o restante do dia, mas não tem precedência sobre rotinas.

- Se uma rotina começar, o computador bloqueia mesmo que ainda existam minutos de bônus; o saldo volta a estar disponível ao final da rotina.

- Bônus local concedido offline deve ser persistido e posteriormente sincronizado sem ser perdido por uma atualização de política.

## 3.3 Rotinas de bloqueio

- Cada rotina possui nome, dias da semana e intervalo de horário.

- Deve aceitar intervalos que atravessam a meia-noite, por exemplo 22:00–08:00.

- Uma rotina bloqueia independentemente de saldo diário ou bônus.

- Rotinas não consomem o saldo, pois o usuário não está autorizado a utilizar o computador naquele período.

- Exemplos previstos: estudos, almoço, hora de dormir.

## 3.4 Pausar e retomar a vigilância

- “Pausar vigilância” é o mecanismo oficial de liberação administrativa temporária.

- Se o computador estiver bloqueado, a pausa deve permitir o uso assim que o comando for aplicado.

- Durante a pausa, não há desconto da cota diária.

- Durante a pausa, rotinas e saldo esgotado são ignorados.

- O agente permanece executando, persistindo estado e consultando o servidor.

- Ao retomar a vigilância, o agente reavalia imediatamente todas as regras. Se houver rotina ativa ou saldo esgotado, bloqueia sem aguardar novo ciclo.

## 3.5 Bloqueio imediato

- A interface web oferece “Bloquear agora”.

- O comando não precisa respeitar o aviso prévio, pois é uma intervenção explicitamente imediata.

- Se a vigilância estiver pausada e o administrador escolher “Bloquear agora”, o comando encerra a pausa e estabelece o bloqueio manual.

## 3.6 Atualizações de configuração

- O cliente consulta periodicamente o servidor e envia seu estado de utilização.

- Nova política recebida substitui imediatamente a política local anterior.

- Sem Internet, o cliente mantém a última política válida e continua aplicando-a.

- O comportamento offline não pode transformar falha de rede em liberação automática.

## 3.7 Senha para intervenção local

- A senha é cadastrada e alterada somente na interface administrativa remota.

- O servidor não distribui a senha em texto puro; distribui apenas um verificador derivado por Argon2id.

- A última credencial sincronizada continua válida offline.

- A interface local não permite trocar a senha.

- Recuperação “Esqueci a senha” e redefinição por e-mail ficam fora da versão 0.1.

# 4. Regras de decisão e precedência

O motor de política deve ser determinístico. Para uma mesma entrada de estado, data/hora e configuração, deve sempre produzir a mesma decisão.

```text
1. Vigilância pausada?
   SIM -> LIBERADO; não contabiliza tempo; ignora rotinas/cota.
   NÃO -> continuar.

2. Bloqueio manual ativo?
   SIM -> BLOQUEADO.
   NÃO -> continuar.

3. Existe rotina válida neste instante?
   SIM -> BLOQUEADO.
   NÃO -> continuar.

4. Calcular tempo restante base = cota diária - tempo utilizado.
   Somar ao tempo restante os acréscimos de tempo extra concedidos no dia.
   O tempo restante está esgotado?
   SIM -> BLOQUEADO.
   NÃO -> LIBERADO e contabiliza tempo.
```

> **Importante —** Tempo extra altera somente o saldo. A única operação que ignora rotinas e suspende a contabilização é “Pausar vigilância”.

# 5. Arquitetura proposta

```text
        REDE / ACESSO ESCOLHIDO
                    |
          +---------+----------+
          |                    |
          v                    v
 +------------------+  +-------------------------+
 | ADMIN FRONTEND   |  |       TEMPO API         |
 | endereço config. +->| endereço config. + banco|
 +------------------+  +------------+------------+
                                   |
                               HTTPS / JSON
                                   |
                                   v
        +-------------------------+
        |      TEMPO AGENT        |  root
        | motor de política       |
        | contador / SQLite       |
        | sync / enforcement      |
        +------+-------------+----+
               |             |
           D-Bus|             | logind
               v             v
        +-------------+   sessão Linux
        | TEMPO LOCAL |
        | UI + alertas|
        +-------------+
```

A separação de responsabilidades reduz a superfície de ataque e facilita testes:

- O agente é a única autoridade local sobre regras e estado persistente.

- O agente é instalado diretamente no sistema operacional da máquina
  monitorada e executado pelo `systemd`. Ele não roda em Docker e não exige que
  Docker esteja instalado no cliente.

- A interface local roda sem privilégios e solicita operações ao agente via D-Bus.

- O backend é a autoridade de configuração, mas não é necessário para executar a política já sincronizada.

- O frontend administrativo é um artefato independente e se comunica com o backend exclusivamente pela API JSON versionada.

- O backend não renderiza HTML, não serve arquivos do frontend e pode ser compilado, testado e implantado sem eles.

- A camada de enforcement usa `systemd-logind` para identificar sessões e um
  helper executado no D-Bus do usuário para descobrir uma capacidade de logout
  normal. O núcleo não identifica o ambiente gráfico pelo nome e não encerra a
  sessão à força se nenhum adaptador compatível estiver disponível.

# 6. Tecnologias adotadas

| **Área**                   | **Tecnologia**              | **Justificativa**                                                                                                                                       |
|----------------------------|-----------------------------|---------------------------------------------------------------------------------------------------------------------------------------------------------|
| Agente e servidor          | Go 1.26.x                   | Binário simples, boa concorrência, HTTP/TLS maduros, testes rápidos e baixa dependência de runtime. Linha 1.26 é a estável atual em agosto/2026 [T1]. |
| Serviço Linux              | systemd                     | Inicialização no boot, restart automático, logs pelo journal, dependências e hardening de serviço.                                                      |
| Controle de sessão         | systemd-logind              | Identifica a sessão gráfica local e seu estado sem interferir na autenticação.                                                                          |
| Logout de sessão           | D-Bus da sessão gráfica     | Descobre capacidades de logout normal por adaptadores extensíveis, sem condicionar o núcleo ao ambiente gráfico; falhas nunca encerram a sessão abruptamente. |
| Persistência local         | SQLite 3.x                  | Banco autocontido, transacional e robusto para política, consumo, fila de eventos e estado offline. SQLite 3.53.x é a linha atual em 2026 [T2].       |
| Persistência servidor v0.1 | SQLite 3.x                  | Suficiente para poucos dispositivos/administradores e simplifica implantação. Esquema preparado para migração futura para PostgreSQL.                   |
| IPC local                  | D-Bus system bus            | Canal Linux nativo para interface não privilegiada solicitar operações ao daemon com políticas de autorização [T3].                                   |
| UI local                   | Python 3 + PyGObject/GTK 4  | Interface nativa pequena, fácil integração com D-Bus e notificações. GTK 4 é a API atual da família [T4].                                             |
| Frontend administrativo    | HTML, CSS e JavaScript independentes | Aplicação separada do backend; começa simples e pode migrar para React ou outra tecnologia sem alterar o servidor.                              |
| CSS/UI web                 | Bootstrap 5.3.x             | Componentes, formulários e layout responsivo estáveis; linha 5.3.8 é a atual no momento da especificação [T5].                                        |
| Senha local                | Argon2id                    | Verificador de senha resistente a ataques offline; somente hash/verificador é sincronizado.                                                             |
| Transporte                 | HTTPS + JSON                | API simples, inspecionável e fácil de testar. Em produção, TLS obrigatório.                                                                             |
| Reverse proxy produção     | Caddy 2 (recomendado)       | Simplifica TLS automático e proxy para o serviço Go; não é dependência lógica do domínio.                                                               |

> **Decisão de simplicidade —** A independência do frontend não exige adotar React, Vue ou Angular agora. A primeira extração reaproveitará HTML, CSS e JavaScript simples. Node.js não será requisito de runtime do servidor, e o heartbeat HTTP continua suficiente para o volume e a latência esperada.

> **Fronteira de implantação —** Em produção, somente o servidor é executado em
> Docker. O agente e a interface local são componentes nativos da
> máquina monitorada. Um contêiner pode ser usado na máquina de desenvolvimento
> apenas para compilar os binários portáteis; ele não faz parte do pacote nem é
> uma dependência do cliente.

# 7. Agente Linux e integração com o sistema

## 7.1 Processo privilegiado

- Executável: `tempo-agent`.

- Usuário: root, iniciado por unidade systemd dedicada.

- Implantação: binário nativo; Docker é proibido como runtime do agente.

- Reinício automático em falha e inicialização antes de sessões gráficas normais.

- Diretório de estado sugerido: `/var/lib/tempo-agent/`.

- Configuração de instalação sugerida: `/etc/tempo-agent/`.

- Socket/nome D-Bus de sistema dedicado, por exemplo `br.com.tempo.Agent`.

## 7.2 Motor de política

Será implementado como pacote Go puro, sem dependência de systemd, HTTP ou SQLite. Isso permite milhares de casos de teste em milissegundos e impede que a regra de negócio fique misturada à infraestrutura.

- Entrada: instante atual, dia da semana, estado de vigilância, bloqueio manual, cota, consumo, bônus e rotinas.

- Saída: LIBERADO/BLOQUEADO, motivo, próximo evento previsível e se deve contabilizar tempo.

- Rotinas que atravessam meia-noite devem ser resolvidas no próprio motor.

## 7.3 Logout seguro após relogin

- Quando a decisão muda de liberado para bloqueado, o agente delega à sessão do
  usuário uma solicitação de logout normal descoberta por capacidade.

- Uma nova autenticação do usuário controlado não é recusada. Se a política
  continuar bloqueando, o agente espera a sessão gráfica ficar estabelecida e
  então aguarda a primeira sincronização concluída depois desse login. Se a
  resposta trouxer saldo, a sessão permanece; se o estado atualizado continuar
  bloqueado, o agente solicita seu logout.

- Uma falha de conexão não é tratada como resposta do servidor. Enquanto não
  houver heartbeat concluído depois do login, a sessão permanece aberta e o
  agente continua tentando sincronizar.

- O agente não deve encerrar uma sessão durante a transição entre o greeter e
  o desktop. O resultado obrigatório é voltar à tela de login operacional,
  nunca permanecer em tela preta apenas com o cursor.

- O usuário administrativo/root do sistema não será submetido à política da conta controlada.

## 7.4 Contabilização

- O consumo é acumulado em segundos, não em minutos, evitando perda de precisão.

- Usar relógio monotônico para medir intervalos de execução; relógio civil é usado somente para calendário/rotinas.

- Persistir checkpoints frequentes para que crash/reboot não devolvam tempo significativo ao usuário.

- Ao reiniciar, reconstruir o saldo a partir do estado persistido e da data local.

# 8. Persistência e funcionamento offline

O agente deve ser funcionalmente autônomo quando o servidor estiver indisponível.

| **Dado local**         | **Persistência** | **Comportamento offline**                                 |
|------------------------|------------------|-----------------------------------------------------------|
| Última política válida | SQLite           | Continua sendo aplicada integralmente.                    |
| Revisão da política    | SQLite           | Permite saber se o servidor possui atualização posterior. |
| Consumo do dia         | SQLite           | Continua aumentando enquanto o uso estiver permitido.     |
| Bônus locais           | SQLite/eventos   | Continuam válidos e ficam pendentes de upload.            |
| Estado de vigilância   | SQLite           | Mantém o último estado recebido/aplicado.                 |
| Bloqueio manual        | SQLite           | Não desaparece por perda de rede.                         |
| Fila de auditoria      | SQLite           | Eventos são enviados quando a conexão voltar.             |

> **Regra de reconciliação —** Política é substituída por revisão; consumo e ações locais são eventos e não podem ser apagados simplesmente por receber uma nova política.

# 9. Sincronização servidor-cliente

## 9.1 Heartbeat

- Intervalo inicial proposto: 10 segundos quando houver conectividade.

- O mesmo ciclo envia telemetria, eventos pendentes e consulta alteração de política/comandos.

- Falhas usam retry com backoff e limite, sem interromper o motor local.

- O servidor considera o equipamento offline após uma janela configurada, por exemplo 60 segundos sem heartbeat.

## 9.2 Revisões

```text
Servidor possui policy_revision = 41
Cliente informa policy_revision = 39

=> servidor retorna política 41 completa
=> cliente valida, grava transacionalmente e aplica imediatamente
=> cliente passa a informar revision 41
```

## 9.3 Eventos idempotentes

- Cada bônus local ou evento relevante possui UUID.

- O servidor aceita o mesmo UUID mais de uma vez sem duplicar o efeito.

- O cliente remove o evento da fila somente após confirmação do servidor.

## 9.4 Comandos

| **Comando**        | **Efeito**                                                       |
|--------------------|------------------------------------------------------------------|
| `pause_monitoring`   | Pausa vigilância e libera o computador sem contabilizar tempo.   |
| `resume_monitoring`  | Retoma vigilância e reavalia imediatamente todas as regras.      |
| `block_now`          | Encerra eventual pausa, ativa bloqueio manual e aplica logout.   |
| `clear_manual_block` | Remove o bloqueio manual e volta à avaliação normal da política. |
| `add_bonus`          | Adiciona saldo pontual ao dia informado.                         |
| `replace_policy`     | Substitui a política local pela nova revisão.                    |

# 10. Segurança e modelo de ameaça

## 10.1 Premissas

- O usuário controlado é conta comum, sem sudo/root.

- O responsável controla as credenciais administrativas web e a senha de intervenção local.

- O agente e seu banco ficam protegidos por permissões root.

- A comunicação de produção usa HTTPS e autenticação individual do dispositivo.

## 10.2 Credenciais

- Senha administrativa web armazenada por hash seguro; nunca reversível.

- Senha local convertida para Argon2id no servidor; apenas o verificador é enviado ao cliente.

- Troca de senha gera nova revisão de política/segurança e entra em vigor no próximo sync.

- Tentativas locais incorretas devem ter rate limit progressivo e auditoria.

## 10.3 Autenticação do dispositivo

- Cada cliente recebe device_id e segredo/token próprio no pareamento.

- Token fica legível apenas por root no cliente.

- Revogação pelo servidor invalida um equipamento comprometido.

## 10.4 Limites do modelo de ameaça

> **Fora da proteção da v0.1 —** Um usuário com root, acesso administrativo ao boot, Live USB ou capacidade de alterar fisicamente o disco pode remover/contornar qualquer software local. Proteção contra esse cenário exige medidas adicionais de firmware, Secure Boot, criptografia e política física, e não é objetivo da v0.1.

# 11. Interface administrativa web

## 11.1 Princípios de usabilidade

- A tela inicial deve responder rapidamente: “quanto tempo resta?”, “qual o próximo bloqueio?” e “o PC está online?”.

- Ações imediatas ficam visíveis na primeira tela; configurações permanentes ficam em áreas separadas.

- Ações destrutivas ou de grande impacto pedem confirmação clara.

- Não exibir funcionalidades futuras desativadas; o que não existe na v0.1 não aparece.

## 11.2 Dashboard / Agora

```text
PC DO QUARTO                                      ONLINE

TEMPO RESTANTE
01 h 42 min

Usado hoje:       2h18 de 4h00
Próximo bloqueio: 22:00 — Hora de dormir

[ + Tempo extra ]  [ Pausar vigilância ]  [ Bloquear agora ]
```

## 11.3 Cotas

| **Dia** | **Cota diária** |
|---------|-----------------|
| Segunda | hh:mm           |
| Terça   | hh:mm           |
| Quarta  | hh:mm           |
| Quinta  | hh:mm           |
| Sexta   | hh:mm           |
| Sábado  | hh:mm           |
| Domingo | hh:mm           |

## 11.4 Rotinas

- Formulário: nome, dias da semana, horário inicial e horário final.

- Lista resumida das rotinas ativas com edição e exclusão.

- Validar conflitos sem proibir sobreposição: duas rotinas simultâneas continuam resultando em bloqueio.

## 11.5 Segurança local

- Área para definir nova senha local e confirmação.

- Não mostrar a senha atual nem oferecer recuperação na v0.1.

- Após salvar, informar “aguardando sincronização” ou “já sincronizado” conforme estado do cliente.

## 11.6 Histórico

- Mostrar data/hora, origem (Web/Local/Sistema), tipo de evento e detalhes não sensíveis.

- Nunca registrar senha ou token em logs visíveis.

# 12. Interface local

A interface local existe para intervenções excepcionais do responsável sem necessidade de abrir o painel remoto. Ela não é uma segunda área administrativa completa.

```text
Adicionar tempo extra

[ 00 h : 30 min ]
     -5   +5

Senha do responsável
[ ••••••••••••••• ]

[ Cancelar ]  [ Adicionar ]
```

- Seletor de tempo com passos de 5 minutos e entrada clara de horas/minutos.

- Ao confirmar, a UI envia ao agente a duração e a senha digitada; a UI não valida a senha sozinha.

- O agente valida contra o verificador Argon2id armazenado localmente.

- Operação funciona offline e gera evento pendente para sincronização posterior.

- A UI não possui edição de cotas, rotinas, senha, pausa de vigilância ou outras configurações administrativas na v0.1.

> **Após logout —** A primeira versão prioriza a intervenção local enquanto existe sessão gráfica disponível. Integração direta com o greeter/login para abrir o diálogo de bônus após a sessão já ter sido encerrada é uma extensão específica de desktop e fica fora do núcleo da v0.1; o desbloqueio após logout permanece possível pela interface remota.

# 13. Alertas e experiência em jogos

- O administrador configura a antecedência principal do aviso de bloqueio previsível.

- Os alertas são gerados para fim de cota e início de rotina.

- O agente calcula o próximo instante de bloqueio conhecido e agenda avisos.

- Quando aplicável, repetir a notificação nos marcos principal, 5 minutos e 1 minuto antes do bloqueio.

- A notificação deve aparecer sobre a sessão e pode emitir som curto, sem roubar foco de forma destrutiva.

- Bloqueio manual “agora” não precisa respeitar contagem regressiva.

- Se a vigilância estiver pausada, não emitir avisos de bloqueios que não serão aplicados.

- Ao retomar a vigilância, recalcular imediatamente o próximo bloqueio e, se ele já estiver vigente, aplicar o bloqueio.

# 14. Modelo de dados conceitual

| **Entidade**            | **Campos principais / responsabilidade**                                                                  |
|-------------------------|-----------------------------------------------------------------------------------------------------------|
| admin_user              | id, login/email, password_hash, ativo, timestamps                                                         |
| device                  | id, nome, device_token_hash, last_seen_at, `policy_revision`, status                                        |
| weekly_quota            | device_id, weekday, seconds_allowed                                                                       |
| routine                 | id, device_id, nome, start_time, end_time, enabled                                                        |
| routine_day             | routine_id, weekday                                                                                       |
| policy                  | device_id, revision, monitoring_state, manual_block, warning_minutes, local_password_verifier, updated_at |
| daily_usage             | device_id, local_date, seconds_used, last_sync_at                                                         |
| bonus                   | id/uuid, device_id, local_date, seconds, origem, created_at                                               |
| audit_event             | uuid, device_id, kind, origin, payload seguro, created_at                                                 |
| pending_event (cliente) | uuid, kind, payload, created_at, retry_count                                                              |

> **Evolução —** O esquema do servidor deve usar chaves e tipos que permitam migração futura para PostgreSQL sem alterar o contrato da API.

# 15. API e eventos

O frontend administrativo acessa todas as funções exclusivamente por esta API
JSON. O backend não oferece páginas HTML nem depende dos arquivos do frontend.
Quando frontend e API usam origens distintas, o CORS fica restrito à origem
administrativa configurada, com cookie seguro sob HTTPS e proteção CSRF para
operações mutáveis. IPs, hostnames e domínios são definidos pela infraestrutura,
não pelo código da aplicação.

| **Endpoint conceitual**           | **Método** | **Uso**                                                                           |
|-----------------------------------|------------|-----------------------------------------------------------------------------------|
| /api/v1/device/heartbeat          | POST       | Envia estado, consumo, revisão e eventos pendentes; recebe comandos/atualizações. |
| /api/v1/admin/devices             | GET        | Lista equipamentos e estado online/offline.                                       |
| /api/v1/admin/devices/{id}/policy | GET/PUT    | Consulta/atualiza cotas, rotinas, alertas e senha local.                          |
| /api/v1/admin/devices/{id}/bonus  | POST       | Concede tempo extra.                                                              |
| /api/v1/admin/devices/{id}/pause  | POST       | Pausa vigilância.                                                                 |
| /api/v1/admin/devices/{id}/resume | POST       | Retoma vigilância.                                                                |
| /api/v1/admin/devices/{id}/block  | POST       | Bloqueia imediatamente.                                                           |
| /api/v1/admin/devices/{id}/events | GET        | Histórico/auditoria.                                                              |

> **Contrato —** A API real será versionada em /api/v1 desde o início. O desenho exato dos payloads será fechado após o motor de política e o esquema local estarem testados.

# 16. Observabilidade, auditoria e logs

- tempo-agent escreve logs estruturados no journald com nível e event_id.

- Eventos funcionais importantes são persistidos no histórico, não apenas no log técnico.

- Exemplos: política aplicada, bloqueio por rotina, bloqueio por saldo, logout, bônus local/remoto, pausa/retomada, troca de senha local, sync falho/recuperado.

- O painel mostra o último heartbeat e a revisão de política aplicada pelo cliente.

- Logs nunca incluem senha digitada, hash completo quando desnecessário, token do dispositivo ou cookies.

# 17. Escopo excluído da v0.1

- Recuperação “Esqueci a senha” e envio de link/código por e-mail.

- Aplicativo móvel nativo.

- Controle de conteúdo, sites, categorias de jogos ou processos específicos.

- Limite individual por aplicativo/jogo.

- Múltiplos perfis de crianças/usuários compartilhando o mesmo desktop com políticas independentes.

- Push em tempo real por WebSocket/MQTT; a v0.1 usa heartbeat HTTP.

- Proteção contra usuário com root, Live USB ou controle físico/firmware.

- Integração específica com greeter para abrir a UI local depois que a sessão controlada já foi encerrada.

- PostgreSQL obrigatório; SQLite é suficiente na primeira versão.

- Relatórios analíticos avançados e dashboards históricos.

# 18. Plano de ação com testes por etapa

A implementação deve avançar em incrementos pequenos. Cada fase só é considerada concluída quando os testes definidos abaixo passam de forma reproduzível.

## Fase 0 — Repositório, convenções e CI

**Entregas**

- Criar monorepo com /agent, /server, /local-ui, /docs e /packaging.

- Configurar gofmt, go test, lint básico e testes automatizados em CI.

- Definir versionamento, migrações SQLite e configuração de desenvolvimento.

**Testes de saída da fase**

- [x] Compilação limpa em Linux amd64.

- [x] go test ./... sem falhas em checkout limpo.

- [x] Banco de teste cria e aplica todas as migrações do zero.

## Fase 1 — Motor de política puro

**Entregas**

- Implementar tipos de cota, bônus, rotina, vigilância e bloqueio manual.

- Implementar cálculo de saldo e resolução de rotina inclusive atravessando meia-noite.

- Implementar cálculo do próximo bloqueio previsível.

**Testes de saída da fase**

- [x] Segunda com cota 2h e consumo 90min retorna 30min restantes.

- [x] Bônus +30min aumenta saldo, mas rotina ativa continua bloqueando.

- [x] Rotina 22:00–08:00 bloqueia 23:30 e 07:59, libera 08:00.

- [x] Vigilância pausada libera mesmo com cota zero e rotina ativa e retorna contabiliza=false.

- [x] Retomar vigilância em rotina ativa retorna bloqueio imediato.

- [x] Bloqueio manual prevalece sobre cota positiva.

- [x] Matriz de todos os dias da semana, bordas exatas de horário e virada de dia.

## Fase 2 — Estado local e persistência SQLite

**Entregas**

- Persistir política, revisão, consumo, bônus, estado e fila de eventos.

- Usar transações para troca de política.

- Implementar checkpoints do contador.

**Testes de saída da fase**

- [x] Reiniciar o daemon durante uso e confirmar perda máxima de poucos segundos, não minutos.

- [x] Simular crash no meio de gravação e verificar banco íntegro.

- [x] Aplicar nova revisão e confirmar que a política antiga não reaparece após reboot.

- [x] Criar bônus offline e verificar persistência após reinício.

## Fase 3 — Daemon systemd e sessões Linux

**Entregas**

- Criar `tempo-agent.service`.

- Detectar sessão do usuário controlado via logind.

- Contabilizar somente em estado permitido.

- Aplicar logout normal por capacidade publicada na sessão gráfica.

**Testes de saída da fase**

- [ ] Boot sem rede inicia agente e carrega política local.

- [ ] Matar processo e verificar restart automático pelo systemd.

- [x] Cota expira durante sessão e a sessão é terminada (teste automatizado com sessão simulada).

- [x] Rotina começa durante sessão e a sessão é terminada (teste automatizado com sessão simulada).

- [x] Pausa recebida antes do bloqueio impede logout e para contador (teste automatizado).

- [x] O helper descobre capacidades de logout no D-Bus da sessão, sem consultar
  o nome do ambiente gráfico e sem fallback para `loginctl terminate-session`
  (teste automatizado).

## Fase 4 — Logout seguro após relogin

**Entregas**

- Permitir a autenticação mesmo quando a política estiver bloqueando.

- Detectar quando a nova sessão gráfica estiver estabelecida antes de aplicar
  o logout.

- Solicitar um logout que devolva o controle ao greeter disponível sem tela
  preta, sem condicionar a lógica ao nome do display manager.

**Testes de saída da fase**

- [ ] Estado bloqueado permite autenticar e concluir a abertura da sessão gráfica.

- [x] O daemon não encerra uma sessão com estado `opening` e aguarda dez
  segundos de estabilização após o estado `active` (teste automatizado com
  logind simulado).

- [x] O agente delega o logout ao helper neutro da sessão; os adaptadores são
  escolhidos pelas capacidades publicadas e uma falha não provoca encerramento
  abrupto (teste automatizado).

- [x] Debian 13: `systemd-run --user --machine=<conta>@.host` alcançou o D-Bus
  do usuário, listou a capacidade de logout e terminou com código zero, sem
  alterar a sessão (teste real com timeout de segurança).

- [x] Debian 13 KDE: o novo `compasso-session-logout -probe` foi executado pelo
  serviço privilegiado dentro da sessão do usuário, descobriu o adaptador
  `plasma-session` e terminou em 70 ms com código zero, sem efetuar logout.

- [x] Uma sessão nova sem saldo aguarda um heartbeat concluído depois do login;
  permanece aberta se a resposta acrescentar tempo e recebe logout se continuar
  bloqueada (testes automatizados).

- [ ] Depois de estabelecida, a sessão bloqueada recebe logout e retorna ao greeter em máquina real.

- [ ] O greeter permanece operacional, sem tela preta com cursor, depois do
  logout; SDDM no Debian 13 validado, GDM ainda pendente.

- [x] Debian 13 KDE/SDDM: chamada real pelo `compasso-session-logout` encerrou
  normalmente a sessão e permitiu novo login com o desktop completo, sem tela
  preta.

- [ ] Conta root/responsável não tem sua sessão encerrada por engano.

> **Requisito substituído —** O protótipo anterior de gate PAM foi validado
> por testes automatizados, mas não foi ativado pelo pacote piloto. A decisão
> atual permite a autenticação e torna esse gate incompatível com a experiência
> definida para a v0.1.

## Fase 5 — Alertas de sessão

**Entregas**

- Criar helper de sessão/GTK que receba eventos do agente.

- Exibir notificação de antecedência e avisos de 5/1 min quando aplicável.

- Adicionar aviso sonoro configurado.

**Testes de saída da fase**

- [x] Rotina às 22:00 com antecedência 10 min gera alertas em 21:50, 21:55 e 21:59 (teste automatizado).

- [x] Fim de saldo estimado dispara os mesmos marcos (teste automatizado).

- [x] Pausar vigilância cancela alertas futuros (teste automatizado).

- [x] Bloquear agora não espera alerta (teste automatizado).

- [x] Implementar helper de cálculo de alertas e validar em testes automatizados.

## Fase 6 — Interface local e senha

**Entregas**

- Criar diálogo GTK de bônus.

- Implementar Argon2id e rate limit no agente.

- Registrar bônus como evento idempotente.

**Testes de saída da fase**

- [x] Senha correta adiciona exatamente o tempo solicitado (teste automatizado).

- [x] Senha incorreta não adiciona tempo e aumenta contador de tentativas (teste automatizado).

- [x] Funciona sem Internet, usando somente política e SQLite locais (teste automatizado).

- [x] Após reinício, bônus continua aplicado (teste automatizado com reabertura do SQLite).

- [x] Bônus não libera uma rotina ativa (teste automatizado).

- [x] No Zorin, o diálogo GTK abre, oculta a senha e sinaliza corretamente que o agente está indisponível.

- [x] No Zorin, o diálogo GTK adiciona bônus pelo D-Bus e apresenta sucesso, senha incorreta e rate limit.

## Fase 7 — Servidor, autenticação e painel web

**Entregas**

- Implementar login administrativo.

- Criar CRUD de dispositivos, cotas, rotinas e senha local.

- Criar dashboard “Agora” e histórico.

- Aplicar CSRF, cookies seguros e sessão expirada.

**Testes de saída da fase**

- [x] Login correto/incorreto (teste HTTP automatizado).

- [x] Alterar cota de terça sem afetar segunda (teste automatizado de persistência e HTTP).

- [x] Criar rotina apenas seg–sex (teste automatizado).

- [x] Rotina 22:00–08:00 preservada corretamente (teste automatizado).

- [x] Troca de senha não revela senha ou verificador na interface/histórico (teste automatizado).

- [x] Ações administrativas aparecem no histórico (teste automatizado).

- [x] No navegador, o fluxo visual de login, dispositivo, cotas, rotina, senha e histórico é aprovado.

## Fase 8 — Heartbeat e sincronização

**Entregas**

- Implementar autenticação por dispositivo.

- Implementar revisão de política e comandos.

- Implementar upload idempotente de eventos locais.

- Implementar online/offline no painel.

**Testes de saída da fase**

- [x] Cliente revision 10 recebe revision 11 e aplica imediatamente (teste integrado automatizado).

- [x] Perder Internet por 30 min mantém política e consumo (teste integrado automatizado).

- [x] Bônus local offline é enviado uma única vez ao reconectar (teste integrado automatizado).

- [x] Servidor recebe heartbeat duplicado sem duplicar consumo/eventos (teste automatizado de persistência).

- [x] Alterar cota remotamente reduz saldo e bloqueia imediatamente quando o novo saldo fica <= 0 (teste integrado automatizado).

- [x] No Zorin, o pareamento e as transições online, offline e reconectado são aprovados no painel.

- [x] Fluxo agente–servidor corrigido: o servidor entrega uma âncora de saldo na
  autorização da sessão; o agente desconta somente uso posterior e heartbeats
  sem mudança não reinicializam o contador (testes integrados automatizados).

- [x] Log real do `pilot6` confirmou que o painel confundia agente online com
  sessão gráfica ativa: após o encerramento da sessão, o servidor ainda animava
  o contador.

- [x] O heartbeat agora transporta presença da sessão gráfica separadamente do
  estado online; o painel só anima os contadores quando ambos estiverem ativos
  (testes HTTP e de persistência automatizados).

- [x] Bônus remoto cria nova revisão e nova âncora; bônus local concedido durante
  um heartbeat em andamento não é apagado pela resposta (testes integrados e de
  persistência automatizados).

- [x] A autorização combina um namespace privado do ciclo do serviço com a
  sessão do logind, impedindo que um identificador reutilizado após reboot ou
  parada explícita herde saldo autorizado para uma sessão anterior (teste
  automatizado da identidade).

## Fase 9 — Segurança e hardening

**Entregas**

- Permissões root dos arquivos sensíveis.

- Hardening da unidade systemd.

- TLS de produção e tokens revogáveis.

- Sanitização de logs e auditoria.

**Testes de saída da fase**

- [x] Usuário comum não consegue alterar SQLite/config do agente (teste real no Zorin após instalação root).

- [x] Usuário comum não consegue parar serviço (teste real no Zorin; systemd exigiu autenticação e o serviço permaneceu ativo).

- [x] Token revogado deixa de autenticar (teste automatizado de persistência e autenticação).

- [x] Logs não contêm senha/token (testes automatizados de sanitização e auditoria).

- [x] Fuzz/testes de payload inválido não derrubam servidor ou agente (2.016.257 execuções de fuzz no Zorin, além do corpus automatizado).

## Fase 10 — Teste ponta a ponta e piloto

**Entregas**

- Instalar em máquinas Linux reais com GNOME/GDM e KDE/SDDM.

- Executar cenários de uma semana acelerada com mudanças remotas e offline.

- Validar UX com jogos e avisos.

- Criar pacote de instalação e rollback sem compilação ou edição manual no
  cliente. Dependências ausentes devem ser apresentadas e instaladas pelo
  gerenciador de pacotes somente após autorização administrativa.

- Manter o cliente independente de uma distribuição ou ambiente gráfico
  específico dentro do escopo systemd e logind. O plano incremental está
  em `docs/portable-client-plan.md`.

- Separar completamente o frontend administrativo do backend. O plano de
  migração incremental está em `docs/admin-frontend-decoupling-plan.md`.

- Distribuir servidor e frontend em um único pacote Docker Compose, mantendo
  API e frontend em contêineres independentes. Túnel, proxy, VPN, DNS, TLS e a
  decisão de expor ou não o servidor pertencem exclusivamente à infraestrutura
  escolhida pelo usuário. O plano está em `docs/server-compose-plan.md`.

**Testes de saída da fase**

- [x] Ciclo real completo no `pilot12`: cota -> alertas -> logout -> retorno ao
  SDDM sem tela preta -> tempo extra -> novo login -> alertas -> segundo logout
  limpo.

- [x] Alerta do `pilot11` entregue na sessão real do Debian 13 KDE antes do fim
  da cota; o canal de notificação e a apresentação visual foram aprovados.

- [ ] Acrescentar som confiável ao alerta; no KDE a apresentação visual foi
  aprovada, mas a dica de áudio enviada pelo agente não produziu som.

- [x] Alerta visual fixo de cinco minutos apareceu no ciclo real do `pilot12`
  com tempo extra; o áudio continuou ausente.

- [x] Alerta visual fixo de um minuto apareceu no ciclo real do `pilot12` com
  tempo extra.

- [x] Saldo confirmado de três segundos é consumido continuamente até zero sem
  consultar a cota de oito horas armazenada localmente (teste automatizado do
  daemon sincronizado).

- [ ] Rotina -> alerta -> bloqueio -> fim da rotina -> login permitido com saldo preservado.

- [x] Durante a rotina **Dormir**, o `pilot12` permitiu autenticar e depois
  executou logout seguro mesmo com tempo extra disponível (teste real).

- [x] Pausa -> uso sem consumo -> retomar -> política aplicada imediatamente (teste real no Zorin).

- [ ] Reboot offline mantém restrições.

- [x] Atualização de senha sincroniza e senha antiga deixa de funcionar (teste integrado automatizado, inclusive falha de sincronização offline).

- [ ] Instalação, logout e desinstalação não deixam o sistema sem greeter operacional.

- [ ] Pacote `.deb` instala no Zorin e Debian 13 KDE sem Go ou `libgo` no cliente.

- [x] Cliente `pilot9` instalado no Debian 13 KDE pela interface; permaneceu
  offline como projetado, sem reutilizar credenciais antigas nem iniciar o
  serviço antes da nova confirmação.

- [x] Início do `pilot9` após a configuração revelou falha antes da
  sincronização: `ProcSubset=pid` ocultava o arquivo `boot_id` lido pelo agente.
  O artefato foi invalidado e a causa reproduzida pelo journal.

- [x] Cliente `pilot10` inicia no Debian 13 KDE mantendo
  `ProtectProc=invisible` e `ProcSubset=pid`, alcança o servidor por HTTPS e
  registra a recusa de credencial como HTTP 401.

- [x] Cliente `pilot10` gerado e validado sem instalação, com namespace privado
  em `RuntimeDirectory`, binários portáteis, dependências e AppStream válidos.

- [x] Reconfiguração do `pilot10` revelou que `enable --now` não reiniciava um
  agente já ativo; o token novo era salvo, mas o processo continuava usando o
  anterior. O artefato foi invalidado.

- [x] Cliente `pilot11` reinicia após reconfiguração e a interface só confirma
  sucesso depois de um heartbeat aceito no Debian 13 KDE (teste real; agente
  online).

- [x] Cliente `pilot11` gerado e validado sem instalação.

- [x] Ensaio real do `pilot11` atingiu o fim da cota, mas o logout normal falhou
  com `orderly logout request failed for session 26: exit status 1`; o agente
  falhou de forma segura e manteve a sessão aberta, sem tela preta.

- [x] A diferença entre o probe anterior, executado depois de uma introspecção,
  e o probe da sessão nova mostrou que a descoberta considerava somente nomes
  D-Bus com proprietário naquele instante; `busctl --user list` confirmou
  `org.kde.Shutdown` registrado para ativação na sessão real.

- [x] Descoberta corrigida para reconhecer serviços D-Bus ativos ou ativáveis,
  mantendo a seleção por capacidade e sem detectar o nome do desktop (teste
  automatizado).

- [x] Cliente `pilot12` encerra a sessão KDE ao fim da cota mesmo quando o
  provedor de logout ainda não está ativo (teste real).

- [x] Após o logout do `pilot12`, o SDDM aparece sem tela preta (teste real).

- [x] Após o logout do `pilot12`, um novo login conclui normalmente; o saldo
  extra é aplicado e termina em novo logout limpo (teste real).

- [x] Cliente `pilot12` gerado e validado sem instalação.

- [x] Configuração inicial do agente é concluída pela interface, sem terminal
  (teste real do `pilot11` no Debian 13 KDE).

- [x] Backend compila e executa sem templates ou arquivos do frontend.

- [x] Frontend administrativo usa somente a API JSON e possui build e
  implantação independentes.

- [x] Alterações visuais são aplicadas sem recompilar ou reiniciar o backend.

- [x] Pacote Docker Compose do servidor instala no Dell Debian 13 sem solicitar
  credenciais; primeiro administrador é configurado depois pelo navegador e o
  painel funciona pela LAN (teste real do `pilot2`).

- [x] Servidor `pilot4` atualizado no Dell pelo procedimento com backup e
  reconstrução do Compose, antes do ensaio do cliente `pilot9` (teste real).

# 19. Critérios de aceite da v0.1

- [ ] O computador aplica corretamente cota e rotinas sem depender da Internet.

- [x] Fim de cota e início de rotina encerram a sessão do usuário controlado
  (testes reais do `pilot12`).

- [x] Ao relogar durante um bloqueio, o usuário entra, recebe logout seguro e
  retorna à tela de login sem tela preta (teste real com a rotina **Dormir**).

- [ ] Cotas são independentes por dia da semana.

- [ ] Rotinas aceitam dias selecionados e intervalos atravessando meia-noite.

- [ ] Tempo extra local e remoto aumenta saldo, mas nunca ignora rotinas.

- [x] Tempo extra disponível não ignorou a rotina **Dormir** ativa (teste real
  do `pilot12`; ainda falta cobrir separadamente as duas origens do bônus).

- [x] Registrar novo requisito: tempo extra remanescente ou não utilizado deve
  ser zerado em alguma circunstância de expiração.

- [ ] Definir em discussão futura qual evento zera o tempo extra remanescente;
  nenhuma regra de expiração deve ser implementada antes dessa decisão.

- [x] Pausar vigilância libera o uso, ignora rotinas/cota e não desconta tempo (teste real no Zorin).

- [x] Retomar vigilância aplica imediatamente a situação corrente (teste real no Zorin).

- [ ] Bloqueio manual remoto é efetivo dentro da latência de heartbeat esperada.

- [x] Alertas previsíveis são exibidos antes do bloqueio (teste real de fim de cota no Zorin).

- [ ] Senha local é definida remotamente, funciona offline e nunca é armazenada em texto puro.

- [ ] Mudanças de política possuem revisão e substituem imediatamente a configuração anterior.

- [ ] Eventos locais offline sincronizam sem duplicação.

- [x] O painel informa estado online/offline, saldo, próximo bloqueio e histórico essencial (testes reais das fases 7, 8 e 10).

- [ ] Reboot, crash do agente e perda de Internet não devolvem controle indevido ao usuário.

- [ ] Instalação/remoção preservam a capacidade de autenticação do sistema Linux.

# 20. Referências técnicas

As escolhas tecnológicas foram conferidas contra documentação oficial disponível em agosto de 2026. As versões exatas de dependências deverão ser fixadas no lock/build do projeto e atualizadas por patch de segurança sem alterar o desenho funcional.

| **Referência**             | **Uso no projeto**                                                                                     |
|----------------------------|--------------------------------------------------------------------------------------------------------|
| [T1] Go Release History  | go.dev/doc/devel/release — Go 1.26 é a linha estável; 1.26.5 publicada em 07/07/2026.                  |
| [T2] SQLite              | sqlite.org — motor autocontido; série 3.53.x atual em julho/agosto de 2026.                            |
| [T3] D-Bus Specification | dbus.freedesktop.org/doc/dbus-specification.html — IPC de sistema e sessão com políticas de segurança. |
| [T4] GTK 4               | docs.gtk.org/gtk4/ — API GTK 4 e integração com plataformas Linux/X11/Wayland.                         |
| [T5] Bootstrap           | getbootstrap.com/docs/5.3/ — linha 5.3.x; documentação identifica 5.3.8 como versão corrente.          |
| [T6] HTMX                | htmx.org — utilizar linha estável 2.x; não depender da linha 4 beta na v0.1.                           |

# Apêndice A — Cenários funcionais de referência

| **Cenário**                                     | **Resultado esperado**                                                |
|-------------------------------------------------|-----------------------------------------------------------------------|
| Saldo disponível, sem rotina, vigilância ativa  | Liberado; contador corre.                                             |
| Saldo zerado                                    | Bloqueado; relogin permitido; nova sessão recebe logout seguro e retorna ao greeter. |
| Saldo zerado + bônus 30 min                     | Liberado por até 30 min, desde que não exista rotina.                 |
| Bônus disponível + rotina ativa                 | Bloqueado; bônus permanece para depois.                               |
| Rotina ativa + vigilância pausada               | Liberado; contador parado.                                            |
| Saldo zerado + vigilância pausada               | Liberado; contador parado.                                            |
| Retomar vigilância durante rotina               | Bloqueio imediato.                                                    |
| Retomar vigilância com saldo zerado             | Bloqueio imediato.                                                    |
| Sem Internet                                    | Última política continua valendo; eventos ficam em fila.              |
| Nova cota remota menor que consumo atual        | Ao sincronizar, saldo <= 0 e bloqueio imediato.                      |
| Troca remota da senha local com cliente offline | Senha antiga vale até a próxima sincronização; depois somente a nova. |
| Bloquear agora durante vigilância pausada       | Pausa é encerrada e bloqueio manual aplicado.                         |
| Fim de rotina com saldo ainda positivo          | Login volta a ser permitido e saldo anterior continua disponível.     |
| Rotina 22:00–08:00 às 00:30                     | Bloqueado pela mesma rotina iniciada no dia anterior.                 |

> **Baseline v0.1 —** Este documento é a referência para implementação. Mudanças funcionais posteriores devem alterar a versão/revisão da especificação e incluir novos testes de aceite correspondentes.
