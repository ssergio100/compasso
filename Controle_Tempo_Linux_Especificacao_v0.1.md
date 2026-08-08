# Controle de Tempo Linux

*Especificação funcional, arquitetura, tecnologias e plano de implementação*

**Baseline da versão 0.1**

| **Status**                  | Especificação funcional consolidada                                                                   |
|-----------------------------|-------------------------------------------------------------------------------------------------------|
| **Data-base**               | 08 de agosto de 2026                                                                                  |
| **Plataforma-alvo inicial** | Linux desktop com systemd/logind/PAM                                                                  |
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

- Impedir que o usuário controlado simplesmente faça login novamente enquanto a política continuar bloqueando o acesso.

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
| Servidor            | Aplicação central que hospeda a interface web, API, usuários administrativos, configuração e histórico.                                              |
| Cota diária         | Quantidade-base de tempo permitida para um dia da semana.                                                                                            |
| Tempo extra / bônus | Acréscimo pontual ao saldo do dia corrente. Não altera a cota permanente do dia da semana e não ignora rotinas.                                      |
| Rotina              | Janela recorrente de bloqueio, associada a nome, dias da semana, horário inicial e horário final.                                                    |
| Vigilância ativa    | Estado normal: regras são aplicadas e o tempo permitido é contabilizado.                                                                             |
| Vigilância pausada  | Override administrativo: computador liberado, rotinas ignoradas e tempo diário não contabilizado.                                                    |
| Bloqueio manual     | Comando administrativo para bloquear o computador imediatamente, independentemente de cota ou rotina.                                                |
| Revisão de política | Número monotonicamente crescente que identifica a versão da configuração distribuída ao cliente.                                                     |
| Evento local        | Operação originada no cliente, como bônus local, registrada com identificador único e sincronizada depois.                                           |

# 3. Requisitos funcionais detalhados

## 3.1 Cotas diárias

- Cada dia da semana terá uma cota configurável individualmente, incluindo valor zero.

- A cota é uma configuração permanente semanal; o saldo do dia é calculado a partir da cota, bônus e consumo.

- Ao atingir saldo zero, a sessão controlada deve ser encerrada e novos logins devem ser recusados enquanto o bloqueio permanecer válido.

- Uma alteração remota da cota do dia atual entra em vigor assim que for sincronizada, podendo aumentar ou reduzir imediatamente o saldo restante.

## 3.2 Tempo extra / bônus

- Pode ser concedido remotamente pela interface web ou localmente mediante senha do responsável.

- É pontual e associado ao dia corrente; não modifica a cota semanal configurada.

- É somado ao saldo de uso, mas não tem precedência sobre rotinas.

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

4. Saldo diário = cota + bônus - consumo está esgotado?
   SIM -> BLOQUEADO.
   NÃO -> LIBERADO e contabiliza tempo.
```

> **Importante —** Tempo extra altera somente o saldo. A única operação que ignora rotinas e suspende a contabilização é “Pausar vigilância”.

# 5. Arquitetura proposta

```text
                 INTERNET
                    |
                    v
        +-------------------------+
        |      TEMPO SERVER       |
        |     Web + API + banco   |
        +------------+------------+
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
           D-Bus|             | logind / PAM
               v             v
        +-------------+   sessão Linux
        | TEMPO LOCAL |
        | UI + alertas|
        +-------------+
```

A separação de responsabilidades reduz a superfície de ataque e facilita testes:

- O agente é a única autoridade local sobre regras e estado persistente.

- A interface local roda sem privilégios e solicita operações ao agente via D-Bus.

- O servidor é a autoridade de configuração, mas não é necessário para executar a política já sincronizada.

- A camada de enforcement usa mecanismos nativos do Linux (systemd-logind e PAM) em vez de depender apenas de janelas gráficas que o usuário poderia fechar.

# 6. Tecnologias adotadas

| **Área**                   | **Tecnologia**              | **Justificativa**                                                                                                                                       |
|----------------------------|-----------------------------|---------------------------------------------------------------------------------------------------------------------------------------------------------|
| Agente e servidor          | Go 1.26.x                   | Binário simples, boa concorrência, HTTP/TLS maduros, testes rápidos e baixa dependência de runtime. Linha 1.26 é a estável atual em agosto/2026 [T1]. |
| Serviço Linux              | systemd                     | Inicialização no boot, restart automático, logs pelo journal, dependências e hardening de serviço.                                                      |
| Controle de sessão         | systemd-logind + D-Bus      | Permite identificar e terminar sessões de forma integrada ao sistema.                                                                                   |
| Bloqueio de novo login     | PAM                         | Gate de autenticação para impedir relogin do usuário controlado enquanto o agente informar estado bloqueado.                                            |
| Persistência local         | SQLite 3.x                  | Banco autocontido, transacional e robusto para política, consumo, fila de eventos e estado offline. SQLite 3.53.x é a linha atual em 2026 [T2].       |
| Persistência servidor v0.1 | SQLite 3.x                  | Suficiente para poucos dispositivos/administradores e simplifica implantação. Esquema preparado para migração futura para PostgreSQL.                   |
| IPC local                  | D-Bus system bus            | Canal Linux nativo para interface não privilegiada solicitar operações ao daemon com políticas de autorização [T3].                                   |
| UI local                   | Python 3 + PyGObject/GTK 4  | Interface nativa pequena, fácil integração com D-Bus e notificações. GTK 4 é a API atual da família [T4].                                             |
| Web                        | HTML server-side + HTMX 2.x | Sem SPA pesada ou build obrigatório; atualizações parciais e interface responsiva com JavaScript mínimo.                                                |
| CSS/UI web                 | Bootstrap 5.3.x             | Componentes, formulários e layout responsivo estáveis; linha 5.3.8 é a atual no momento da especificação [T5].                                        |
| Senha local                | Argon2id                    | Verificador de senha resistente a ataques offline; somente hash/verificador é sincronizado.                                                             |
| Transporte                 | HTTPS + JSON                | API simples, inspecionável e fácil de testar. Em produção, TLS obrigatório.                                                                             |
| Reverse proxy produção     | Caddy 2 (recomendado)       | Simplifica TLS automático e proxy para o serviço Go; não é dependência lógica do domínio.                                                               |

> **Decisão de simplicidade —** Não usar React, Vue, Angular, Node.js como requisito de runtime nem broker MQTT na v0.1. O heartbeat HTTP é suficiente para o volume e a latência esperada.

# 7. Agente Linux e integração com o sistema

## 7.1 Processo privilegiado

- Executável: `tempo-agent`.

- Usuário: root, iniciado por unidade systemd dedicada.

- Reinício automático em falha e inicialização antes de sessões gráficas normais.

- Diretório de estado sugerido: `/var/lib/tempo-agent/`.

- Configuração de instalação sugerida: `/etc/tempo-agent/`.

- Socket/nome D-Bus de sistema dedicado, por exemplo `br.com.tempo.Agent`.

## 7.2 Motor de política

Será implementado como pacote Go puro, sem dependência de systemd, HTTP ou SQLite. Isso permite milhares de casos de teste em milissegundos e impede que a regra de negócio fique misturada à infraestrutura.

- Entrada: instante atual, dia da semana, estado de vigilância, bloqueio manual, cota, consumo, bônus e rotinas.

- Saída: LIBERADO/BLOQUEADO, motivo, próximo evento previsível e se deve contabilizar tempo.

- Rotinas que atravessam meia-noite devem ser resolvidas no próprio motor.

## 7.3 Logout e prevenção de relogin

- Quando a decisão muda de liberado para bloqueado, o agente solicita ao logind o encerramento da sessão do usuário controlado.

- Um gate PAM consulta um helper local do agente durante nova autenticação. Se a política estiver bloqueando, retorna falha e impede login.

- O gate deve ser instalado com backup e validação da configuração PAM para evitar quebrar o login do sistema inteiro.

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

- [ ] Reiniciar o daemon durante uso e confirmar perda máxima de poucos segundos, não minutos.

- [ ] Simular crash no meio de gravação e verificar banco íntegro.

- [ ] Aplicar nova revisão e confirmar que a política antiga não reaparece após reboot.

- [ ] Criar bônus offline e verificar persistência após reinício.

## Fase 3 — Daemon systemd e sessões Linux

**Entregas**

- Criar `tempo-agent.service`.

- Detectar sessão do usuário controlado via logind.

- Contabilizar somente em estado permitido.

- Aplicar logout por logind.

**Testes de saída da fase**

- [ ] Boot sem rede inicia agente e carrega política local.

- [ ] Matar processo e verificar restart automático pelo systemd.

- [ ] Cota expira durante sessão e a sessão é terminada.

- [ ] Rotina começa durante sessão e a sessão é terminada.

- [ ] Pausa recebida antes do bloqueio impede logout e para contador.

## Fase 4 — Gate PAM contra relogin

**Entregas**

- Implementar helper mínimo de consulta ao estado do agente.

- Integrar pam_exec ou módulo equivalente no fluxo do display manager alvo.

- Criar instalador com backup/reversão segura.

**Testes de saída da fase**

- [ ] Estado liberado permite login do usuário controlado.

- [ ] Estado bloqueado recusa login do usuário controlado.

- [ ] Conta root/responsável não é bloqueada por engano.

- [ ] Agente indisponível segue política fail-safe definida e não quebra permanentemente o PAM.

- [ ] Desinstalação restaura configuração PAM original.

## Fase 5 — Alertas de sessão

**Entregas**

- Criar helper de sessão/GTK que receba eventos do agente.

- Exibir notificação de antecedência e avisos de 5/1 min quando aplicável.

- Adicionar aviso sonoro configurado.

**Testes de saída da fase**

- [ ] Rotina às 22:00 com antecedência 10 min gera alertas em 21:50, 21:55 e 21:59.

- [ ] Fim de saldo estimado dispara os mesmos marcos.

- [ ] Pausar vigilância cancela alertas futuros.

- [ ] Bloquear agora não espera alerta.

## Fase 6 — Interface local e senha

**Entregas**

- Criar diálogo GTK de bônus.

- Implementar Argon2id e rate limit no agente.

- Registrar bônus como evento idempotente.

**Testes de saída da fase**

- [ ] Senha correta adiciona exatamente o tempo solicitado.

- [ ] Senha incorreta não adiciona tempo e aumenta contador de tentativas.

- [ ] Funciona sem Internet.

- [ ] Após reinício, bônus continua aplicado.

- [ ] Bônus não libera uma rotina ativa.

## Fase 7 — Servidor, autenticação e painel web

**Entregas**

- Implementar login administrativo.

- Criar CRUD de dispositivos, cotas, rotinas e senha local.

- Criar dashboard “Agora” e histórico.

- Aplicar CSRF, cookies seguros e sessão expirada.

**Testes de saída da fase**

- [ ] Login correto/incorreto.

- [ ] Alterar cota de terça sem afetar segunda.

- [ ] Criar rotina apenas seg–sex.

- [ ] Rotina 22:00–08:00 preservada corretamente.

- [ ] Troca de senha não revela senha antiga.

- [ ] Ações administrativas aparecem no histórico.

## Fase 8 — Heartbeat e sincronização

**Entregas**

- Implementar autenticação por dispositivo.

- Implementar revisão de política e comandos.

- Implementar upload idempotente de eventos locais.

- Implementar online/offline no painel.

**Testes de saída da fase**

- [ ] Cliente revision 10 recebe revision 11 e aplica imediatamente.

- [ ] Perder Internet por 30 min mantém política e consumo.

- [ ] Bônus local offline é enviado uma única vez ao reconectar.

- [ ] Servidor recebe heartbeat duplicado sem duplicar consumo/eventos.

- [ ] Alterar cota remotamente reduz saldo e bloqueia imediatamente quando o novo saldo fica <= 0.

## Fase 9 — Segurança e hardening

**Entregas**

- Permissões root dos arquivos sensíveis.

- Hardening da unidade systemd.

- TLS de produção e tokens revogáveis.

- Sanitização de logs e auditoria.

**Testes de saída da fase**

- [ ] Usuário comum não consegue alterar SQLite/config do agente.

- [ ] Usuário comum não consegue parar serviço.

- [ ] Token revogado deixa de autenticar.

- [ ] Logs não contêm senha/token.

- [ ] Fuzz/testes de payload inválido não derrubam servidor ou agente.

## Fase 10 — Teste ponta a ponta e piloto

**Entregas**

- Instalar em máquina Linux real usada como cliente.

- Executar cenários de uma semana acelerada com mudanças remotas e offline.

- Validar UX com jogos e avisos.

- Criar pacote de instalação e rollback.

**Testes de saída da fase**

- [ ] Ciclo completo: cota -> alerta -> logout -> bloqueio de login -> bônus -> retorno.

- [ ] Rotina -> alerta -> bloqueio -> fim da rotina -> login permitido com saldo preservado.

- [ ] Pausa -> uso sem consumo -> retomar -> política aplicada imediatamente.

- [ ] Reboot offline mantém restrições.

- [ ] Atualização de senha sincroniza e senha antiga deixa de funcionar.

- [ ] Instalação e desinstalação não deixam o sistema sem login.

# 19. Critérios de aceite da v0.1

- [ ] O computador aplica corretamente cota e rotinas sem depender da Internet.

- [ ] Fim de cota e início de rotina encerram a sessão do usuário controlado.

- [ ] O usuário controlado não consegue relogar enquanto a política bloquear.

- [ ] Cotas são independentes por dia da semana.

- [ ] Rotinas aceitam dias selecionados e intervalos atravessando meia-noite.

- [ ] Tempo extra local e remoto aumenta saldo, mas nunca ignora rotinas.

- [ ] Pausar vigilância libera o uso, ignora rotinas/cota e não desconta tempo.

- [ ] Retomar vigilância aplica imediatamente a situação corrente.

- [ ] Bloqueio manual remoto é efetivo dentro da latência de heartbeat esperada.

- [ ] Alertas previsíveis são exibidos antes do bloqueio.

- [ ] Senha local é definida remotamente, funciona offline e nunca é armazenada em texto puro.

- [ ] Mudanças de política possuem revisão e substituem imediatamente a configuração anterior.

- [ ] Eventos locais offline sincronizam sem duplicação.

- [ ] O painel informa estado online/offline, saldo, próximo bloqueio e histórico essencial.

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
| Saldo zerado                                    | Bloqueado; sessão encerrada; login negado.                            |
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
